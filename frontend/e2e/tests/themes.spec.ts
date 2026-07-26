import { test, expect, type Page } from '@playwright/test';
import '../utils/types';

// A custom theme carries its colors in the settings payload; there is no
// `[data-theme="custom-..."]` CSS rule, so they only take effect when the app
// writes them as inline CSS variables on the root element.
const CUSTOM_THEME = {
  id: 'custom-test-theme',
  name: 'Test Custom',
  type: 'dark' as const,
  colors: {
    background: '#101014',
    foreground: '#e8e6e3',
    primary: '#d2691e',
    'status-bar-bg': '#1b1b21',
  },
};

function rootVar(page: Page, name: string): Promise<string> {
  return page.evaluate(
    (varName) => document.documentElement.style.getPropertyValue(varName),
    name,
  );
}

test.describe('Custom themes', () => {
  test('applies the saved custom theme colors on load', async ({ page }) => {
    await page.addInitScript((theme) => {
      window.__cmdexE2E_SEED__ = {
        settings: {
          theme: theme.id,
          customThemes: JSON.stringify([theme]),
        },
      };
    }, CUSTOM_THEME);
    await page.goto('/');
    await expect(page.locator('.sidebar')).toBeVisible();

    await expect.poll(() => rootVar(page, '--background')).toBe('#101014');
    expect(await rootVar(page, '--primary')).toBe('#d2691e');
    expect(await rootVar(page, '--status-bar-bg')).toBe('#1b1b21');
    expect(await page.getAttribute('html', 'data-theme')).toBe(CUSTOM_THEME.id);
  });

  test('applies a custom theme delivered via settings-changed', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.sidebar')).toBeVisible();
    expect(await rootVar(page, '--background')).toBe('');

    // The settings window emits this after an import; the payload arrives
    // wrapped in a WailsEvent, so the colors sit under `data`.
    await page.evaluate((theme) => {
      window.__cmdexE2E?.emit('settings-changed', {
        name: 'settings-changed',
        data: { theme: theme.id, customThemes: JSON.stringify([theme]) },
        sender: 'e2e',
      });
    }, CUSTOM_THEME);

    await expect.poll(() => rootVar(page, '--background')).toBe('#101014');
    expect(await rootVar(page, '--foreground')).toBe('#e8e6e3');
    expect(await page.getAttribute('html', 'data-theme')).toBe(CUSTOM_THEME.id);
  });

  test('clears custom colors when switching back to a built-in theme', async ({ page }) => {
    await page.addInitScript((theme) => {
      window.__cmdexE2E_SEED__ = {
        settings: {
          theme: theme.id,
          customThemes: JSON.stringify([theme]),
        },
      };
    }, CUSTOM_THEME);
    await page.goto('/');
    await expect.poll(() => rootVar(page, '--background')).toBe('#101014');

    await page.evaluate(() => {
      window.__cmdexE2E?.emit('settings-changed', {
        name: 'settings-changed',
        data: { theme: 'vscode-light', customThemes: '[]' },
        sender: 'e2e',
      });
    });

    await expect.poll(() => rootVar(page, '--background')).toBe('');
    expect(await rootVar(page, '--primary')).toBe('');
    expect(await page.getAttribute('html', 'data-theme')).toBe('vscode-light');
  });
});
