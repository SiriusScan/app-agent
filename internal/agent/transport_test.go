package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	pb "github.com/SiriusScan/app-agent/proto/hello"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// All keys and certificates are synthetic, short-lived, and removed by t.TempDir.
// Tests bind only ephemeral loopback ports and never contact real agents/services.
func syntheticTLSFiles(t *testing.T, expired, includeIP bool) (certPath, keyPath, caPath string) {
	t.Helper()
	caPublic, caKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Synthetic test CA"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, caPublic, caKey)
	if err != nil {
		t.Fatal(err)
	}
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(2), DNSNames: []string{"engine.test"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if includeIP {
		leaf.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	}
	if expired {
		leaf.NotAfter = now.Add(-time.Minute)
	}
	der, err := x509.CreateCertificate(rand.Reader, leaf, ca, public, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	write := func(name, typ string, der []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	return write("server.pem", "CERTIFICATE", der), write("server.key", "PRIVATE KEY", keyDER), write("ca.pem", "CERTIFICATE", caDER)
}

type tlsTestService struct {
	pb.UnimplementedHelloServiceServer
	calls      atomic.Int32
	tlsVersion atomic.Uint32
}

func (s *tlsTestService) ConnectStream(stream pb.HelloService_ConnectStreamServer) error {
	s.calls.Add(1)
	if p, ok := peer.FromContext(stream.Context()); ok {
		if info, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			s.tlsVersion.Store(uint32(info.State.Version))
		}
	}
	msg, err := stream.Recv()
	if err != nil {
		return err
	}
	md, _ := metadata.FromIncomingContext(stream.Context())
	if msg.AuthToken != "synthetic-token" || len(md.Get("agent_id")) != 1 {
		return stream.Send(&pb.ServerMessage{AuthToken: "invalid-test-payload"})
	}
	return stream.Send(&pb.ServerMessage{AuthToken: "synthetic-response"})
}

func TestGRPCTransportServerAuthentication(t *testing.T) {
	certFile, keyFile, caFile := syntheticTLSFiles(t, false, true)
	dnsCert, dnsKey, dnsCA := syntheticTLSFiles(t, false, false)
	expiredCert, expiredKey, expiredCA := syntheticTLSFiles(t, true, true)
	_, _, unrelatedCA := syntheticTLSFiles(t, false, true)
	cases := []struct {
		name                                                    string
		certFile, keyFile, caFile, serverName                   string
		plaintextServer, insecureClient, legacyServer, succeeds bool
	}{
		{name: "default TLS verifies target IP", certFile: certFile, keyFile: keyFile, caFile: caFile, succeeds: true},
		{name: "explicit DNS identity", certFile: dnsCert, keyFile: dnsKey, caFile: dnsCA, serverName: "engine.test", succeeds: true},
		{name: "wrong hostname", certFile: certFile, keyFile: keyFile, caFile: caFile, serverName: "impostor.test"},
		{name: "missing IP SAN", certFile: dnsCert, keyFile: dnsKey, caFile: dnsCA},
		{name: "untrusted CA", certFile: certFile, keyFile: keyFile, caFile: unrelatedCA},
		{name: "system roots reject test CA", certFile: certFile, keyFile: keyFile},
		{name: "expired certificate", certFile: expiredCert, keyFile: expiredKey, caFile: expiredCA},
		{name: "no downgrade to plaintext", caFile: caFile, plaintextServer: true},
		{name: "no downgrade to TLS 1.1", certFile: certFile, keyFile: keyFile, caFile: caFile, legacyServer: true},
		{name: "explicit loopback development", plaintextServer: true, insecureClient: true, succeeds: true},
		{name: "TLS server rejects plaintext", certFile: certFile, keyFile: keyFile, insecureClient: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = listener.Close() })
			scfg := &config.ServerConfig{Address: listener.Addr().String(), TLSCertFile: tc.certFile, TLSKeyFile: tc.keyFile}
			if tc.plaintextServer {
				scfg.GRPCTransport = config.GRPCTransportInsecureLoopback
			}
			creds, err := scfg.GRPCTransportCredentials()
			if err != nil {
				t.Fatal(err)
			}
			if tc.legacyServer {
				cert, err := tls.LoadX509KeyPair(tc.certFile, tc.keyFile)
				if err != nil {
					t.Fatal(err)
				}
				// Deliberately obsolete test peer, never used by production transport.
				creds = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11, Certificates: []tls.Certificate{cert}})
			}
			server := grpc.NewServer(grpc.Creds(creds))
			service := &tlsTestService{}
			pb.RegisterHelloServiceServer(server, service)
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(server.Stop)

			cfg := &config.AgentConfig{ServerAddress: listener.Addr().String(), TLSCAFile: tc.caFile, TLSServerName: tc.serverName}
			if tc.insecureClient {
				cfg.GRPCTransport = config.GRPCTransportInsecureLoopback
			}
			transport := newGRPCTransport(zap.NewNop())
			if err := transport.Connect(cfg); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = transport.Close() })
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			stream, rpcErr := transport.OpenStream(ctx, map[string]string{"agent_id": "synthetic-agent"})
			if rpcErr == nil {
				rpcErr = transport.Send(&pb.AgentMessage{AuthToken: "synthetic-token"})
				if rpcErr == nil {
					var msg *pb.ServerMessage
					msg, rpcErr = stream.Recv()
					if rpcErr == nil && msg.AuthToken != "synthetic-response" {
						t.Fatal("test request not received intact")
					}
				}
			}
			if (rpcErr == nil) != tc.succeeds {
				t.Fatalf("success=%v, want %v: %v", rpcErr == nil, tc.succeeds, rpcErr)
			}
			if !tc.succeeds && service.calls.Load() != 0 {
				t.Fatal("unauthenticated peer received gRPC metadata or token")
			}
			if tc.succeeds && service.calls.Load() != 1 {
				t.Fatal("authenticated request did not reach server")
			}
			if tc.succeeds && !tc.plaintextServer && service.tlsVersion.Load() < tls.VersionTLS12 {
				t.Fatal("TLS below 1.2 negotiated")
			}
		})
	}
}

func TestGRPCTransportRejectsConfigurationBeforeConnecting(t *testing.T) {
	transport := newGRPCTransport(zap.NewNop())
	cfg := &config.AgentConfig{ServerAddress: "192.0.2.1:50051", GRPCTransport: config.GRPCTransportInsecureLoopback}
	if err := transport.Connect(cfg); err == nil {
		t.Fatal("remote plaintext must fail before connecting")
	}
	if transport.conn != nil || transport.client != nil {
		t.Fatal("client initialized with invalid transport")
	}
}

func TestGRPCTransportLoopbackIgnoresHostileProxy(t *testing.T) {
	if os.Getenv("SIRIUS_TEST_PROXY_CHILD") != "1" {
		var proxyCalls atomic.Int32
		proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			proxyCalls.Add(1)
			http.Error(w, "test proxy must never receive this connection", http.StatusBadGateway)
		}))
		defer proxy.Close()
		// gRPC caches proxy environment settings. Use a fresh test process so
		// the hostile proxy applies before any gRPC client has been created.
		cmd := exec.Command(os.Args[0], "-test.run=^TestGRPCTransportLoopbackIgnoresHostileProxy$")
		cmd.Env = append(os.Environ(), "SIRIUS_TEST_PROXY_CHILD=1",
			"HTTPS_PROXY="+proxy.URL, "https_proxy="+proxy.URL,
			"HTTP_PROXY="+proxy.URL, "http_proxy="+proxy.URL,
			"ALL_PROXY="+proxy.URL, "all_proxy="+proxy.URL,
			"NO_PROXY=", "no_proxy=")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("plaintext loopback failed with hostile proxy: %v\n%s", err, output)
		}
		if proxyCalls.Load() != 0 {
			t.Fatal("plaintext loopback traffic was sent to a proxy")
		}
		return
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	server := grpc.NewServer()
	service := &tlsTestService{}
	pb.RegisterHelloServiceServer(server, service)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()

	transport := newGRPCTransport(zap.NewNop())
	cfg := &config.AgentConfig{ServerAddress: listener.Addr().String(), GRPCTransport: config.GRPCTransportInsecureLoopback}
	if err := transport.Connect(cfg); err != nil {
		t.Fatal(err)
	}
	defer transport.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := transport.OpenStream(ctx, map[string]string{"agent_id": "synthetic-agent"})
	if err != nil {
		t.Fatal(err)
	}
	if err := transport.Send(&pb.AgentMessage{AuthToken: "synthetic-token"}); err != nil {
		t.Fatal(err)
	}
	msg, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if msg.AuthToken != "synthetic-response" || service.calls.Load() != 1 {
		t.Fatal("local server did not receive request")
	}
}
