package main

import (
	"runtime"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func testCleanup(t *testing.T, dbConn *DB, catID, cmdID string) {
	t.Helper()
	dbConn.conn.Exec(`DELETE FROM commands WHERE id = ?`, cmdID)
	dbConn.conn.Exec(`DELETE FROM categories WHERE id = ?`, catID)
}

func testDBCreateCommand(t *testing.T, catID, cmdID, categoryName, cmdTitle, scriptContent, workingDirJSON string) (*DB, func()) {
	t.Helper()

	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}

	testCleanup(t, initDB, catID, cmdID)

	_, err = initDB.conn.Exec(`INSERT INTO categories (id, name, icon, color) VALUES (?, ?, '', '')`, catID, categoryName)
	if err != nil {
		initDB.Close()
		t.Fatalf("insert category: %v", err)
	}

	_, err = initDB.conn.Exec(
		`INSERT INTO commands (id, category_id, title, script_content, working_dir, position) VALUES (?, ?, ?, ?, ?, 0)`,
		cmdID, catID, cmdTitle, GenerateScript(scriptContent), workingDirJSON,
	)
	if err != nil {
		initDB.Close()
		t.Fatalf("insert command: %v", err)
	}

	prevDB := db
	db = initDB

	return initDB, func() {
		db = prevDB
		initDB.Close()
	}
}

// testWithTerminalSvc sets up a real TerminalService so RunCommand can dispatch
// via terminalSvc.Write. Mirrors the save/restore pattern from
// TestTerminalService_ServiceStartupAssignsTerminalSvc. The returned cleanup
// func must be called (typically via defer) to restore the previous global
// state and shut down the test service.
func testWithTerminalSvc(t *testing.T) func() {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode (requires real PTY)")
	}
	prevTerminalSvc := terminalSvc
	terminalSvc = nil
	ts := &TerminalService{}
	if err := ts.ServiceStartup(nil, application.ServiceOptions{}); err != nil {
		terminalSvc = prevTerminalSvc
		t.Skipf("TerminalService.ServiceStartup failed: %v", err)
	}
	return func() {
		_ = ts.ServiceShutdown()
		terminalSvc = prevTerminalSvc
	}
}

func TestRunCommand_FinalCmdWithWorkingDir(t *testing.T) {
	defer testWithTerminalSvc(t)()

	workingDirJSON := `{"` + runtime.GOOS + `":"/Users/test"}`
	_, cleanup := testDBCreateCommand(t, "test-cat-wd-18", "test-cmd-wd-18", "TestWD", "Test Cmd WD", "echo hello", workingDirJSON)
	defer cleanup()

	svc := &ExecutionService{}
	record := svc.RunCommand("test-cmd-wd-18", nil)

	if record.Error != "" {
		t.Errorf("Error = %q, want empty", record.Error)
	}

	want := "cd '/Users/test' && echo hello\n"
	if record.FinalCmd != want {
		t.Errorf("FinalCmd = %q, want %q", record.FinalCmd, want)
	}
}

func TestRunCommand_FinalCmdNoWorkingDir(t *testing.T) {
	defer testWithTerminalSvc(t)()

	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	defer initDB.Close()

	settings, err := initDB.GetSettings()
	if err == nil {
		settings.DefaultWorkingDir = &OSPathMap{}
		_ = initDB.SetSettings(settings)
	}

	_, cleanup := testDBCreateCommand(t, "test-cat-nowd-18", "test-cmd-nowd-18", "TestNoWD", "Test Cmd NoWD", "echo hello", `{}`)
	defer cleanup()

	svc := &ExecutionService{}
	record := svc.RunCommand("test-cmd-nowd-18", nil)

	if record.Error != "" {
		t.Errorf("Error = %q, want empty", record.Error)
	}

	want := "echo hello\n"
	if record.FinalCmd != want {
		t.Errorf("FinalCmd = %q, want %q", record.FinalCmd, want)
	}
}

func TestRunCommand_FinalCmdMultilineScript(t *testing.T) {
	defer testWithTerminalSvc(t)()

	workingDirJSON := `{"` + runtime.GOOS + `":"/Users/test"}`
	_, cleanup := testDBCreateCommand(t, "test-cat-ml-18", "test-cmd-ml-18", "TestML", "Test Cmd ML", "line1\nline2", workingDirJSON)
	defer cleanup()

	svc := &ExecutionService{}
	record := svc.RunCommand("test-cmd-ml-18", nil)

	if record.Error != "" {
		t.Errorf("Error = %q, want empty", record.Error)
	}

	want := "cd '/Users/test' && line1\nline2\n"
	if record.FinalCmd != want {
		t.Errorf("FinalCmd = %q, want %q", record.FinalCmd, want)
	}
}

func TestRunCommand_GetCommandError(t *testing.T) {
	defer testWithTerminalSvc(t)()

	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	defer initDB.Close()

	prevDB := db
	db = initDB
	defer func() { db = prevDB }()

	svc := &ExecutionService{}
	record := svc.RunCommand("nonexistent-id-for-test", nil)

	if record.Error == "" {
		t.Error("expected Error field to be set when GetCommand fails, got empty string")
	}
	if record.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", record.ExitCode)
	}
}

func TestRunCommand_NoHistoryPersistence(t *testing.T) {
	defer testWithTerminalSvc(t)()

	initDB, cleanup := testDBCreateCommand(t, "test-cat-nohist-18", "test-cmd-nohist-18", "TestNoHist", "Test Cmd NoHist", "echo hello", `{}`)
	defer cleanup()

	_ = initDB.ClearExecutions()

	svc := &ExecutionService{}
	svc.RunCommand("test-cmd-nohist-18", nil)

	records, err := initDB.GetExecutions()
	if err != nil {
		t.Fatalf("GetExecutions failed: %v", err)
	}
	if len(records) > 0 {
		t.Errorf("expected 0 execution records after RunCommand, got %d", len(records))
	}
}

func TestRunCommand_NilTerminalSvc(t *testing.T) {
	prevTerminalSvc := terminalSvc
	terminalSvc = nil
	defer func() { terminalSvc = prevTerminalSvc }()

	_, cleanup := testDBCreateCommand(t, "test-cat-nilsvc-24", "test-cmd-nilsvc-24", "TestNilSvc", "Test Cmd NilSvc", "echo hi", `{}`)
	defer cleanup()

	svc := &ExecutionService{}
	record := svc.RunCommand("test-cmd-nilsvc-24", nil)
	if record.Error == "" {
		t.Error("expected Error to be set when terminalSvc is nil, got empty")
	}
	if record.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", record.ExitCode)
	}
}

func TestRunCommand_NoActiveSession(t *testing.T) {
	defer testWithTerminalSvc(t)()

	// testWithTerminalSvc creates a default session. To exercise the
	// "no active" path, clear the activeSessionID on the service.
	if terminalSvc == nil {
		t.Skip("terminalSvc not initialized")
	}
	terminalSvc.mu.Lock()
	terminalSvc.activeSessionID = ""
	terminalSvc.mu.Unlock()

	_, cleanup := testDBCreateCommand(t, "test-cat-noact-24", "test-cmd-noact-24", "TestNoAct", "Test Cmd NoAct", "echo hi", `{}`)
	defer cleanup()

	svc := &ExecutionService{}
	record := svc.RunCommand("test-cmd-noact-24", nil)
	if record.Error == "" {
		t.Error("expected Error to be set when no active session, got empty")
	}
	if !strings.Contains(record.Error, "no active") {
		t.Errorf("Error = %q, want it to contain 'no active'", record.Error)
	}
	if record.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", record.ExitCode)
	}
}

func TestRunCommand_ExecutesOnActiveSession(t *testing.T) {
	defer testWithTerminalSvc(t)()

	// testWithTerminalSvc creates a default session and makes it active.
	_, cleanup := testDBCreateCommand(t, "test-cat-exec-24", "test-cmd-exec-24", "TestExec", "Test Cmd Exec", "true", `{}`)
	defer cleanup()

	svc := &ExecutionService{}
	record := svc.RunCommand("test-cmd-exec-24", nil)
	if record.Error != "" {
		t.Errorf("Error = %q, want empty (happy path)", record.Error)
	}
	if record.FinalCmd != "true\n" {
		t.Errorf("FinalCmd = %q, want %q", record.FinalCmd, "true\n")
	}
}

func TestShellQuoteDir(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"simple path", "/Users/test", "'/Users/test'"},
		{"path with spaces", "/Users/My Folder", "'/Users/My Folder'"},
		{"path with quote", "/Users/O'Brien", "'/Users/O'\"'\"'Brien'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuoteDir(tt.dir)
			if got != tt.want {
				t.Errorf("shellQuoteDir(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

// TestToTerminalInput pins the regression that a command streamed into the
// terminal sat at the prompt without running on Windows: the ConPTY input
// parser submits a line on CR, the byte xterm.js sends for Enter, and ignores
// a bare LF.
func TestToTerminalInput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"single line", "echo hi\n", "echo hi\r"},
		{"multiline", "line1\nline2\n", "line1\rline2\r"},
		{"crlf collapses to one cr", "echo hi\r\n", "echo hi\r"},
		{"mixed endings", "line1\r\nline2\n", "line1\rline2\r"},
		{"already cr", "echo hi\r", "echo hi\r"},
		{"no trailing newline", "echo hi", "echo hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toTerminalInput(tt.in)
			if got != tt.want {
				t.Errorf("toTerminalInput(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestPrefixWorkingDir pins the per-shell cd syntax. The POSIX form is not
// portable: cmd.exe treats single quotes as literal characters, and Windows
// PowerShell 5.1 rejects `&&` outright, so a command with a working directory
// would fail before running on a machine without pwsh installed.
func TestPrefixWorkingDir(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		dir   string
		want  string
	}{
		{"bash", "/bin/bash", "/Users/test", "cd '/Users/test' && echo hi"},
		{"zsh with quote in path", "/bin/zsh", "/Users/O'Brien", `cd '/Users/O'"'"'Brien' && echo hi`},
		{
			"powershell 7",
			`C:\Program Files\PowerShell\7\pwsh.exe`,
			`C:\Users\test`,
			`Set-Location -LiteralPath 'C:\Users\test'; echo hi`,
		},
		{
			"windows powershell 5.1",
			`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
			`C:\Program Files`,
			`Set-Location -LiteralPath 'C:\Program Files'; echo hi`,
		},
		{
			"powershell path with quote",
			"pwsh",
			`C:\Users\O'Brien`,
			`Set-Location -LiteralPath 'C:\Users\O''Brien'; echo hi`,
		},
		{"cmd", `C:\Windows\System32\cmd.exe`, `D:\work`, `cd /d "D:\work" && echo hi`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prefixWorkingDir(tt.shell, tt.dir, "echo hi")
			if got != tt.want {
				t.Errorf("prefixWorkingDir(%q, %q, ...) = %q, want %q", tt.shell, tt.dir, got, tt.want)
			}
		})
	}
}

func TestTerminalService_ServiceStartupAssignsTerminalSvc(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	prevTerminalSvc := terminalSvc
	terminalSvc = nil
	defer func() { terminalSvc = prevTerminalSvc }()

	s := &TerminalService{}
	_ = s.ServiceStartup(nil, application.ServiceOptions{})
	defer s.ServiceShutdown()

	if terminalSvc == nil {
		t.Error("terminalSvc should be non-nil after ServiceStartup, got nil")
	}

	s.mu.RLock()
	count := len(s.sessions)
	s.mu.RUnlock()

	if count != 1 {
		t.Errorf("expected 1 default session after ServiceStartup, got %d", count)
	}

	if s.sessionCounter != 1 {
		t.Errorf("expected sessionCounter=1 after ServiceStartup, got %d", s.sessionCounter)
	}
}
