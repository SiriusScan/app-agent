// Package agentserver exposes construction of the Community agent server so an
// embedding can provide a policy-neutral enrollment authorizer before serving.
package agentserver

import (
	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/app-agent/internal/server"
	"go.uber.org/zap"
)

// Config specifies server transport and optional Community enrollment settings.
type Config = config.ServerConfig

// Server is the Community agent service. SetEnrollmentAuthorizer installs an
// admission implementation; it must be called before accepting streams.
type Server = server.Server

// New constructs the Community server with its standard services.
func New(cfg *Config, logger *zap.Logger) (*Server, error) { return server.NewServer(cfg, logger) }
