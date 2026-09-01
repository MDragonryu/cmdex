import { describe, expect, it } from 'vitest';

import { applyRefreshedPresets, transitionLauncherQuery } from './launcher';
import { type Command } from '../types';

function command(id: string): Command {
  return {
    id,
    title: { String: id, Valid: true },
    description: { String: '', Valid: false },
    scriptContent: '',
    tags: [],
    variables: [],
    presets: [],
    workingDir: {},
    categoryId: '',
    position: 0,
    createdAt: '',
    updatedAt: '',
  };
}

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

describe('applyRefreshedPresets', () => {
  const commandA = command('A');
  const presetA = [{ id: 'preset-a', name: 'A', position: 0, values: {} }];

  it('does not restore a canceled command from a stale refresh', () => {
    expect(applyRefreshedPresets(null, commandA, presetA)).toBeNull();
  });

  it('does not replace command B with a stale refresh for command A', () => {
    const commandB = command('B');
    expect(applyRefreshedPresets(commandB, commandA, presetA)).toBe(commandB);
  });

  it('refreshes presets when the pending command is still the same command', () => {
    expect(applyRefreshedPresets(commandA, commandA, presetA)).toEqual({
      ...commandA,
      presets: presetA,
    });
  });
});
