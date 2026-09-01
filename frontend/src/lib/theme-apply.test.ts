import { describe, expect, it } from 'vitest';

import { CUSTOM_THEME_VAR_KEYS, parseCustomThemes } from './theme-apply';

function colors() {
  return Object.fromEntries(CUSTOM_THEME_VAR_KEYS.map(key => [key, '#000'])) as Record<string, string>;
}

function theme(overrides: Record<string, unknown> = {}) {
  return {
    id: 'custom-test',
    name: 'Test theme',
    type: 'dark',
    colors: colors(),
    ...overrides,
  };
}

describe('parseCustomThemes', () => {
  it('keeps complete custom color maps', () => {
    expect(parseCustomThemes(JSON.stringify([theme()]))).toHaveLength(1);
  });

  it('rejects incomplete or invalid custom color maps', () => {
    const incomplete = colors();
    delete incomplete.background;
    const invalidValue = { ...colors(), foreground: null };

    expect(parseCustomThemes(JSON.stringify([
      theme({ colors: incomplete }),
      theme({ colors: invalidValue }),
      null,
    ]))).toEqual([]);
  });
});
