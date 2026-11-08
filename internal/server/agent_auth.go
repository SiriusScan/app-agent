package server

import (
	"context"
	"fmt"

	goapistore "github.com/SiriusScan/go-api/sirius/store"
)

// authenticateAgent returns a token only to the connection that atomically
// enrolled the agent. Failed reads, unsupported stores, and concurrent losers
// cannot enroll, replace a token, or register a stream.
func (s *Server) authenticateAgent(ctx context.Context, agentID, suppliedToken string) (string, error) {
	if s.kvStore == nil {
		return "", fmt.Errorf("agent authentication store unavailable")
	}
	if suppliedToken != "" {
		if _, err := goapistore.ValidateAgentToken(ctx, s.kvStore, agentID, suppliedToken); err != nil {
			return "", fmt.Errorf("invalid agent token: %w", err)
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
	generate := s.agentTokenGenerator
	if generate == nil {
		generate = goapistore.GenerateAgentToken
	}
	token, err := generate()
	if err != nil {
		return "", fmt.Errorf("failed to generate agent token: %w", err)
	}
	created, err := goapistore.StoreAgentTokenIfAbsent(ctx, s.kvStore, agentID, token)
	if err != nil {
		return "", fmt.Errorf("failed to enroll agent: %w", err)
	}
	if !created {
		return "", fmt.Errorf("agent %s must present auth_token", agentID)
	}
	return token, nil
}
