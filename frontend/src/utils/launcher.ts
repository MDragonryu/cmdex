import { type Command, type VariablePreset } from '../types';

export type LauncherStage = 'search' | 'variables' | 'running';

export interface LauncherQueryTransition {
  query: string;
  stage: LauncherStage;
  clearRunState: boolean;
}

/** Whether a completed launcher data request may update component state. */
export function isCurrentLauncherRequest(request: number, currentRequest: number): boolean {
  return request === currentRequest;
}

/**
 * Serialize launcher resize effects and coalesce requests that have not
 * started yet. A resize already in flight is followed by the latest request,
 * so an old completion cannot leave the persistent launcher at a stale size.
 */
export function createLauncherResizeQueue(resize: (expanded: boolean) => Promise<void>) {
  let tail: Promise<void> = Promise.resolve();
  let generation = 0;

  return {
    enqueue(expanded: boolean): Promise<void> {
      const request = ++generation;
      const operation = tail
        .catch(() => {})
        .then(async () => {
          if (request !== generation) return;
          await resize(expanded);
        });
      tail = operation.catch(() => {});
      return operation;
    },
  };
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
