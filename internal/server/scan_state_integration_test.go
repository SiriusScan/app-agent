package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	valkey "github.com/valkey-io/valkey-go"
)

// Opt-in store contract test. Only a fresh random expiring key is touched;
// this test never reads or changes live scan state or dispatches any command.
func TestAgentScanStateStoreIntegration(t *testing.T) {
	address := os.Getenv("SIRIUS_TEST_VALKEY_ADDR")
	if address == "" {
		t.Skip("set SIRIUS_TEST_VALKEY_ADDR for isolated-key integration test")
	}
	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{address}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	kv := valkeyScanStateStore{client}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var suffix [16]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	key := "sirius:test:agent-scan-cas:" + hex.EncodeToString(suffix[:])
	defer func() { _ = client.Do(context.Background(), client.B().Del().Key(key).Build()).Error() }()
	if _, exists, err := kv.Get(ctx, key); err != nil || exists {
		t.Fatalf("missing GET: %v/%v", exists, err)
	}
	if ok, err := kv.CompareAndSetValue(ctx, key, "", "wrong"); err != nil || ok {
		t.Fatalf("missing key matched: %v/%v", ok, err)
	}
	if ok, err := kv.Create(ctx, key, "original", time.Minute); err != nil || !ok {
		t.Fatalf("create failed: %v/%v", ok, err)
	}
	if ok, err := kv.Create(ctx, key, "wrong", time.Hour); err != nil || ok {
		t.Fatalf("NX overwrote existing key: %v/%v", ok, err)
	}
	if ok, err := kv.CompareAndSetValue(ctx, key, "wrong", "wrong"); err != nil || ok {
		t.Fatalf("stale CAS matched: %v/%v", ok, err)
	}
	if ok, err := kv.CompareAndSetValue(ctx, key, "original", "updated"); err != nil || !ok {
		t.Fatalf("CAS failed: %v/%v", ok, err)
	}
	if ttl, err := client.Do(ctx, client.B().Ttl().Key(key).Build()).ToInt64(); err != nil || ttl <= 0 || ttl > 60 {
		t.Fatalf("TTL not preserved: %d/%v", ttl, err)
	}
	if value, exists, err := kv.Get(ctx, key); err != nil || !exists || value != "updated" {
		t.Fatalf("wrong value: %s/%v/%v", value, exists, err)
	}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := kv.CompareAndSetValue(ctx, key, "updated", "winner")
			if err != nil {
				t.Error(err)
			}
			if ok {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("concurrent winners: %d", winners.Load())
	}
	if err := client.Do(ctx, client.B().Del().Key(key).Build()).Error(); err != nil {
		t.Fatal(err)
	}
	if ok, err := kv.CompareAndSetValue(ctx, key, "winner", "resurrected"); err != nil || ok {
		t.Fatalf("deleted key resurrected: %v/%v", ok, err)
	}
}
