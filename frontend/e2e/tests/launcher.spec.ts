import { test, expect, type Page } from '../fixtures';

function command(id: string, title: string, variables: unknown[] = []) {
  const now = new Date().toISOString();
  return {
    id,
    title: { String: title, Valid: true },
    description: { String: '', Valid: false },
    scriptContent: 'echo hello',
    tags: [],
    variables,
    presets: [],
    workingDir: {},
    categoryId: '',
    position: 0,
    createdAt: now,
    updatedAt: now,
  };
}

async function openLauncher(page: Page) {
  await page.goto('/?window=launcher');
  await expect(page.locator('.launcher-root')).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => window.__cmdexE2E!.hasListener('launcher-shown')))
    .toBe(true);
}

test.describe('Launcher', () => {
  test('the mock preserves visibility and emits the production launcher event contract', async ({ page }) => {
    await openLauncher(page);

    await page.evaluate(() => window.__cmdexE2E!.invokeLauncher('Show'));
    await page.evaluate(() => window.__cmdexE2E!.invokeLauncher('Toggle'));
    await page.evaluate(() => window.__cmdexE2E!.invokeLauncher('Toggle'));

    const state = await page.evaluate(() => ({
      visible: window.__cmdexE2E!.launcherVisible,
      events: window.__cmdexE2E!.launcherEventLog.map((event) => ({
        name: event.name,
        hasPayload: event.data !== undefined,
      })),
    }));
    expect(state.visible).toBe(true);
    expect(state.events).toEqual([
      { name: 'launcher-shown', hasPayload: false },
      { name: 'launcher-hidden', hasPayload: false },
      { name: 'launcher-shown', hasPayload: false },
    ]);
  });

  test('showing the persistent launcher resets an in-progress variable prompt', async ({ page, seed }) => {
    await seed({
      commands: [command('launcher-vars', 'Needs variables', [{ name: 'name', description: '', default: '' }])],
    });
    await openLauncher(page);

    await page.getByRole('option', { name: /Needs variables/ }).click();
    await expect(page.locator('[data-testid="fill-variables-dialog"]')).toBeVisible();

    await page.evaluate(() => window.__cmdexE2E!.invokeLauncher('Show'));

    await expect(page.locator('[data-testid="fill-variables-dialog"]')).toHaveCount(0);
    await expect(page.locator('.launcher-results')).toBeVisible();
    await expect(page.getByRole('option', { name: /Needs variables/ })).toBeVisible();
  });

  test('showing the persistent launcher resets the running output stage', async ({ page, seed }) => {
    await seed({ commands: [command('launcher-run', 'Run me')] });
    await openLauncher(page);

    await page.getByRole('option', { name: /Run me/ }).click();
    await expect(page.locator('.launcher-run-panel')).toBeVisible();

    await page.evaluate(() => window.__cmdexE2E!.invokeLauncher('Show'));

    await expect(page.locator('.launcher-run-panel')).toHaveCount(0);
    await expect(page.locator('.launcher-results')).toBeVisible();
    await expect(page.getByRole('option', { name: /Run me/ })).toBeVisible();
  });
});
