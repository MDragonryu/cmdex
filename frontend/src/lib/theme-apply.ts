const CUSTOM_THEME_VAR_KEYS = [
  'background', 'foreground', 'card', 'card-foreground', 'popover', 'popover-foreground',
  'primary', 'primary-foreground', 'secondary', 'secondary-foreground', 'muted', 'muted-foreground',
  'accent', 'accent-foreground', 'destructive', 'destructive-foreground', 'success', 'success-foreground',
  'border', 'input', 'ring', 'tab-bar-bg', 'tab-active-bg', 'tab-inactive-bg',
  'tab-active-indicator', 'status-bar-bg', 'status-bar-fg',
];

export function applyTheme(themeId: string, customColors?: Record<string, string> | null) {
  document.documentElement.setAttribute('data-theme', themeId);
  // Always clear first: a custom palette may omit keys the previous one set,
  // and those would otherwise stay behind and mix two themes together.
  CUSTOM_THEME_VAR_KEYS.forEach((key) => {
    document.documentElement.style.removeProperty(`--${key}`);
  });
  if (customColors) {
    Object.entries(customColors).forEach(([key, value]) => {
      document.documentElement.style.setProperty(`--${key}`, value);
    });
  }
}

export function applyDensity(density: string) {
  document.documentElement.setAttribute('data-density', density);
}

export function applyFonts(uiFont: string, monoFont: string) {
  const fontValue = uiFont === 'System Default'
    ? 'system-ui, -apple-system, sans-serif'
    : `'${uiFont}', system-ui, sans-serif`;
  document.documentElement.style.setProperty('--font-sans', fontValue);
  document.documentElement.style.setProperty('--font-mono', "'" + monoFont + "', monospace");
}
