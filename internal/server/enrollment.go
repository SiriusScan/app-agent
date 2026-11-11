package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"

	"github.com/SiriusScan/app-agent/enrollment"
)

// sharedCredentialAuthorizer preserves legacy global enrollment when no shared
// credential is configured. When configured, it is required on enrollment and
// changing it revokes subsequent reconnects for tokens issued under it.
type sharedCredentialAuthorizer struct{ credential string }

func (a sharedCredentialAuthorizer) credentialID() string {
	sum := sha256.Sum256([]byte(a.credential))
	return "sha256:" + hex.EncodeToString(sum[:])
}
func (a sharedCredentialAuthorizer) AuthorizeEnrollment(ctx context.Context, req enrollment.Request) (enrollment.Identity, error) {
	if err := ctx.Err(); err != nil {
		return enrollment.Identity{}, err
	}
	if a.credential == "" {
		return enrollment.Identity{}, nil
	}
	if subtle.ConstantTimeCompare([]byte(a.credential), []byte(req.Credential)) != 1 {
		return enrollment.Identity{}, errors.New("invalid enrollment credential")
	}
	return enrollment.Identity{CredentialID: a.credentialID()}, nil
}
func (a sharedCredentialAuthorizer) ValidateCredential(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.credential == "" || subtle.ConstantTimeCompare([]byte(a.credentialID()), []byte(id)) != 1 {
		return errors.New("enrollment credential revoked")
	}
	return nil
}

// SetEnrollmentAuthorizer installs an admission implementation before serving.
// Passing nil restores the configured Community authorizer.
func (s *Server) SetEnrollmentAuthorizer(a enrollment.Authorizer) { s.enrollmentAuthorizer = a }
func (s *Server) authorizer() enrollment.Authorizer {
	if s.enrollmentAuthorizer != nil {
		return s.enrollmentAuthorizer
	}
	if s.config != nil {
		return sharedCredentialAuthorizer{credential: s.config.EnrollmentCredential}
	}
	return sharedCredentialAuthorizer{}
}
