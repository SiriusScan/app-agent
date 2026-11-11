package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SiriusScan/app-agent/enrollment"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	pb "github.com/SiriusScan/app-agent/proto/hello"
	goapistore "github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"
)

// Synthetic credentials only. The embedded interface makes unexpected store
// operations fail the test rather than contacting an external service.
type agentAuthKV struct {
	goapistore.KVStore
	mu                 sync.Mutex
	value              string
	present            bool
	readErr, createErr error
	commitOnError      bool
	creates, writes    int
	missingRead        func()
}

func (s *agentAuthKV) GetValue(ctx context.Context, _ string) (goapistore.ValkeyResponse, error) {
	s.mu.Lock()
	value, present, err, hook := s.value, s.present, s.readErr, s.missingRead
	s.mu.Unlock()
	if ctx.Err() != nil {
		return goapistore.ValkeyResponse{}, ctx.Err()
	}
	if err != nil {
		return goapistore.ValkeyResponse{}, err
	}
	if !present {
		if hook != nil {
			hook()
		}
		return goapistore.ValkeyResponse{}, goapistore.ErrKeyNotFound
	}
	return goapistore.ValkeyResponse{Message: goapistore.ValkeyValue{Value: value}}, nil
}
func (s *agentAuthKV) SetValue(context.Context, string, string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes++
	return errors.New("unconditional write forbidden")
}
func (s *agentAuthKV) SetValueIfAbsent(ctx context.Context, _ string, value string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creates++
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if s.createErr != nil {
		if s.commitOnError {
			s.value, s.present = value, true
		}
		return false, s.createErr
	}
	if s.present {
		return false, nil
	}
	s.value, s.present = value, true
	return true, nil
}

type agentAuthStream struct {
	pb.HelloService_ConnectStreamServer
	ctx      context.Context
	initial  *pb.AgentMessage
	welcomed []*pb.ServerMessage
}

func (s *agentAuthStream) Context() context.Context { return s.ctx }
func (s *agentAuthStream) Recv() (*pb.AgentMessage, error) {
	if s.initial != nil {
		msg := s.initial
		s.initial = nil
		return msg, nil
	}
	return nil, io.EOF
}
func (s *agentAuthStream) Send(msg *pb.ServerMessage) error {
	s.welcomed = append(s.welcomed, msg)
	return nil
}
func authTestServer(kv goapistore.KVStore) *Server {
	return &Server{logger: zap.NewNop(), kvStore: kv, agents: map[string]pb.HelloService_ConnectStreamServer{}, agentTokenGenerator: func() (string, error) { return "fake-new-token", nil }}
}
func authTestStream(token string) *agentAuthStream {
	return &agentAuthStream{ctx: context.Background(), initial: &pb.AgentMessage{AgentId: "fixture", AuthToken: token}}
}

func TestAgentEnrollmentRejectsBeforeRegisteringStream(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kv       *agentAuthKV
		nilStore bool
		token    string
	}{
		{name: "missing store", nilStore: true},
		{name: "outage", kv: &agentAuthKV{readErr: errors.New("synthetic read failure")}},
		{name: "untyped missing", kv: &agentAuthKV{readErr: errors.New("not found")}},
		{name: "existing", kv: &agentAuthKV{present: true, value: `{"agent_id":"fixture","token":"fake-existing"}`}},
		{name: "corrupt", kv: &agentAuthKV{present: true, value: `corrupt`}},
		{name: "write failure", kv: &agentAuthKV{createErr: errors.New("synthetic write failure")}},
		{name: "wrong token", kv: &agentAuthKV{present: true, value: `{"token":"fake-existing"}`}, token: "fake-wrong"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var kv goapistore.KVStore
			if !tc.nilStore {
				kv = tc.kv
			}
			server := authTestServer(kv)
			stream := authTestStream(tc.token)
			err := server.ConnectStream(stream)
			if err == nil || errors.Is(err, io.EOF) {
				t.Fatalf("authentication accepted: %v", err)
			}
			if len(stream.welcomed) != 0 || len(server.agents) != 0 {
				t.Fatal("rejected stream registered or welcomed")
			}
			if tc.kv != nil && tc.kv.writes != 0 {
				t.Fatal("unconditional write")
			}
			if tc.kv != nil && tc.name != "write failure" && tc.kv.creates != 0 {
				t.Fatal("attempted enrollment without confirmed absence")
			}
		})
	}
}

func TestAgentEnrollmentWelcomeAndReconnect(t *testing.T) {
	kv := &agentAuthKV{}
	server := authTestServer(kv)
	stream := authTestStream("")
	if err := server.ConnectStream(stream); !errors.Is(err, io.EOF) {
		t.Fatalf("expected end of test stream: %v", err)
	}
	if len(stream.welcomed) != 1 || stream.welcomed[0].AuthToken != "fake-new-token" {
		t.Fatal("enrollment token missing from welcome")
	}
	reconnect := authTestStream("fake-new-token")
	if err := server.ConnectStream(reconnect); !errors.Is(err, io.EOF) {
		t.Fatalf("valid reconnect rejected: %v", err)
	}
	if len(reconnect.welcomed) != 1 || reconnect.welcomed[0].AuthToken != "" {
		t.Fatal("reconnect issued a new token")
	}
	if kv.creates != 1 || kv.writes != 0 {
		t.Fatal("reconnect changed token")
	}
}

func TestAgentEnrollmentConcurrentConnectionsHaveOneWinner(t *testing.T) {
	const count = 24
	var reads atomic.Int32
	ready := make(chan struct{})
	kv := &agentAuthKV{missingRead: func() {
		if reads.Add(1) == count {
			close(ready)
		}
		<-ready
	}}
	server := authTestServer(kv)
	var sequence atomic.Int32
	server.agentTokenGenerator = func() (string, error) { return fmt.Sprintf("fake-token-%d", sequence.Add(1)), nil }
	var wg sync.WaitGroup
	var winners atomic.Int32
	var winner string
	var mu sync.Mutex
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stream := authTestStream("")
			err := server.ConnectStream(stream)
			if len(stream.welcomed) == 0 {
				if err == nil || errors.Is(err, io.EOF) {
					t.Errorf("loser accepted: %v", err)
				}
				return
			}
			if !errors.Is(err, io.EOF) {
				t.Error(err)
			}
			winners.Add(1)
			mu.Lock()
			winner = stream.welcomed[0].AuthToken
			mu.Unlock()
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("welcomed %d connections", winners.Load())
	}
	var stored goapistore.AgentTokenMeta
	if err := json.Unmarshal([]byte(kv.value), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Token != winner || kv.writes != 0 {
		t.Fatal("winner token overwritten")
	}
}

func TestAgentEnrollmentUsesStreamContext(t *testing.T) {
	kv := &agentAuthKV{}
	server := authTestServer(kv)
	stream := authTestStream("")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream.ctx = ctx
	if err := server.ConnectStream(stream); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if kv.creates != 0 || len(stream.welcomed) != 0 {
		t.Fatal("cancelled stream enrolled")
	}
}

func TestAgentEnrollmentGenerationFailureDoesNotCreateToken(t *testing.T) {
	kv := &agentAuthKV{}
	server := authTestServer(kv)
	expected := errors.New("synthetic entropy failure")
	server.agentTokenGenerator = func() (string, error) { return "", expected }
	token, err := server.authenticateAgent(context.Background(), "fixture", "", "")
	if token != "" || !errors.Is(err, expected) || kv.creates != 0 {
		t.Fatalf("generation failure was not closed: %v", err)
	}
}

func TestAgentEnrollmentRejectsStoresWithoutAtomicCreate(t *testing.T) {
	kv := &agentAuthKV{}
	legacy := struct{ goapistore.KVStore }{kv}
	server := authTestServer(legacy)
	stream := authTestStream("")
	if err := server.ConnectStream(stream); err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("legacy store accepted: %v", err)
	}
	if len(stream.welcomed) != 0 || kv.writes != 0 || kv.creates != 0 {
		t.Fatal("unsafe fallback or token disclosure")
	}
}

func TestAgentEnrollmentUncertainWriteNeverWelcomesOrOverwrites(t *testing.T) {
	kv := &agentAuthKV{createErr: errors.New("synthetic lost write acknowledgement"), commitOnError: true}
	server := authTestServer(kv)
	stream := authTestStream("")
	if err := server.ConnectStream(stream); !errors.Is(err, kv.createErr) {
		t.Fatalf("write uncertainty lost: %v", err)
	}
	if len(stream.welcomed) != 0 || !kv.present {
		t.Fatal("uncertain enrollment welcomed or test failed to persist")
	}
	original := kv.value
	kv.createErr = nil
	retry := authTestStream("")
	if err := server.ConnectStream(retry); err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("tokenless retry accepted: %v", err)
	}
	if len(retry.welcomed) != 0 || kv.value != original || kv.creates != 1 || kv.writes != 0 {
		t.Fatal("uncertain enrollment overwritten or disclosed")
	}
}

type fixedEnrollmentAuthorizer struct {
	identity enrollment.Identity
	reject   bool
	revoked  bool
}

func (a *fixedEnrollmentAuthorizer) AuthorizeEnrollment(_ context.Context, req enrollment.Request) (enrollment.Identity, error) {
	if a.reject || req.Credential != "synthetic-credential" {
		return enrollment.Identity{}, errors.New("rejected")
	}
	return a.identity, nil
}
func (a *fixedEnrollmentAuthorizer) ValidateCredential(_ context.Context, id string) error {
	if a.revoked || id != a.identity.CredentialID {
		return errors.New("revoked")
	}
	return nil
}

func TestEnrollmentCredentialAttributionAndRevocation(t *testing.T) {
	kv := &agentAuthKV{}
	server := authTestServer(kv)
	policy := &fixedEnrollmentAuthorizer{identity: enrollment.Identity{SubjectID: "opaque-subject", CredentialID: "credential-ref"}}
	server.SetEnrollmentAuthorizer(policy)
	denied := authTestStream("")
	denied.initial.EnrollmentCredential = "invalid"
	if err := server.ConnectStream(denied); err == nil || len(denied.welcomed) != 0 || kv.creates != 0 {
		t.Fatalf("invalid credential accepted: %v", err)
	}
	first := authTestStream("")
	first.initial.EnrollmentCredential = "synthetic-credential"
	if err := server.ConnectStream(first); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	var meta goapistore.AgentTokenMeta
	if err := json.Unmarshal([]byte(kv.value), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.SubjectID != "opaque-subject" || meta.CredentialID != "credential-ref" {
		t.Fatalf("lost attribution: %+v", meta)
	}
	if strings.Contains(kv.value, "synthetic-credential") {
		t.Fatal("secret persisted")
	}
	reconnect := authTestStream("fake-new-token")
	if err := server.ConnectStream(reconnect); !errors.Is(err, io.EOF) {
		t.Fatalf("valid reconnect: %v", err)
	}
	policy.revoked = true
	revoked := authTestStream("fake-new-token")
	if err := server.ConnectStream(revoked); err == nil || len(revoked.welcomed) != 0 {
		t.Fatalf("revoked credential accepted: %v", err)
	}
}

func TestSharedEnrollmentCredential(t *testing.T) {
	a := sharedCredentialAuthorizer{credential: "synthetic-credential"}
	if _, err := a.AuthorizeEnrollment(context.Background(), enrollment.Request{Credential: "wrong"}); err == nil {
		t.Fatal("wrong credential accepted")
	}
	id, err := a.AuthorizeEnrollment(context.Background(), enrollment.Request{Credential: "synthetic-credential"})
	if err != nil || id.CredentialID == "" {
		t.Fatalf("valid credential: %v", err)
	}
	if err := a.ValidateCredential(context.Background(), id.CredentialID); err != nil {
		t.Fatal(err)
	}
	if err := (sharedCredentialAuthorizer{credential: "new"}).ValidateCredential(context.Background(), id.CredentialID); err == nil {
		t.Fatal("rotated credential accepted")
	}
}
