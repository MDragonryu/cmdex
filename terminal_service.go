package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ptyWinsize struct {
	Rows uint16
	Cols uint16
}

// SessionInfo is the public metadata for a terminal session, sent to the frontend.
type SessionInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Running    bool   `json:"running"`
	ShellPath  string `json:"shellPath"`
	WorkingDir string `json:"workingDir"`
}

// sessionState is the internal per-session state: PTY, process, goroutine tracking.
type sessionState struct {
	id              string
	name            string
	workingDir      string
	createdAt       time.Time

	mu              sync.Mutex
	ptmx            *os.File
	cmd             *exec.Cmd
	shellPath       string
	shellFlag       string
	lastSize        ptyWinsize
	stopCh          chan struct{}
	running         bool
	starting        bool
	intentionalStop bool

	readerWg     sync.WaitGroup
	outputCh     chan string
	outputSeq    uint64
	emitterWg    sync.WaitGroup
	droppedCount atomic.Uint64
}

// TerminalService manages multiple PTY-backed shell sessions.
type TerminalService struct {
	mu              sync.RWMutex
	sessions        map[string]*sessionState
	activeSessionID string
	sessionCounter  int
}

// info returns the public SessionInfo for this session.
func (ss *sessionState) info() *SessionInfo {
	return &SessionInfo{
		ID:         ss.id,
		Name:       ss.name,
		Running:    ss.running,
		ShellPath:  ss.shellPath,
		WorkingDir: ss.workingDir,
	}
}

// getWorkingDir returns os.UserHomeDir() as the working directory for new sessions.
func getWorkingDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// resolveSession resolves a sessionId to a *sessionState pointer.
// If sessionId is "", falls back to the active session.
// The caller must NOT hold s.mu while operating on the returned sessionState.
func (s *TerminalService) resolveSession(sessionId string) (*sessionState, error) {
	if sessionId == "" {
		s.mu.RLock()
		sessionId = s.activeSessionID
		s.mu.RUnlock()
		if sessionId == "" {
			return nil, fmt.Errorf("no active session")
		}
	}
	s.mu.RLock()
	ss, ok := s.sessions[sessionId]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionId)
	}
	return ss, nil
}

func detectShell() (path, flag string) {
	if runtime.GOOS == "windows" {
		for _, shell := range []string{"pwsh", "powershell"} {
			if lp, err := exec.LookPath(shell); err == nil {
				return lp, "-NoLogo"
			}
		}
		return "cmd", ""
	}

	path = os.Getenv("SHELL")
	if path == "" {
		path = "/bin/sh"
	}
	flag = "-l"
	return path, flag
}

// ========== Service Lifecycle ==========

func (s *TerminalService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	terminalSvc = s

	s.sessions = make(map[string]*sessionState)

	_, err := s.CreateSession()
	if err != nil {
		fmt.Printf("TerminalService: CreateSession failed (graceful degradation): %v\n", err)
	}
	return nil
}

func (s *TerminalService) ServiceShutdown() error {
	s.mu.Lock()
	sessionIDs := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	s.mu.Unlock()

	for _, id := range sessionIDs {
		s.CloseSession(id)
	}
	return nil
}

// ========== Session CRUD ==========

// CreateSession creates a new terminal session with a UUID v4 ID and default name "Terminal N".
func (s *TerminalService) CreateSession() (*SessionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionCounter++
	name := fmt.Sprintf("Terminal %d", s.sessionCounter)
	id := uuid.New().String()

	ss := &sessionState{
		id:         id,
		name:       name,
		workingDir: getWorkingDir(),
		createdAt:  time.Now(),
		lastSize:   ptyWinsize{Rows: 24, Cols: 80},
	}

	s.sessions[id] = ss
	if s.activeSessionID == "" {
		s.activeSessionID = id
	}

	return ss.info(), nil
}

// ListSessions returns SessionInfo for all sessions in the manager.
func (s *TerminalService) ListSessions() []*SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*SessionInfo, 0, len(s.sessions))
	for _, ss := range s.sessions {
		result = append(result, ss.info())
	}
	return result
}

// CloseSession removes a session from the manager. If the closed session was the
// active one, another session is assigned as active (or cleared if none remain).
func (s *TerminalService) CloseSession(id string) error {
	s.mu.Lock()
	ss, ok := s.sessions[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("CloseSession: session not found: %s", id)
	}

	delete(s.sessions, id)

	if s.activeSessionID == id {
		s.activeSessionID = ""
		for otherID := range s.sessions {
			s.activeSessionID = otherID
			break
		}
	}
	s.mu.Unlock()

	// TODO(Plan 02): call ss.stopSession() to kill PTY and wait for goroutines
	_ = ss

	return nil
}

// RenameSession updates the name of a session. Returns an error if name is empty
// or the session does not exist.
func (s *TerminalService) RenameSession(id string, name string) error {
	if name == "" {
		return fmt.Errorf("RenameSession: name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ss, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("RenameSession: session not found: %s", id)
	}

	ss.name = name
	return nil
}

// SetActiveSession sets the active session by ID.
func (s *TerminalService) SetActiveSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[id]; !ok {
		return fmt.Errorf("SetActiveSession: session not found: %s", id)
	}

	s.activeSessionID = id
	return nil
}

// GetActiveSession returns the SessionInfo for the currently active session,
// or nil if no active session exists.
func (s *TerminalService) GetActiveSession() *SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.activeSessionID == "" {
		return nil
	}

	ss, ok := s.sessions[s.activeSessionID]
	if !ok {
		return nil
	}
	return ss.info()
}
