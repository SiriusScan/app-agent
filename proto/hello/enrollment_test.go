package hello

import (
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestEnrollmentCredentialIsAdditive(t *testing.T) {
	old := &AgentMessage{AgentId: "agent", AuthToken: "token"}
	b, err := proto.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AgentMessage
	if err := proto.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetEnrollmentCredential() != "" || decoded.GetAuthToken() != "token" {
		t.Fatalf("legacy message: %+v", &decoded)
	}
	newMessage := &AgentMessage{AgentId: "agent", EnrollmentCredential: "credential"}
	b, err = proto.Marshal(newMessage)
	if err != nil {
		t.Fatal(err)
	}
	if err := proto.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetEnrollmentCredential() != "credential" {
		t.Fatal("credential did not round trip")
	}
}
