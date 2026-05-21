package main

import (
	"io"
	"os"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// testCleanup removes test data from the shared DB after the test.
func testCleanup(t *testing.T, dbConn *DB, catID, cmdID string) {
	t.Helper()
	dbConn.conn.Exec(`DELETE FROM commands WHERE id = ?`, cmdID)
	dbConn.conn.Exec(`DELETE FROM categories WHERE id = ?`, catID)
}

// TestRunCommand_PTYWriteWithWorkingDir verifies that RunCommand writes
// a cd-sandwich-wrapped command line to the PTY when the command has
// a working directory set.
func TestRunCommand_PTYWriteWithWorkingDir(t *testing.T) {
	// Set up a pipe to capture what RunCommand writes to PTY.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	// Swap in a TerminalService whose ptmx is our pipe writer.
	prevTerminalSvc := terminalSvc
	terminalSvc = &TerminalService{ptmx: w}
	defer func() { terminalSvc = prevTerminalSvc }()

	// Set up a known command in the database with a working directory.
	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	defer initDB.Close()

	catID := "test-cat-wd-18"
	cmdID := "test-cmd-wd-18"

	// Clean up any previous test data.
	testCleanup(t, initDB, catID, cmdID)

	_, err = initDB.conn.Exec(`INSERT INTO categories (id, name, icon, color) VALUES (?, ?, '', '')`, catID, "TestWD")
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}

	workingDirJSON := `{"darwin":"/Users/test"}`
	_, err = initDB.conn.Exec(
		`INSERT INTO commands (id, category_id, title, script_content, working_dir, position) VALUES (?, ?, ?, ?, ?, 0)`,
		cmdID, catID, "Test Cmd WD", GenerateScript("echo hello"), workingDirJSON,
	)
	if err != nil {
		t.Fatalf("insert command: %v", err)
	}

	prevDB := db
	db = initDB
	defer func() { db = prevDB }()

	svc := &ExecutionService{}
	record := svc.RunCommand(cmdID, nil)

	// Read what was written to the PTY pipe.
	w.Close() // signal EOF so ReadAll completes
	written, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading PTY pipe: %v", err)
	}

	expected := "cd '/Users/test' && echo hello && cd ~\n"
	if string(written) != expected {
		t.Errorf("RunCommand wrote %q, want %q", string(written), expected)
	}

	// Verify returned ExecutionRecord.
	if record.CommandID != cmdID {
		t.Errorf("CommandID = %q, want %q", record.CommandID, cmdID)
	}
	if record.Error != "" {
		t.Errorf("Error = %q, want empty", record.Error)
	}
}

// TestRunCommand_PTYWriteNoWorkingDir verifies that RunCommand writes
// just the resolved script (no cd sandwich) when the command has no
// explicit working dir set and no global default.
func TestRunCommand_PTYWriteNoWorkingDir(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	prevTerminalSvc := terminalSvc
	terminalSvc = &TerminalService{ptmx: w}
	defer func() { terminalSvc = prevTerminalSvc }()

	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	defer initDB.Close()

	// Clear the global default working dir for a clean test baseline.
	settings, err := initDB.GetSettings()
	if err == nil {
		settings.DefaultWorkingDir = &OSPathMap{}
		_ = initDB.SetSettings(settings)
	}

	catID := "test-cat-nowd-18"
	cmdID := "test-cmd-nowd-18"
	testCleanup(t, initDB, catID, cmdID)

	_, err = initDB.conn.Exec(`INSERT INTO categories (id, name, icon, color) VALUES (?, ?, '', '')`, catID, "TestNoWD")
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}

	// Command with no working dir (empty OSPathMap).
	_, err = initDB.conn.Exec(
		`INSERT INTO commands (id, category_id, title, script_content, working_dir, position) VALUES (?, ?, ?, ?, '{}', 0)`,
		cmdID, catID, "Test Cmd NoWD", GenerateScript("echo hello"),
	)
	if err != nil {
		t.Fatalf("insert command: %v", err)
	}

	prevDB := db
	db = initDB
	defer func() { db = prevDB }()

	svc := &ExecutionService{}
	record := svc.RunCommand(cmdID, nil)

	w.Close()
	written, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading PTY pipe: %v", err)
	}

	expected := "echo hello\n"
	if string(written) != expected {
		t.Errorf("RunCommand wrote %q, want %q", string(written), expected)
	}

	if record.CommandID != cmdID {
		t.Errorf("CommandID = %q, want %q", record.CommandID, cmdID)
	}
	if record.Error != "" {
		t.Errorf("Error = %q, want empty", record.Error)
	}
}

// TestRunCommand_PTYWriteMultilineScript verifies that RunCommand preserves
// internal newlines in multi-line scripts while stripping only the trailing
// newline from the cd sandwich.
func TestRunCommand_PTYWriteMultilineScript(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	prevTerminalSvc := terminalSvc
	terminalSvc = &TerminalService{ptmx: w}
	defer func() { terminalSvc = prevTerminalSvc }()

	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	defer initDB.Close()

	catID := "test-cat-ml-18"
	cmdID := "test-cmd-ml-18"
	testCleanup(t, initDB, catID, cmdID)

	_, err = initDB.conn.Exec(`INSERT INTO categories (id, name, icon, color) VALUES (?, ?, '', '')`, catID, "TestML")
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}

	workingDirJSON := `{"darwin":"/Users/test"}`
	_, err = initDB.conn.Exec(
		`INSERT INTO commands (id, category_id, title, script_content, working_dir, position) VALUES (?, ?, ?, ?, ?, 0)`,
		cmdID, catID, "Test Cmd ML", GenerateScript("line1\nline2"), workingDirJSON,
	)
	if err != nil {
		t.Fatalf("insert command: %v", err)
	}

	prevDB := db
	db = initDB
	defer func() { db = prevDB }()

	svc := &ExecutionService{}
	record := svc.RunCommand(cmdID, nil)

	w.Close()
	written, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading PTY pipe: %v", err)
	}

	expected := "cd '/Users/test' && line1\nline2 && cd ~\n"
	if string(written) != expected {
		t.Errorf("RunCommand wrote %q, want %q", string(written), expected)
	}

	if record.CommandID != cmdID {
		t.Errorf("CommandID = %q, want %q", record.CommandID, cmdID)
	}
	if record.Error != "" {
		t.Errorf("Error = %q, want empty", record.Error)
	}
}

// TestRunCommand_GetCommandError verifies that RunCommand returns
// an ExecutionRecord with an Error field when db.GetCommand fails.
func TestRunCommand_GetCommandError(t *testing.T) {
	// Set up a real DB but query for a nonexistent command.
	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	defer initDB.Close()

	prevTerminalSvc := terminalSvc
	r, w, _ := os.Pipe()
	terminalSvc = &TerminalService{ptmx: w}
	defer func() { terminalSvc = prevTerminalSvc; w.Close(); r.Close() }()

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

// TestRunCommand_WriteError verifies that RunCommand returns an
// ExecutionRecord with an Error field when terminalSvc.Write fails.
func TestRunCommand_WriteError(t *testing.T) {
	// terminal not started: ptmx is nil so Write returns error.
	prevTerminalSvc := terminalSvc
	terminalSvc = &TerminalService{ptmx: nil}
	defer func() { terminalSvc = prevTerminalSvc }()

	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	defer initDB.Close()

	catID := "test-cat-writeerr-18"
	cmdID := "test-cmd-writeerr-18"
	testCleanup(t, initDB, catID, cmdID)

	_, err = initDB.conn.Exec(`INSERT INTO categories (id, name, icon, color) VALUES (?, ?, '', '')`, catID, "TestWriteErr")
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}

	_, err = initDB.conn.Exec(
		`INSERT INTO commands (id, category_id, title, script_content, working_dir, position) VALUES (?, ?, ?, ?, '{}', 0)`,
		cmdID, catID, "Test Cmd WriteErr", "echo hello",
	)
	if err != nil {
		t.Fatalf("insert command: %v", err)
	}

	prevDB := db
	db = initDB
	defer func() { db = prevDB }()

	svc := &ExecutionService{}
	record := svc.RunCommand(cmdID, nil)

	if record.Error == "" {
		t.Error("expected Error field when Write fails, got empty string")
	}
	if record.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", record.ExitCode)
	}
}

// TestTerminalService_ServiceStartupAssignsTerminalSvc verifies that
// ServiceStartup sets the package-level terminalSvc variable.
func TestTerminalService_ServiceStartupAssignsTerminalSvc(t *testing.T) {
	prevTerminalSvc := terminalSvc
	terminalSvc = nil
	defer func() { terminalSvc = prevTerminalSvc }()

	s := &TerminalService{}
	_ = s.ServiceStartup(nil, application.ServiceOptions{})
	defer s.Stop()

	if terminalSvc == nil {
		t.Error("terminalSvc should be non-nil after ServiceStartup, got nil")
	}
}

// TestShellQuoteDir verifies shellQuoteDir behavior for
// paths with and without single quotes.
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

// TestRunCommand_NoHistoryPersistence verifies that RunCommand does NOT
// call db.AddExecution.
func TestRunCommand_NoHistoryPersistence(t *testing.T) {
	initDB, err := NewDB()
	if err != nil {
		t.Skipf("cannot open test DB: %v", err)
	}
	defer initDB.Close()

	_ = initDB.ClearExecutions()

	catID := "test-cat-nohist-18"
	cmdID := "test-cmd-nohist-18"
	testCleanup(t, initDB, catID, cmdID)

	_, err = initDB.conn.Exec(`INSERT INTO categories (id, name, icon, color) VALUES (?, ?, '', '')`, catID, "TestNoHist")
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}

	_, err = initDB.conn.Exec(
		`INSERT INTO commands (id, category_id, title, script_content, working_dir, position) VALUES (?, ?, ?, ?, '{}', 0)`,
		cmdID, catID, "Test Cmd NoHist", "echo hello",
	)
	if err != nil {
		t.Fatalf("insert command: %v", err)
	}

	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()

	prevTerminalSvc := terminalSvc
	terminalSvc = &TerminalService{ptmx: w}
	defer func() { terminalSvc = prevTerminalSvc }()

	prevDB := db
	db = initDB
	defer func() { db = prevDB }()

	svc := &ExecutionService{}
	svc.RunCommand(cmdID, nil)

	records, err := initDB.GetExecutions()
	if err != nil {
		t.Fatalf("GetExecutions failed: %v", err)
	}
	if len(records) > 0 {
		t.Errorf("expected 0 execution records after RunCommand, got %d", len(records))
	}
}
