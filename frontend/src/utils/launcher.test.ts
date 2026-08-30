import { describe, expect, it } from 'vitest';

import { transitionLauncherQuery } from './launcher';

describe('transitionLauncherQuery', () => {
  it('returns to search and clears execution state when typing over output', () => {
    expect(transitionLauncherQuery('running', 'next command')).toEqual({
      query: 'next command',
      stage: 'search',
      clearRunState: true,
    });
  });

  it('keeps variable prompts and search navigation intact while typing', () => {
    expect(transitionLauncherQuery('variables', 'typed value')).toEqual({
      query: 'typed value',
      stage: 'variables',
      clearRunState: false,
    });
    expect(transitionLauncherQuery('search', 'typed value')).toEqual({
      query: 'typed value',
      stage: 'search',
      clearRunState: false,
    });
  });
});
