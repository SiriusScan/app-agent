package scan

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
)

// Target represents a scan target with its configuration
type Target struct {
	Value string            // IP, CIDR, or hostname
	Type  string            // "ip", "cidr", "hostname"
	Args  map[string]string // Additional scan arguments
}

// Result represents the result of a scan operation
type Result struct {
	Target    string   // The scanned target
	Findings  []string // List of findings/vulnerabilities
	Error     string   // Error message if scan failed
	Timestamp int64    // Unix timestamp of when the scan completed
}

// Scanner defines the interface for performing scans
type Scanner interface {
	// Start initiates a scan for the given target
	Start(ctx context.Context, target *Target) error
	// Stop cancels an ongoing scan
	Stop(ctx context.Context) error
	// Status returns the current scan status
	Status(ctx context.Context) (*Result, error)
}

// DefaultScanner implements the Scanner interface
type DefaultScanner struct {
	logger     *slog.Logger
	target     *Target
	result     *Result
	cancelFunc context.CancelFunc
	mu         sync.RWMutex
	wg         sync.WaitGroup
}

// NewScanner creates a new instance of DefaultScanner
func NewScanner(logger *slog.Logger) Scanner {
	return &DefaultScanner{
		logger: logger,
	}
}

// Start initiates a scan for the given target
func (s *DefaultScanner) Start(ctx context.Context, target *Target) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFunc != nil {
		return fmt.Errorf("scan already in progress")
	}

	// Validate target
	if target == nil || target.Value == "" {
		return fmt.Errorf("invalid target")
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.target = target
	s.result = &Result{
		Target:    target.Value,
		Timestamp: 0, // Will be set when scan completes
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			s.cancelFunc = nil
			s.mu.Unlock()
		}()

		if err := s.runScan(ctx); err != nil {
			s.mu.Lock()
			s.result.Error = err.Error()
			s.mu.Unlock()
		}
	}()

	return nil
}

// Stop cancels an ongoing scan
func (s *DefaultScanner) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFunc == nil {
		return fmt.Errorf("no scan in progress")
	}

	s.cancelFunc()
	s.wg.Wait()
	return nil
}

// Status returns the current scan status
func (s *DefaultScanner) Status(ctx context.Context) (*Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.result == nil {
		return nil, fmt.Errorf("no scan results available")
	}

	return s.result, nil
}

// runScan performs the actual scanning operation
func (s *DefaultScanner) runScan(ctx context.Context) error {
	s.logger.Info("starting scan", "target", s.target.Value, "type", s.target.Type)

	var targets []string
	var err error

	// Process target based on type
	switch s.target.Type {
	case "ip":
		if net.ParseIP(s.target.Value) == nil {
			return fmt.Errorf("invalid IP address: %s", s.target.Value)
		}
		targets = []string{s.target.Value}
	case "cidr":
		_, ipNet, err := net.ParseCIDR(s.target.Value)
		if err != nil {
			return fmt.Errorf("invalid CIDR: %s", s.target.Value)
		}
		targets = expandCIDR(ipNet)
	case "hostname":
		ips, err := net.LookupHost(s.target.Value)
		if err != nil {
			return fmt.Errorf("hostname resolution failed: %s", err)
		}
		targets = ips
	default:
		return fmt.Errorf("unsupported target type: %s", s.target.Type)
	}

	// Scan each target
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.scanTarget(ctx, target); err != nil {
				s.logger.Error("scan failed", "target", target, "error", err)
				continue
			}
		}
	}

	return err
}

// scanTarget performs a scan on a single target
func (s *DefaultScanner) scanTarget(ctx context.Context, target string) error {
	s.logger.Debug("scanning target", "target", target)

	// TODO: Implement actual vulnerability scanning logic here
	// This could include:
	// - Port scanning
	// - Service detection
	// - Vulnerability checks
	// - etc.

	// For now, just add a placeholder finding
	s.mu.Lock()
	s.result.Findings = append(s.result.Findings, fmt.Sprintf("Scanned target: %s", target))
	s.mu.Unlock()

	return nil
}

// expandCIDR returns a list of IP addresses in the CIDR range
func expandCIDR(ipNet *net.IPNet) []string {
	var ips []string
	for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// Remove network and broadcast addresses for IPv4
	if len(ipNet.IP) == 4 && len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	return ips
}

// inc increments an IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
