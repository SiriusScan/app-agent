package scan

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestNewScanner(t *testing.T) {
	logger := slog.Default()
	scanner := NewScanner(logger)
	if scanner == nil {
		t.Error("NewScanner returned nil")
	}
}

func TestDefaultScanner_Start(t *testing.T) {
	logger := slog.Default()
	scanner := NewScanner(logger)

	tests := []struct {
		name    string
		target  *Target
		wantErr bool
	}{
		{
			name: "valid IP target",
			target: &Target{
				Value: "192.168.1.1",
				Type:  "ip",
			},
			wantErr: false,
		},
		{
			name: "valid CIDR target",
			target: &Target{
				Value: "192.168.1.0/24",
				Type:  "cidr",
			},
			wantErr: false,
		},
		{
			name: "valid hostname target",
			target: &Target{
				Value: "localhost",
				Type:  "hostname",
			},
			wantErr: false,
		},
		{
			name: "invalid IP target",
			target: &Target{
				Value: "invalid-ip",
				Type:  "ip",
			},
			wantErr: true,
		},
		{
			name:    "nil target",
			target:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := scanner.Start(ctx, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				// Give some time for the scan to start
				time.Sleep(100 * time.Millisecond)
				if err := scanner.Stop(ctx); err != nil {
					t.Errorf("Stop() error = %v", err)
				}
			}
		})
	}
}

func TestDefaultScanner_Stop(t *testing.T) {
	logger := slog.Default()
	scanner := NewScanner(logger)
	ctx := context.Background()

	// Test stopping without an active scan
	if err := scanner.Stop(ctx); err == nil {
		t.Error("Stop() should return error when no scan is in progress")
	}

	// Start a scan
	target := &Target{
		Value: "127.0.0.1",
		Type:  "ip",
	}
	if err := scanner.Start(ctx, target); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give some time for the scan to start
	time.Sleep(100 * time.Millisecond)

	// Stop the scan
	if err := scanner.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Try stopping again
	if err := scanner.Stop(ctx); err == nil {
		t.Error("Stop() should return error when called twice")
	}
}

func TestDefaultScanner_Status(t *testing.T) {
	logger := slog.Default()
	scanner := NewScanner(logger)
	ctx := context.Background()

	// Test status before starting a scan
	if _, err := scanner.Status(ctx); err == nil {
		t.Error("Status() should return error when no scan has been started")
	}

	// Start a scan
	target := &Target{
		Value: "127.0.0.1",
		Type:  "ip",
	}
	if err := scanner.Start(ctx, target); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give some time for the scan to start
	time.Sleep(100 * time.Millisecond)

	// Check status
	result, err := scanner.Status(ctx)
	if err != nil {
		t.Errorf("Status() error = %v", err)
	}
	if result == nil {
		t.Error("Status() returned nil result")
	} else {
		if result.Target != target.Value {
			t.Errorf("Status() target = %v, want %v", result.Target, target.Value)
		}
		if len(result.Findings) == 0 {
			t.Error("Status() returned no findings")
		}
	}

	// Clean up
	if err := scanner.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestDefaultScanner_ConcurrentScans(t *testing.T) {
	logger := slog.Default()
	scanner := NewScanner(logger)
	ctx := context.Background()

	// Start first scan
	target1 := &Target{
		Value: "127.0.0.1",
		Type:  "ip",
	}
	if err := scanner.Start(ctx, target1); err != nil {
		t.Fatalf("Start() first scan error = %v", err)
	}

	// Try to start second scan while first is running
	target2 := &Target{
		Value: "localhost",
		Type:  "hostname",
	}
	if err := scanner.Start(ctx, target2); err == nil {
		t.Error("Start() should return error when starting concurrent scan")
	}

	// Clean up
	if err := scanner.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}
