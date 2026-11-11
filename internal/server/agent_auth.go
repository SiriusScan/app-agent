package server

import (
	"context"
	"fmt"

	"github.com/SiriusScan/app-agent/enrollment"
	goapistore "github.com/SiriusScan/go-api/sirius/store"
)

// authenticateAgent returns a token only to the connection that atomically
// enrolled the agent. Failed reads, unsupported stores, and concurrent losers
// cannot enroll, replace a token, or register a stream.
func (s *Server) authenticateAgent(ctx context.Context, agentID, suppliedToken, credential string) (string, error) {
	if s.kvStore == nil {
		return "", fmt.Errorf("agent authentication store unavailable")
	}
	if suppliedToken != "" {
		meta, err := goapistore.ValidateAgentToken(ctx, s.kvStore, agentID, suppliedToken)
		if err != nil {
			return "", fmt.Errorf("invalid agent token: %w", err)
		}
		if meta.CredentialID != "" {
			if err := s.authorizer().ValidateCredential(ctx, meta.CredentialID); err != nil {
				return "", fmt.Errorf("enrollment credential invalid or revoked")
			}
		}
		return "", nil
	}

	exists, err := goapistore.AgentTokenExists(ctx, s.kvStore, agentID)
	if err != nil {
		return "", fmt.Errorf("agent token lookup failed: %w", err)
	}
	if exists {
		return "", fmt.Errorf("agent %s must present auth_token", agentID)
	}
	identity, err := s.authorizer().AuthorizeEnrollment(ctx, enrollment.Request{AgentID: agentID, Credential: credential})
	if err != nil {
		return "", fmt.Errorf("enrollment credential rejected")
	}
	generate := s.agentTokenGenerator
	if generate == nil {
		generate = goapistore.GenerateAgentToken
	}
	token, err := generate()
	if err != nil {
		return "", fmt.Errorf("failed to generate agent token: %w", err)
	}
	created, err := goapistore.StoreAgentTokenWithMetadataIfAbsent(ctx, s.kvStore, goapistore.AgentTokenMeta{AgentID: agentID, Token: token, SubjectID: identity.SubjectID, CredentialID: identity.CredentialID})
	if err != nil {
		return "", fmt.Errorf("failed to enroll agent: %w", err)
	}
	if !created {
		return "", fmt.Errorf("agent %s must present auth_token", agentID)
	}
	return token, nil
}
