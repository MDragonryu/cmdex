import { type Command, type VariablePreset } from '../types';

export type LauncherStage = 'search' | 'variables' | 'running';

export interface LauncherQueryTransition {
  query: string;
  stage: LauncherStage;
  clearRunState: boolean;
}

/**
 * Typing always belongs to command search. In particular, a persistent
 * launcher may still have its search input mounted while the terminal output
 * stage is visible; the first keystroke must leave that stage immediately.
 */
export function transitionLauncherQuery(stage: LauncherStage, query: string): LauncherQueryTransition {
  if (stage !== 'running') {
    return { query, stage, clearRunState: false };
  }
  return { query, stage: 'search', clearRunState: true };
}

/**
 * Apply an asynchronous preset refresh only while its command is still open.
 * A canceled prompt has no current command, and a replacement prompt has a
 * different ID; both must ignore the stale response.
 */
export function applyRefreshedPresets(
  current: Command | null,
  refreshedCommand: Command,
  presets: VariablePreset[],
): Command | null {
  return current?.id === refreshedCommand.id
    ? { ...current, presets }
    : current;
}
