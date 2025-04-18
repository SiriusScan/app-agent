package scan

import (
	"context"
	"net"
	"testing"
	"time"

	"log/slog"

	pb "github.com/SiriusScan/app-agent/proto/scan/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

type mockScanServer struct {
	pb.UnimplementedScanServiceServer
	startScanFunc       func(context.Context, *pb.StartScanRequest) (*pb.StartScanResponse, error)
	checkScanStatusFunc func(context.Context, *pb.CheckScanStatusRequest) (*pb.CheckScanStatusResponse, error)
}

func (m *mockScanServer) StartScan(ctx context.Context, req *pb.StartScanRequest) (*pb.StartScanResponse, error) {
	if m.startScanFunc != nil {
		return m.startScanFunc(ctx, req)
	}
	return &pb.StartScanResponse{ScanId: "test-scan-id"}, nil
}

func (m *mockScanServer) CheckScanStatus(ctx context.Context, req *pb.CheckScanStatusRequest) (*pb.CheckScanStatusResponse, error) {
	if m.checkScanStatusFunc != nil {
		return m.checkScanStatusFunc(ctx, req)
	}
	return &pb.CheckScanStatusResponse{
		Status:    "completed",
		Findings:  []string{"test-finding"},
		Error:     "",
		Timestamp: time.Now().Unix(),
	}, nil
}

func setupTest(t *testing.T) (*GRPCScanner, func()) {
	t.Helper()

	// Create a new gRPC server with the mock implementation
	s := grpc.NewServer()
	mockServer := &mockScanServer{}
	pb.RegisterScanServiceServer(s, mockServer)

	// Start serving on the listener in a goroutine
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Errorf("error serving server: %v", err)
		}
	}()

	// Create a client connection to the in-memory server
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	// Create a new scanner with the test connection
	scanner := &GRPCScanner{
		logger: slog.Default(),
		client: pb.NewScanServiceClient(conn),
		conn:   conn,
	}

	cleanup := func() {
		conn.Close()
		s.Stop()
	}

	return scanner, cleanup
}

func TestGRPCScanner_Start(t *testing.T) {
	scanner, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()
	target := &Target{
		Value: "test-target",
		Type:  "test-type",
		Args:  map[string]string{"arg1": "value1", "arg2": "value2"},
	}

	err := scanner.Start(ctx, target)
	assert.NoError(t, err)
	assert.NotEmpty(t, scanner.scanID)

	// Test starting a scan when one is already in progress
	err = scanner.Start(ctx, target)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan already in progress")

	// Test with invalid target
	err = scanner.Start(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target")
}

func TestGRPCScanner_Stop(t *testing.T) {
	scanner, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test stopping when no scan is in progress
	err := scanner.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no scan in progress")

	// Start a scan
	target := &Target{Value: "test-target"}
	err = scanner.Start(ctx, target)
	assert.NoError(t, err)

	// Stop the scan
	err = scanner.Stop(ctx)
	assert.NoError(t, err)
	assert.Empty(t, scanner.scanID)
}

func TestGRPCScanner_Status(t *testing.T) {
	scanner, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test getting status when no scan is in progress
	result, err := scanner.Status(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no scan in progress")
	assert.Nil(t, result)

	// Start a scan
	target := &Target{Value: "test-target"}
	err = scanner.Start(ctx, target)
	assert.NoError(t, err)

	// Get status
	result, err = scanner.Status(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, target.Value, result.Target)
	assert.NotEmpty(t, result.Findings)
	assert.Empty(t, result.Error)
	assert.NotZero(t, result.Timestamp)
}

func TestGRPCScanner_Close(t *testing.T) {
	scanner, cleanup := setupTest(t)
	defer cleanup()

	err := scanner.Close()
	assert.NoError(t, err)

	// Test closing again (should be safe)
	err = scanner.Close()
	assert.NoError(t, err)
}
