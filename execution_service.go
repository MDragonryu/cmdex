package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ExecutionService handles running commands and execution history.
type ExecutionService struct{}

func (s *ExecutionService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return nil
}

// resolveWorkingDir determines the working directory for a command using the fallback chain:
// 1. Command-specific working dir for the current OS
// 2. Global default working dir for the current OS
// 3. OS home directory
// 4. Current working directory
// 5. OS temporary directory
// The function never returns an empty string.
func (s *ExecutionService) resolveWorkingDir(cmd Command) string {
	// Step 1: use per-command working directory if set
	if path := cmd.WorkingDir.GetCurrentOS(); path != "" {
		return path
	}

	// Step 2: fall back to global default working directory for current OS
	settings, err := db.GetSettings()
	if err != nil {
		fmt.Printf("resolveWorkingDir: GetSettings failed: %v\n", err)
	} else if settings.DefaultWorkingDir != nil {
		if path := settings.DefaultWorkingDir.GetCurrentOS(); path != "" {
			return path
		}
	}

	// Step 3: final fallback to user home directory
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		if err != nil {
			fmt.Printf("resolveWorkingDir: UserHomeDir failed: %v, trying Getwd\n", err)
		}
		cwd, err := os.Getwd()
		if err != nil || cwd == "" {
			if err != nil {
				fmt.Printf("resolveWorkingDir: Getwd failed: %v, falling back to TempDir\n", err)
			}
			return os.TempDir()
		}
		return cwd
	}
	return home
}

// GetVariables returns variable prompts for a command.
func (s *ExecutionService) GetVariables(commandID string) []VariablePrompt {
	cmd, err := db.GetCommand(commandID)
	if err != nil {
		return []VariablePrompt{}
	}

	if len(cmd.Variables) == 0 {
		return []VariablePrompt{}
	}

	evaluated := executor.EvalDefaults(cmd.Variables)

	var prompts []VariablePrompt
	for _, v := range cmd.Variables {
		p := VariablePrompt{
			Name:        v.Name,
			Description: v.Description,
			Example:     v.Example,
			DefaultExpr: v.Default,
		}
		if val, exists := evaluated[v.Name]; exists {
			p.DefaultValue = val
		}
		prompts = append(prompts, p)
	}
	if prompts == nil {
		prompts = []VariablePrompt{}
	}
	return prompts
}

// hasExplicitWorkingDir returns true when either the command or the global
// settings define a working directory for the current OS. If neither have
// one, the shell stays in its current (home) directory — no cd sandwich needed.
func (s *ExecutionService) hasExplicitWorkingDir(cmd Command) bool {
	if cmd.WorkingDir.GetCurrentOS() != "" {
		return true
	}
	settings, err := db.GetSettings()
	if err != nil {
		return false
	}
	if settings.DefaultWorkingDir != nil {
		return settings.DefaultWorkingDir.GetCurrentOS() != ""
	}
	return false
}

// failedExecution builds an ExecutionRecord describing a failure to run.
func failedExecution(commandID string, err error) ExecutionRecord {
	return ExecutionRecord{
		ID:        uuid.New().String(),
		CommandID: commandID,
		Error:     err.Error(),
		ExitCode:  -1,
	}
}

// RunCommand resolves the command's template variables and writes the
// resulting command line directly to the active terminal session's PTY
// via TerminalService.Write. Output streams back through the session's
// pty-output event (handled by Terminal.tsx) and Ctrl+C interrupts are
// handled by the PTY's foreground process group.
func (s *ExecutionService) RunCommand(commandID string, variables map[string]string) ExecutionRecord {
	if terminalSvc == nil {
		return failedExecution(commandID, errors.New("terminal service not initialized"))
	}
	session := terminalSvc.GetActiveSession()
	if session == nil {
		return failedExecution(commandID, errors.New("no active terminal session"))
	}
	return s.RunCommandInSession(commandID, variables, session.ID)
}

// RunCommandInSession is RunCommand targeted at an explicit terminal session
// rather than whichever session the main window has focused. The global quick
// launcher uses it to run commands in its own dedicated session so its output
// stays self-contained, while sharing this identical resolution path.
func (s *ExecutionService) RunCommandInSession(commandID string, variables map[string]string, sessionID string) ExecutionRecord {
	if terminalSvc == nil {
		return failedExecution(commandID, errors.New("terminal service not initialized"))
	}
	if sessionID == "" {
		return failedExecution(commandID, errors.New("no terminal session specified"))
	}

	cmd, err := db.GetCommand(commandID)
	if err != nil {
		return failedExecution(commandID, err)
	}

	resolvedScript := ReplaceTemplateVars(cmd.ScriptContent, variables)
	resolvedScript = stripShebang(resolvedScript)
	resolvedScript = strings.TrimRight(resolvedScript, "\n")
	workingDir := s.resolveWorkingDir(cmd)

	var cmdLine string
	if s.hasExplicitWorkingDir(cmd) {
		// Sessions are all spawned from detectShell, so it names the shell that
		// will parse this line — the cd syntax has to match it.
		shellPath, _ := detectShell()
		cmdLine = prefixWorkingDir(shellPath, workingDir, resolvedScript) + "\n"
	} else {
		cmdLine = resolvedScript + "\n"
	}

	if err := terminalSvc.Write(sessionID, cmdLine); err != nil {
		return failedExecution(commandID, err)
	}

	return ExecutionRecord{
		ID:         uuid.New().String(),
		CommandID:  commandID,
		FinalCmd:   cmdLine,
		WorkingDir: workingDir,
		ExecutedAt: time.Now(),
	}
}
