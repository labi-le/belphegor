//go:build e2e

// Package e2e drives two real belphegor nodes (built with -tags null, the
// display-less headless backend) over a loopback TCP transport and verifies,
// end to end, that a clipboard copy on one node is injected into the other AND
// that every business-logic step fired, by asserting on the nodes' debug logs.
//
// Run with:
//
//	go test -tags e2e ./e2e/ -run TestE2E -v
package e2e_test

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuf is a goroutine-safe sink for a child process's combined output.
type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func buildNullBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "belphegor-null")

	cmd := exec.Command("go", "build", "-tags", "null", "-o", bin, "./cmd/cli")
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build -tags null failed: %v\n%s", err, out)
	}
	return bin
}

type node struct {
	name    string
	log     *syncBuf
	inFile  string
	outFile string
}

// startNode launches one headless belphegor process. A distinct HOME/TMPDIR
// gives it its own single-instance lock and file cache; a distinct
// BELPHEGOR_NODE_ID gives it a distinct network identity (both nodes would
// otherwise hash the same MAC to the same id and treat each other's messages
// as self-originated).
func startNode(ctx context.Context, t *testing.T, bin, name, home string, port, nodeID int, connectTo, secret string) *node {
	t.Helper()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}

	n := &node{
		name:    name,
		log:     &syncBuf{},
		inFile:  filepath.Join(home, "clip_in"),
		outFile: filepath.Join(home, "clip_out"),
	}

	args := []string{
		"-p", strconv.Itoa(port),
		"--transport", "tcp",
		"--node_discover=false",
		"--secret", secret,
		"--verbose",
	}
	if connectTo != "" {
		args = append(args, "-c", connectTo)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(),
		"TMPDIR="+home,
		"BELPHEGOR_NODE_ID="+strconv.Itoa(nodeID),
		"BELPHEGOR_HEADLESS_IN="+n.inFile,
		"BELPHEGOR_HEADLESS_OUT="+n.outFile,
	)
	cmd.Stdout = n.log
	cmd.Stderr = n.log

	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", name, err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	return n
}

func waitPort(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%s never started listening within %s", addr, timeout)
}

func waitLog(t *testing.T, n *node, substr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(n.log.String(), substr) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%s: debug log never contained %q within %s\n---- %s log ----\n%s",
		n.name, substr, timeout, n.name, n.log.String())
}

func waitFileContains(t *testing.T, n *node, path, substr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil && strings.Contains(string(b), substr) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	got, _ := os.ReadFile(path)
	t.Fatalf("%s: %s never contained %q within %s (have %q)\n---- %s log ----\n%s",
		n.name, path, substr, timeout, got, n.name, n.log.String())
}

func TestE2E_ClipboardPropagation(t *testing.T) {
	bin := buildNullBinary(t)
	base := t.TempDir()
	const secret = "e2e-secret"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// node1 listens; node2 dials node1.
	n1 := startNode(ctx, t, bin, "node1", filepath.Join(base, "n1"), 19001, 1, "", secret)
	waitPort(t, "127.0.0.1:19001", 20*time.Second)

	n2 := startNode(ctx, t, bin, "node2", filepath.Join(base, "n2"), 19002, 2, "127.0.0.1:19001", secret)

	// Business logic #1: the two nodes complete a handshake and connect.
	waitLog(t, n2, "connected", 20*time.Second)

	// Drive a real clipboard copy on node1 by writing to its IN file.
	const payload = "e2e-clipboard-payload-42"
	if err := os.WriteFile(n1.inFile, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}

	// End-to-end: the copy must be injected into node2's clipboard (OUT file).
	waitFileContains(t, n2, n2.outFile, payload, 20*time.Second)

	// Business logic #2: verify each step fired in the debug logs.
	waitLog(t, n1, "new update", 5*time.Second)         // node1: detected + accepted (not dup)
	waitLog(t, n1, "announced", 5*time.Second)          // node1: broadcast announce
	waitLog(t, n2, "requesting message", 5*time.Second) // node2: pulls on announce
	waitLog(t, n1, "received request", 5*time.Second)   // node1: serves the request
	waitLog(t, n2, "set clipboard data", 5*time.Second) // node2: injects into clipboard

	// Business logic #3: node1 must NOT echo its own copy back into its own
	// clipboard (self-origin skip / loop prevention).
	if b, err := os.ReadFile(n1.outFile); err == nil && strings.Contains(string(b), payload) {
		t.Errorf("node1 injected its own copy (self-echo): %q", b)
	}
}

// TestE2E_FiveNodeStar drives a 5-node star (node1 hub, node2..5 leaves) and
// verifies a copy on the hub fans out to every leaf: broadcast -> per-leaf
// request -> per-leaf inject, all confirmed through content + debug logs.
func TestE2E_FiveNodeStar(t *testing.T) {
	bin := buildNullBinary(t)
	base := t.TempDir()
	const secret = "e2e-secret-5"
	const hubPort = 19101

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := startNode(ctx, t, bin, "node1", filepath.Join(base, "n1"), hubPort, 1, "", secret)
	waitPort(t, "127.0.0.1:"+strconv.Itoa(hubPort), 20*time.Second)

	leaves := make([]*node, 0, 4)
	for i := range 4 {
		id := i + 2
		leaf := startNode(ctx, t, bin,
			"node"+strconv.Itoa(id),
			filepath.Join(base, "n"+strconv.Itoa(id)),
			hubPort+id-1, id,
			"127.0.0.1:"+strconv.Itoa(hubPort), secret)
		leaves = append(leaves, leaf)
	}

	// every leaf hand-shakes with the hub
	for _, leaf := range leaves {
		waitLog(t, leaf, "connected", 25*time.Second)
	}

	// a single copy on the hub
	const payload = "five-node-broadcast-99"
	if err := os.WriteFile(hub.inFile, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}

	// must reach and be injected into every one of the 4 leaves
	for _, leaf := range leaves {
		waitFileContains(t, leaf, leaf.outFile, payload, 25*time.Second)
		waitLog(t, leaf, "requesting message", 5*time.Second)
		waitLog(t, leaf, "set clipboard data", 5*time.Second)
	}

	// the hub detected the local copy and broadcast it
	waitLog(t, hub, "new update", 5*time.Second)
	waitLog(t, hub, "announced", 5*time.Second)
}
