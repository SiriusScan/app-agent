// Package telnetlinemode implements a non-destructive probe for GNU InetUtils telnetd
// LINEMODE / SLC negotiation behavior associated with CVE-2026-32746.
package telnetlinemode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/SiriusScan/app-agent/internal/modules"
	"github.com/SiriusScan/app-agent/internal/modules/registry"
	"github.com/SiriusScan/app-agent/internal/template/types"
)

// Telnet protocol constants (RFC 854 / telnet.h)
const (
	iac = 0xff
	do  = 0xfd
	will = 0xfb
	sb   = 0xfa
	se   = 0xf0

	optLinemode = 0x22
	optTTYPE      = 0x18
	optTSPEED     = 0x20
	lmSLC         = 0x03
)

// Module performs a local TCP negotiation probe similar to the public detect.py PoC.
type Module struct{}

// Execute connects to telnetd, completes a minimal option exchange, and reports whether
// the server accepts LINEMODE (and optionally advertises SLC subnegotiation).
//
// Config:
//   - host (string, optional): target host (default 127.0.0.1)
//   - port (int, optional): TCP port (default 23)
//   - timeout_seconds (int, optional): per-read deadline and dial timeout (default 10)
//   - negotiate_pause_ms (int, optional): delay between negotiation rounds (default 1000)
//   - recv_idle_timeout_ms (int, optional): max idle wait per recv round for more bytes (default 2000)
//   - require_slc (bool, optional): if true, match only when SLC subnegotiation is observed
func (m *Module) Execute(ctx context.Context, config modules.StepConfig) (*modules.Result, error) {
	host := config.GetString("host")
	if host == "" {
		host = "127.0.0.1"
	}
	port := config.GetInt("port")
	if port == 0 {
		port = 23
	}
	if port < 1 || port > 65535 {
		return &modules.Result{
			Matched: false,
			Error:   fmt.Sprintf("invalid port: %d", port),
		}, nil
	}

	timeoutSec := config.GetInt("timeout_seconds")
	if timeoutSec <= 0 {
		timeoutSec = 10
	}
	pauseMs := config.GetInt("negotiate_pause_ms")
	if pauseMs <= 0 {
		pauseMs = 1000
	}
	requireSLC := config.GetBool("require_slc")

	idleMs := config.GetInt("recv_idle_timeout_ms")
	if idleMs <= 0 {
		idleMs = 2000
	}
	readDeadline := time.Duration(idleMs) * time.Millisecond

	dialer := net.Dialer{Timeout: time.Duration(timeoutSec) * time.Second}
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return &modules.Result{
			Matched: false,
			Evidence: map[string]interface{}{
				"host": host,
				"port": port,
				"err":  err.Error(),
			},
			Error: fmt.Sprintf("connection failed: %v", err),
		}, nil
	}
	defer conn.Close()

	// Round 1: initial server options
	time.Sleep(time.Duration(pauseMs) * time.Millisecond)
	data1, err := recvAll(ctx, conn, readDeadline)
	if err != nil {
		return &modules.Result{Matched: false, Error: err.Error()}, nil
	}
	if len(data1) == 0 {
		return &modules.Result{
			Matched: false,
			Evidence: map[string]interface{}{
				"host": host,
				"port": port,
			},
			Error: "no initial data from server",
		}, nil
	}

	resp := buildClientResponse(data1)
	sent := append(resp,
		iac, will, optLinemode,
		iac, sb, optTTYPE, 0x00)
	sent = append(sent, []byte("xterm")...)
	sent = append(sent, iac, se)
	sent = append(sent, iac, sb, optTSPEED, 0x00)
	sent = append(sent, []byte("38400,38400")...)
	sent = append(sent, iac, se)

	if _, err := conn.Write(sent); err != nil {
		return &modules.Result{Matched: false, Error: fmt.Sprintf("write failed: %v", err)}, nil
	}

	// Round 2: LINEMODE / SLC from server
	time.Sleep(time.Duration(pauseMs) * time.Millisecond)
	data2, err := recvAll(ctx, conn, readDeadline)
	if err != nil {
		return &modules.Result{Matched: false, Error: err.Error()}, nil
	}

	gotLM, gotSLC := parseSecondRound(data2)
	matched := gotLM
	if requireSLC {
		matched = gotLM && gotSLC
	}

	evidence := map[string]interface{}{
		"host":              host,
		"port":              port,
		"linemode_accepted": gotLM,
		"slc_negotiation":   gotSLC,
		"bytes_round2":      len(data2),
	}

	return &modules.Result{
		Matched:  matched,
		Evidence: evidence,
	}, nil
}

func recvAll(ctx context.Context, conn net.Conn, perRead time.Duration) ([]byte, error) {
	var buf bytes.Buffer
	tmp := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return buf.Bytes(), ctx.Err()
		default:
		}
		if err := conn.SetReadDeadline(time.Now().Add(perRead)); err != nil {
			return nil, err
		}
		n, err := conn.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			if errors.Is(err, io.EOF) {
				break
			}
			return buf.Bytes(), err
		}
	}
	return buf.Bytes(), nil
}

func buildClientResponse(initial []byte) []byte {
	var out []byte
	i := 0
	for i+2 < len(initial) {
		if initial[i] != iac {
			i++
			continue
		}
		cmd := initial[i+1]
		opt := initial[i+2]
		if cmd == do {
			out = append(out, iac, will, opt)
		} else if cmd == will {
			out = append(out, iac, do, opt)
		}
		i += 3
	}
	return out
}

func parseSecondRound(data []byte) (gotLinemode, gotSLC bool) {
	for i := 0; i+2 < len(data); i++ {
		if data[i] == iac && data[i+1] == do && data[i+2] == optLinemode {
			gotLinemode = true
			break
		}
	}
	if bytes.Contains(data, []byte{iac, sb, optLinemode, lmSLC}) {
		gotSLC = true
	}
	return gotLinemode, gotSLC
}

func init() {
	descriptor := modules.Descriptor{
		Type:        "telnet_linemode",
		Name:        "Telnet LINEMODE / SLC negotiation probe",
		Description: "Non-destructive TCP probe for GNU InetUtils-style telnetd LINEMODE and SLC handling (CVE-2026-32746 exposure signal)",
		Version:     "1.0.0",
		Author:      "Sirius Scan",
		SupportedOS: []string{string(types.PlatformLinux), string(types.PlatformDarwin), string(types.PlatformWindows)},
		ConfigDocs: map[string]string{
			"host":                   "Target host (default 127.0.0.1)",
			"port":                   "Telnet TCP port (default 23)",
			"timeout_seconds":        "Dial timeout in seconds (default 10)",
			"negotiate_pause_ms":     "Pause between negotiation rounds in ms (default 1000)",
			"recv_idle_timeout_ms":   "Idle timeout for accumulating each negotiation read in ms (default 2000)",
			"require_slc":            "If true, require SLC subnegotiation bytes in addition to DO LINEMODE",
		},
	}
	if err := registry.Register(&Module{}, descriptor); err != nil {
		panic(fmt.Sprintf("failed to register telnet_linemode module: %v", err))
	}
}
