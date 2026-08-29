package agent

import (
	"io"
	"strings"
	"testing"
)

func newStreamTestManager(beanID string) (*Manager, *Session, *runningProcess) {
	m := &Manager{
		sessions:    make(map[string]*Session),
		processes:   make(map[string]*runningProcess),
		subscribers: make(map[string][]chan struct{}),
	}
	session := &Session{
		ID:           beanID,
		AgentType:    "claude",
		Status:       StatusRunning,
		Messages:     []Message{{Role: RoleUser, Content: "hello"}},
		streamingIdx: -1,
	}
	m.sessions[beanID] = session
	proc := &runningProcess{done: make(chan struct{})}
	m.processes[beanID] = proc
	return m, session, proc
}

// TestReadOutputOversizedLine covers a stream the scanner cannot read: a single
// line past its 1MB buffer. Without a check on scanner.Err() the read ended
// silently and the session stayed RUNNING forever, with nothing in the UI
// saying why the agent had gone quiet.
func TestReadOutputOversizedLine(t *testing.T) {
	m, session, proc := newStreamTestManager("bean-oversized")

	huge := strings.Repeat("x", 2*1024*1024)
	line := `{"type":"content_block_delta","delta":{"type":"text_delta","text":"` + huge + `"}}`

	m.readOutput("bean-oversized", strings.NewReader(line), "", proc)

	if session.Status != StatusError {
		t.Errorf("Status = %q, want %q", session.Status, StatusError)
	}
	if session.Error == "" {
		t.Error("Error is empty, want a message naming the stream failure")
	}
}

// TestReadOutputBrokenPipe covers the stream dying mid-flight — the agent
// process crashed or the pipe broke.
func TestReadOutputBrokenPipe(t *testing.T) {
	m, session, proc := newStreamTestManager("bean-broken")

	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write([]byte(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"partial"}}` + "\n"))
		_ = pw.CloseWithError(io.ErrUnexpectedEOF)
	}()

	m.readOutput("bean-broken", pr, "", proc)

	if session.Status != StatusError {
		t.Errorf("Status = %q, want %q", session.Status, StatusError)
	}
	if session.Error == "" {
		t.Error("Error is empty, want a message naming the stream failure")
	}
}

// TestReadOutputCleanEndLeavesStatus makes sure the error path does not fire on
// an ordinary end of stream — that one is spawnAndRun's job to finish.
func TestReadOutputCleanEndLeavesStatus(t *testing.T) {
	m, session, proc := newStreamTestManager("bean-clean")

	lines := strings.Join([]string{
		`{"type":"content_block_start","content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"all good"}}`,
		`{"type":"result","session_id":"sess-1"}`,
	}, "\n")

	m.readOutput("bean-clean", strings.NewReader(lines), "", proc)

	if session.Status == StatusError {
		t.Errorf("Status = %q after a clean stream, want anything but error (Error=%q)", session.Status, session.Error)
	}
}
