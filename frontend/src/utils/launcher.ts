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
