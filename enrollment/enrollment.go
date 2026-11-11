// Package enrollment defines policy-neutral agent admission.
package enrollment

import "context"

// Request is presented only when an agent has no established token.
// Credential is secret and must never be logged or retained in Identity.
type Request struct {
	AgentID    string
	Credential string
}

// Identity is non-secret attribution associated with the issued agent token.
// Empty fields represent the legacy global enrollment identity.
type Identity struct {
	SubjectID    string `json:"subject_id,omitempty"`
	CredentialID string `json:"credential_id,omitempty"`
}

// Authorizer decides first enrollment and checks credential references on
// reconnect. Implementations must treat backend errors as denials and must not
// return secret material in errors. A nonempty CredentialID must be checked on
// reconnect, allowing revocation without sending the credential again.
type Authorizer interface {
	AuthorizeEnrollment(context.Context, Request) (Identity, error)
	ValidateCredential(context.Context, string) error
}
