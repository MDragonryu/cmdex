package main

import (
	"runtime"
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

func TestRunCommand_FinalCmdWithWorkingDir(t *testing.T) {
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
