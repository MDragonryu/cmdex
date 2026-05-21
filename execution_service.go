package main

import (
	"context"
	"fmt"
	"os"
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

// RunCommand writes the resolved command to the PTY terminal for
// in-terminal execution. No subprocess or temp-script is used;
// output appears via the existing pty-output event stream.
func (s *ExecutionService) RunCommand(commandID string, variables map[string]string) ExecutionRecord {
	cmd, err := db.GetCommand(commandID)
	if err != nil {
		return ExecutionRecord{
			ID:       uuid.New().String(),
			Error:    err.Error(),
			ExitCode: -1,
		}
	}

	resolvedScript := ReplaceTemplateVars(cmd.ScriptContent, variables)
	workingDir := s.resolveWorkingDir(cmd)

	var cmdLine string
	if s.hasExplicitWorkingDir(cmd) {
		cmdLine = fmt.Sprintf("cd %s && %s && cd ~\n", shellQuoteDir(workingDir), resolvedScript)
	} else {
		cmdLine = resolvedScript + "\n"
	}

	if err := terminalSvc.Write(cmdLine); err != nil {
		return ExecutionRecord{
			ID:         uuid.New().String(),
			CommandID:  commandID,
			FinalCmd:   cmdLine,
			Error:      fmt.Sprintf("terminal write failed: %v", err),
			ExitCode:   -1,
			ExecutedAt: time.Now(),
		}
	}

	return ExecutionRecord{
		ID:         uuid.New().String(),
		CommandID:  commandID,
		FinalCmd:   cmdLine,
		ExecutedAt: time.Now(),
	}
}

// RunInTerminal opens the command in the system terminal.
func (s *ExecutionService) RunInTerminal(commandID string, variables map[string]string) error {
	cmd, err := db.GetCommand(commandID)
	if err != nil {
		return err
	}

	resolvedScript := ReplaceTemplateVars(cmd.ScriptContent, variables)
	workingDir := s.resolveWorkingDir(cmd)

	settings, err := db.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}
	return executor.OpenInTerminal(settings.Terminal, resolvedScript, workingDir)
}

// GetExecutionHistory returns all past execution records.
func (s *ExecutionService) GetExecutionHistory() []ExecutionRecord {
	records, err := db.GetExecutions()
	if err != nil {
		fmt.Println("Error getting executions:", err)
		return []ExecutionRecord{}
	}
	return records
}

// ClearExecutionHistory deletes all execution history.
func (s *ExecutionService) ClearExecutionHistory() error {
	return db.ClearExecutions()
}
