package telnetlinemode

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/SiriusScan/app-agent/internal/modules"
)

func TestParseSecondRound(t *testing.T) {
	lm, slc := parseSecondRound([]byte{iac, do, optLinemode})
	if !lm || slc {
		t.Fatalf("expected linemode only, got lm=%v slc=%v", lm, slc)
	}
	raw := []byte{0x01, iac, sb, optLinemode, lmSLC, 0x00, iac, se}
	_, slc2 := parseSecondRound(raw)
	if !slc2 {
		t.Fatal("expected SLC subnegotiation detected")
	}
}

func TestModule_Execute_vulnerableHandshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.Write([]byte{iac, do, 0x01})
		buf := make([]byte, 4096)
		_, _ = c.Read(buf)
		_, _ = c.Write([]byte{
			iac, do, optLinemode,
			iac, sb, optLinemode, lmSLC, 0x00, iac, se,
		})
		_, _ = c.Read(buf)
	}()

	mod := &Module{}
	ctx := context.Background()
	res, err := mod.Execute(ctx, modules.StepConfig{
		"host":                   "127.0.0.1",
		"port":                   port,
		"timeout_seconds":        5,
		"negotiate_pause_ms":     20,
		"recv_idle_timeout_ms":   50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched {
		t.Fatalf("expected match, got %+v", res)
	}
}

func TestModule_Execute_connectionRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	_ = ln.Close()

	mod := &Module{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := mod.Execute(ctx, modules.StepConfig{
		"host":            "127.0.0.1",
		"port":            port,
		"timeout_seconds": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched || res.Error == "" {
		t.Fatalf("expected no match and error, got %+v", res)
	}
}
