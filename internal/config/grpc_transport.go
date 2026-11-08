package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	GRPCTransportTLS = "tls"
	// GRPCTransportInsecureLoopback is only for isolated local development.
	// It cannot be used for a network listener or remote server address.
	GRPCTransportInsecureLoopback = "insecure-loopback"
)

func grpcTransportMode(mode string) (string, error) {
	switch mode {
	case "", GRPCTransportTLS:
		return GRPCTransportTLS, nil
	case GRPCTransportInsecureLoopback:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid GRPC_TRANSPORT %q: use tls (default) or insecure-loopback for local development", mode)
	}
}

// requireLoopbackAddress deliberately accepts only literal IP addresses. DNS
// aliases (including localhost), wildcard binds and gRPC resolver targets could
// resolve outside loopback, so none may opt out of server authentication.
func requireLoopbackAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("insecure-loopback requires a literal loopback IP:port: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("insecure-loopback requires a literal loopback IP, such as 127.0.0.1 or [::1]")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 0 || n > 65535 {
		return fmt.Errorf("insecure-loopback requires a numeric TCP port between 0 and 65535")
	}
	return nil
}

// GRPCTransportCredentials authenticates the server to agents. A zero-value
// configuration requires TLS. Missing or invalid files always fail closed.
func (c *ServerConfig) GRPCTransportCredentials() (credentials.TransportCredentials, error) {
	if c == nil {
		return nil, fmt.Errorf("server configuration is required")
	}
	mode, err := grpcTransportMode(c.GRPCTransport)
	if err != nil {
		return nil, err
	}
	if mode == GRPCTransportInsecureLoopback {
		if c.TLSCertFile != "" || c.TLSKeyFile != "" {
			return nil, fmt.Errorf("insecure-loopback cannot be combined with GRPC_TLS_CERT_FILE or GRPC_TLS_KEY_FILE")
		}
		if err := requireLoopbackAddress(c.Address); err != nil {
			return nil, err
		}
		return insecure.NewCredentials(), nil
	}
	if c.TLSCertFile == "" || c.TLSKeyFile == "" {
		return nil, fmt.Errorf("TLS is required: set both GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE (see documentation/grpc-tls.md)")
	}
	cert, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load gRPC server TLS certificate and key: %w", err)
	}
	return credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}), nil
}

// GRPCTransportCredentials verifies both the trust chain and server identity
// before any gRPC metadata, enrollment token or command is exchanged. With no
// explicit CA bundle, Go uses the operating system roots. A configured bundle
// replaces those roots, restricting trust to the operator-provided CA(s).
func (c *AgentConfig) GRPCTransportCredentials() (credentials.TransportCredentials, error) {
	if c == nil {
		return nil, fmt.Errorf("agent configuration is required")
	}
	mode, err := grpcTransportMode(c.GRPCTransport)
	if err != nil {
		return nil, err
	}
	if mode == GRPCTransportInsecureLoopback {
		if c.TLSCAFile != "" || c.TLSServerName != "" {
			return nil, fmt.Errorf("insecure-loopback cannot be combined with GRPC_TLS_CA_FILE or GRPC_TLS_SERVER_NAME")
		}
		if err := requireLoopbackAddress(c.ServerAddress); err != nil {
			return nil, err
		}
		return insecure.NewCredentials(), nil
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: c.TLSServerName,
	}
	if c.TLSCAFile != "" {
		pem, err := os.ReadFile(c.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("read GRPC_TLS_CA_FILE: %w", err)
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("GRPC_TLS_CA_FILE contains no valid PEM certificates")
		}
		tlsConfig.RootCAs = roots
	}
	return credentials.NewTLS(tlsConfig), nil
}
