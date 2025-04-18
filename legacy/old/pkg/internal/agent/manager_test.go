package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/config"
	"github.com/SiriusScan/go-api/sirius/store"
)

// mockKVStore implements store.KVStore for testing
type mockKVStore struct {
	closeErr error
	mu       sync.Mutex
	values   map[string]string
}

func newMockKVStore(closeErr error) *mockKVStore {
	return &mockKVStore{
		closeErr: closeErr,
		values:   make(map[string]string),
	}
}

func (m *mockKVStore) SetValue(ctx context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
	return nil
}

func (m *mockKVStore) GetValue(ctx context.Context, key string) (store.ValkeyResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if val, ok := m.values[key]; ok {
		return store.ValkeyResponse{
			Message: store.ValkeyValue{Value: val},
			Type:    "test",
		}, nil
	}
	return store.ValkeyResponse{}, fmt.Errorf("key not found: %s", key)
}

func (m *mockKVStore) Close() error {
	return m.closeErr
}

// mockStoreFactory creates a new mockKVStore for testing
func mockStoreFactory(closeErr error) func() (store.KVStore, error) {
	return func() (store.KVStore, error) {
		return newMockKVStore(closeErr), nil
	}
}

func TestNewManager(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	cfg := &config.Config{}

	tests := []struct {
		name         string
		cfg          *config.Config
		logger       *slog.Logger
		storeFactory func() (store.KVStore, error)
		wantErr      bool
	}{
		{
			name:         "valid configuration",
			cfg:          cfg,
			logger:       logger,
			storeFactory: mockStoreFactory(nil),
			wantErr:      false,
		},
		{
			name:         "nil config",
			cfg:          nil,
			logger:       logger,
			storeFactory: mockStoreFactory(nil),
			wantErr:      false,
		},
		{
			name:         "nil logger",
			cfg:          cfg,
			logger:       nil,
			storeFactory: mockStoreFactory(nil),
			wantErr:      false,
		},
		{
			name:         "nil store factory uses default",
			cfg:          cfg,
			logger:       logger,
			storeFactory: nil,
			wantErr:      false,
		},
		{
			name:   "store factory returns error",
			cfg:    cfg,
			logger: logger,
			storeFactory: func() (store.KVStore, error) {
				return nil, fmt.Errorf("store creation failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewManager(tt.cfg, tt.logger, tt.storeFactory)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewManager() returned nil manager")
			}
		})
	}
}

func TestManager_StartStop(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	cfg := &config.Config{}

	tests := []struct {
		name           string
		closeErr       error
		startupDelay   time.Duration // Time to wait after starting before stopping
		shutdownDelay  time.Duration // Max time to wait for shutdown
		wantStartErr   bool
		wantStopErr    bool
		checkConnected bool // Whether to verify connection status
	}{
		{
			name:           "successful start and stop",
			closeErr:       nil,
			startupDelay:   50 * time.Millisecond,
			shutdownDelay:  100 * time.Millisecond,
			wantStartErr:   false,
			wantStopErr:    false,
			checkConnected: true,
		},
		{
			name:           "error on close",
			closeErr:       fmt.Errorf("failed to close"),
			startupDelay:   50 * time.Millisecond,
			shutdownDelay:  100 * time.Millisecond,
			wantStartErr:   false,
			wantStopErr:    false, // We don't propagate close errors
			checkConnected: true,
		},
		{
			name:           "immediate shutdown",
			closeErr:       nil,
			startupDelay:   0,
			shutdownDelay:  100 * time.Millisecond,
			wantStartErr:   false,
			wantStopErr:    false,
			checkConnected: false,
		},
		{
			name:           "shutdown timeout",
			closeErr:       nil,
			startupDelay:   50 * time.Millisecond,
			shutdownDelay:  1 * time.Millisecond, // Very short timeout
			wantStartErr:   false,
			wantStopErr:    true, // Should get context deadline exceeded
			checkConnected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create manager with mocked store
			m, err := NewManager(cfg, logger, mockStoreFactory(tt.closeErr))
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			// Create a context with timeout for the entire test
			ctx, cancel := context.WithTimeout(context.Background(), tt.shutdownDelay+tt.startupDelay+100*time.Millisecond)
			defer cancel()

			// Start the manager
			errCh := make(chan error, 1)
			go func() {
				errCh <- m.Start(ctx)
			}()

			// Wait for startup delay
			time.Sleep(tt.startupDelay)

			// Create shutdown context
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), tt.shutdownDelay)
			defer shutdownCancel()

			// Stop the manager
			if err := m.Stop(shutdownCtx); (err != nil) != tt.wantStopErr {
				t.Errorf("Manager.Stop() error = %v, wantStopErr %v", err, tt.wantStopErr)
			}

			// Check if Start returned any errors
			select {
			case err := <-errCh:
				if (err != nil) != tt.wantStartErr {
					t.Errorf("Manager.Start() error = %v, wantStartErr %v", err, tt.wantStartErr)
				}
			case <-time.After(tt.shutdownDelay + 50*time.Millisecond):
				t.Error("Manager.Start() didn't return after Stop()")
			}
		})
	}
}

func TestManager_QueueMessageHandling(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	cfg := &config.Config{}

	tests := []struct {
		name      string
		queueName string
		message   string
	}{
		{
			name:      "task message",
			queueName: "tasks",
			message:   "test task message",
		},
		{
			name:      "status message",
			queueName: "status",
			message:   "test status message",
		},
		{
			name:      "empty message",
			queueName: "tasks",
			message:   "",
		},
		{
			name:      "unknown queue",
			queueName: "unknown",
			message:   "test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewManager(cfg, logger, mockStoreFactory(nil))
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			// Test that handleQueueMessage doesn't panic
			ctx := context.Background()
			m.handleQueueMessage(ctx, tt.queueName, tt.message)
		})
	}
}
