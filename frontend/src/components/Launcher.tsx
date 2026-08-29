import React, {
  useState,
  useMemo,
  useRef,
  useEffect,
  useCallback,
} from 'react';
import { Events } from '@wailsio/runtime';
import { FileText, Search, X, Terminal as TerminalIcon, ArrowUpRight } from 'lucide-react';
import { toast } from 'sonner';

import {
  type Command,
  type Category,
  type VariablePrompt as VariablePromptType,
} from '../types';
import { getCommandDisplayTitle } from '../utils/tab';
import { filterCommands, scriptSnippet } from '../utils/commandSearch';
import { eventNames, initEventNames } from '../wails/events';
import { Kbd } from './ui/kbd';
import TerminalComponent, { type TerminalHandle } from './Terminal';
import VariablePrompt from './VariablePrompt';

import { GetCommands, GetCategories, GetPresets, SavePreset, UpdatePreset, DeletePreset } from '../../bindings/cmdex/commandservice';
import { GetVariables, RunCommandInSession } from '../../bindings/cmdex/executionservice';
import { Hide, Resize, GetSessionID, ShowMainWindow } from '../../bindings/cmdex/launcherservice';

/**
 * The global quick launcher.
 *
 * Runs in its own frameless, always-on-top window (`/?window=launcher`) that the
 * Go side shows and hides rather than creating and destroying, so this component
 * mounts once for the lifetime of the app. All per-open reset work happens in
 * response to the `launcher-shown` event instead of on mount.
 *
 * Commands execute in the launcher's own terminal session and stream into the
 * panel below the search field, keeping each invocation self-contained: run it,
 * read the output, press Escape, move on.
 */

type Stage = 'search' | 'variables' | 'running';

interface LauncherProps {
  theme: string;
}

const Launcher: React.FC<LauncherProps> = ({ theme }) => {
  const [commands, setCommands] = useState<Command[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);
  const [stage, setStage] = useState<Stage>('search');
  const [sessionId, setSessionId] = useState('');
  const [pendingCommand, setPendingCommand] = useState<Command | null>(null);
  const [variables, setVariables] = useState<VariablePromptType[]>([]);
  const [ranCommand, setRanCommand] = useState<Command | null>(null);
  const [eventsInitialized, setEventsInitialized] = useState(false);
  const activationRef = useRef(0);

  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<TerminalHandle>(null);

  const catMap = useMemo(() => {
    const m: Record<string, string> = {};
    categories.forEach((c) => { m[c.id] = c.name; });
    return m;
  }, [categories]);

  const filtered = useMemo(() => filterCommands(query, commands), [query, commands]);

  const loadData = useCallback(async () => {
    try {
      const [cmds, cats] = await Promise.all([GetCommands(), GetCategories()]);
      setCommands(cmds || []);
      setCategories(cats || []);
    } catch (err) {
      console.error('launcher: failed to load commands', err);
    }
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- async fetch; state is set from the promise callback
    loadData();
  }, [loadData]);

  // Reset selection whenever the result set changes.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setActiveIndex(0);
  }, [filtered]);

  // Keep the highlighted row in view.
  useEffect(() => {
    if (!listRef.current) return;
    const el = listRef.current.querySelector<HTMLElement>(`[data-idx="${activeIndex}"]`);
    el?.scrollIntoView({ block: 'nearest' });
  }, [activeIndex]);

  const focusSearch = useCallback(() => {
    // Two frames: the window is still being raised on the first one, and
    // focus() there is dropped by the compositor on some platforms.
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        const input = inputRef.current;
        if (!input) return;
        input.focus();
        input.select();
      });
    });
  }, []);

  // Every time the window is revealed: refresh the command list (it may have
  // changed in the main window), focus the field and select any existing text
  // so typing replaces the previous query.
  useEffect(() => {
    initEventNames().then(() => setEventsInitialized(true));
  }, []);

  useEffect(() => {
    if (!eventsInitialized) return;
    const cleanup = Events.On(eventNames.launcherShown, () => {
      loadData();
      focusSearch();
    });
    return () => cleanup();
  }, [eventsInitialized, loadData, focusSearch]);

  useEffect(() => {
    focusSearch();
  }, [focusSearch]);

  const close = useCallback(() => {
    activationRef.current += 1;
    Hide().catch(() => {});
  }, []);

  /** Collapse back to the bare search field, discarding the run panel. */
  const resetToSearch = useCallback(() => {
    activationRef.current += 1;
    setStage('search');
    setPendingCommand(null);
    setVariables([]);
    setRanCommand(null);
    Resize(false).catch(() => {});
    focusSearch();
  }, [focusSearch]);

  const execute = useCallback(async (
    cmd: Command,
    values: Record<string, string>,
    activation = activationRef.current,
  ) => {
    try {
      const id = sessionId || (await GetSessionID());
      if (activation !== activationRef.current) return;
      if (!sessionId) setSessionId(id);

      setStage('running');
      setRanCommand(cmd);
      await Resize(true).catch(() => {});
      if (activation !== activationRef.current) return;

      const record = await RunCommandInSession(cmd.id, values, id);
      if (record?.error) {
        toast.error(record.error);
      }
      // Focus the terminal so Ctrl+C and interactive prompts reach the PTY.
      requestAnimationFrame(() => terminalRef.current?.focus());
    } catch (err) {
      if (activation !== activationRef.current) return;
      toast.error('Failed to run command: ' + String(err));
      console.error('launcher: execute failed', err);
    }
  }, [sessionId]);

  /** Activate the selected result: prompt for variables, or run straight away. */
  const activate = useCallback(async (cmd: Command) => {
    const activation = activationRef.current + 1;
    activationRef.current = activation;
    try {
      const prompts = await GetVariables(cmd.id);
      if (activation !== activationRef.current) return;
      if (prompts && prompts.length > 0) {
        // Reuse the main window's variable entry UI rather than a second one.
        setPendingCommand(cmd);
        setVariables(prompts);
        setStage('variables');
        Resize(true).catch(() => {});
        return;
      }
      if (activation !== activationRef.current) return;
      await execute(cmd, {}, activation);
    } catch (err) {
      if (activation !== activationRef.current) return;
      toast.error('Failed to prepare command: ' + String(err));
      console.error('launcher: activate failed', err);
    }
  }, [execute]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, filtered.length - 1));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === 'Enter') {
        e.preventDefault();
        const cmd = filtered[activeIndex];
        if (cmd) activate(cmd);
      }
      // Escape is handled by the window capture listener below, for every
      // focus target rather than just this input.
    },
    [filtered, activeIndex, activate],
  );

  // Escape anywhere in the window, including while the terminal has focus.
  //
  // This must run in the capture phase: xterm installs its own keydown handler
  // and calls stopPropagation() for keys it consumes, Escape among them (it
  // forwards \x1b to the PTY). A bubble-phase listener would therefore never
  // see Escape once the terminal has focus. Capturing runs before xterm and
  // stops the event there, so Escape means "go back" in the launcher rather
  // than reaching the shell.
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      // Let the variable dialog handle its own cancel — don't swallow it here.
      if (stage === 'variables') return;
      e.preventDefault();
      e.stopPropagation();
      if (stage === 'running') resetToSearch();
      else close();
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [stage, resetToSearch, close]);

  const noop = useCallback(async () => {}, []);

  /** Re-read the presets for a command after one was added, renamed or removed. */
  const refreshPresets = useCallback(async (cmd: Command) => {
    const presets = await GetPresets(cmd.id);
    setPendingCommand({ ...cmd, presets: presets || [] });
  }, []);

  // Screen readers follow the highlighted row through aria-activedescendant:
  // focus stays in the search field while the arrow keys move the selection.
  const showResults = stage !== 'running';
  const activeOptionId =
    showResults && filtered[activeIndex] ? `launcher-option-${activeIndex}` : undefined;

  return (
    <div className="launcher-root">
      <div className="launcher-search">
        <Search size={16} className="launcher-search-icon" />
        <input
          ref={inputRef}
          className="launcher-input"
          placeholder="Search CmDex commands…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          autoComplete="off"
          spellCheck={false}
          role="combobox"
          aria-label="Search CmDex commands"
          aria-controls="launcher-results"
          aria-expanded={showResults}
          aria-autocomplete="list"
          aria-activedescendant={activeOptionId}
        />
        {query && (
          <button
            className="launcher-clear"
            onClick={() => { setQuery(''); focusSearch(); }}
            aria-label="Clear search"
          >
            <X size={13} />
          </button>
        )}
      </div>

      {stage === 'running' ? (
        <div className="launcher-run-panel">
          <div className="launcher-run-header">
            <TerminalIcon size={13} />
            <span className="launcher-run-title">
              {ranCommand ? getCommandDisplayTitle(ranCommand) : 'Running'}
            </span>
            <button className="launcher-run-back" onClick={resetToSearch}>
              Back to search
            </button>
          </div>
          <div className="launcher-terminal">
            {sessionId && (
              <TerminalComponent
                ref={terminalRef}
                isVisible
                theme={theme}
                sessionId={sessionId}
              />
            )}
          </div>
        </div>
      ) : (
        <div
          className="launcher-results"
          ref={listRef}
          id="launcher-results"
          // Only a listbox once it actually holds options — the empty state is
          // a message, not a selectable row.
          role={filtered.length > 0 ? 'listbox' : undefined}
          aria-label="Command results"
        >
          {filtered.length === 0 ? (
            <div className="launcher-empty">
              {commands.length === 0
                ? 'No commands saved yet.'
                : `No commands match "${query}"`}
            </div>
          ) : (
            filtered.map((cmd, i) => {
              const catName = cmd.categoryId ? catMap[cmd.categoryId] : null;
              const isActive = i === activeIndex;
              return (
                <div
                  key={cmd.id}
                  id={`launcher-option-${i}`}
                  data-idx={i}
                  role="option"
                  aria-selected={isActive}
                  className={`launcher-item${isActive ? ' active' : ''}`}
                  onMouseEnter={() => setActiveIndex(i)}
                  onClick={() => activate(cmd)}
                >
                  <FileText size={14} className="launcher-item-icon" />
                  <div className="launcher-item-body">
                    <span className="launcher-item-title">{getCommandDisplayTitle(cmd)}</span>
                    {cmd.scriptContent && (
                      <span className="launcher-item-script">{scriptSnippet(cmd.scriptContent)}</span>
                    )}
                  </div>
                  {catName && <span className="launcher-cat-badge">{catName}</span>}
                </div>
              );
            })
          )}
        </div>
      )}

      <div className="launcher-footer">
        {stage === 'running' ? (
          <>
            <span className="launcher-hint"><Kbd>Esc</Kbd> back to search</span>
            <span className="launcher-hint"><Kbd>Ctrl</Kbd><Kbd>C</Kbd> interrupt</span>
          </>
        ) : (
          <>
            <span className="launcher-hint"><Kbd>↑</Kbd><Kbd>↓</Kbd> navigate</span>
            <span className="launcher-hint"><Kbd>↩</Kbd> run</span>
            <span className="launcher-hint"><Kbd>Esc</Kbd> close</span>
            <span className="launcher-hint launcher-hint-count">
              {filtered.length} result{filtered.length !== 1 ? 's' : ''}
            </span>
          </>
        )}
        <button className="launcher-open-app" onClick={() => ShowMainWindow().catch(() => {})}>
          Open CmDex <ArrowUpRight size={12} />
        </button>
      </div>

      {stage === 'variables' && pendingCommand && (
        <VariablePrompt
          mode="fill"
          variables={variables}
          presets={pendingCommand.presets || []}
          onSubmit={(values) => {
            const cmd = pendingCommand;
            setPendingCommand(null);
            setVariables([]);
            if (cmd) execute(cmd, values);
          }}
          onCancel={resetToSearch}
          onSavePreset={async (name, values) => {
            if (!pendingCommand) return;
            try {
              await SavePreset(pendingCommand.id, name, values);
              await refreshPresets(pendingCommand);
            } catch (err) {
              toast.error('Failed to save preset: ' + String(err));
              console.error('launcher: save preset failed', err);
            }
          }}
          onUpdatePreset={async (presetId, name, values) => {
            if (!pendingCommand) return;
            try {
              await UpdatePreset(pendingCommand.id, presetId, name, values);
              await refreshPresets(pendingCommand);
            } catch (err) {
              toast.error('Failed to update preset: ' + String(err));
              console.error('launcher: update preset failed', err);
            }
          }}
          onDeletePreset={async (presetId) => {
            if (!pendingCommand) return;
            try {
              await DeletePreset(pendingCommand.id, presetId);
              await refreshPresets(pendingCommand);
            } catch (err) {
              toast.error('Failed to delete preset: ' + String(err));
              console.error('launcher: delete preset failed', err);
            }
          }}
          onPresetChange={noop}
        />
      )}
    </div>
  );
};

export default Launcher;
