import { expect, test } from '@grafana/plugin-e2e';

const DS_UID = 'loki-e2e';

function exploreUrl(expr: string, editorMode = 'code'): string {
  const panes = JSON.stringify({
    a: {
      datasource: DS_UID,
      queries: [
        {
          refId: 'A',
          expr,
          queryType: 'range',
          editorMode,
          datasource: { type: 'loki', uid: DS_UID },
        },
      ],
      range: { from: 'now-3h', to: 'now' },
    },
  });
  return `/explore?orgId=1&schemaVersion=1&panes=${encodeURIComponent(panes)}`;
}

test.describe('Query editor', () => {
  test.describe.configure({ mode: 'serial' });

  test.describe('rendering', () => {
    test(
      'smoke: renders Builder and Code mode options',
      { tag: '@plugins' },
      async ({ page }) => {
        await page.goto(exploreUrl(''));

        await expect(page.getByRole('radio', { name: 'Builder' })).toBeVisible();
        await expect(page.getByRole('radio', { name: 'Code' })).toBeVisible();
      }
    );

    test('Code mode shows the LogQL editor', async ({ page }) => {
      await page.goto(exploreUrl('', 'code'));

      await expect(page.getByRole('radio', { name: 'Code' })).toBeChecked();
      await expect(
        page.getByRole('textbox', { name: 'Editor content;Press Alt+F1 for Accessibility Options.' })
      ).toBeVisible();
    });

    test('Builder mode shows Label filters UI', async ({ page }) => {
      await page.goto(exploreUrl('', 'builder'));

      await expect(page.getByRole('radio', { name: 'Builder' })).toBeChecked();
      await expect(page.getByText('Label filters')).toBeVisible();
      // The label selector combobox has aria-describedby="Select label" (not aria-label),
      // so we assert the label text is visible rather than querying by name.
      await expect(page.getByText('Select label')).toBeVisible();
    });
  });

  test.describe('Code mode', () => {
    test('can type a LogQL expression into the editor', async ({ page }) => {
      await page.goto(exploreUrl('', 'code'));

      await expect(page.getByRole('radio', { name: 'Code' })).toBeChecked();

      const editor = page.getByRole('textbox', {
        name: 'Editor content;Press Alt+F1 for Accessibility Options.',
      });
      await editor.click();
      await page.keyboard.press('ControlOrMeta+a');
      await page.keyboard.type('{job="e2e-test"}');

      await expect(editor).toHaveValue('{job="e2e-test"}');
    });

    test('switching from Builder to Code preserves mode after confirmation', async ({ page }) => {
      await page.goto(exploreUrl('', 'builder'));
      await expect(page.getByRole('radio', { name: 'Builder' })).toBeChecked();

      await page.getByRole('radio', { name: 'Code' }).click();

      // Handle optional "Cannot convert" confirmation dialog
      const continueBtn = page.getByRole('button', { name: 'Continue' });
      if (await continueBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
        await continueBtn.click();
      }

      await expect(page.getByRole('radio', { name: 'Code' })).toBeChecked();
    });
  });

  test.describe('Builder mode', () => {
    test('shows line filter input for filtering log content', async ({ page }) => {
      await page.goto(exploreUrl('', 'builder'));

      await expect(page.getByRole('radio', { name: 'Builder' })).toBeChecked();
      await expect(page.getByText('Label filters')).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Find log lines that contains this text' })).toBeVisible();
    });
  });
});

test.describe('Query editor with fixture data', () => {
  test.describe.configure({ mode: 'serial' });

  test('Code mode: log stream query returns results', async ({ page }) => {
    const responsePromise = page.waitForResponse(
      async (resp) => {
        if (!resp.url().includes('/api/ds/query')) {
          return false;
        }
        try {
          const body = await resp.json();
          return body?.results?.A !== undefined;
        } catch {
          return false;
        }
      },
      { timeout: 15000 }
    );

    await page.goto(exploreUrl('{job="e2e-test"}'));
    const response = await responsePromise;
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.results.A.frames).toBeDefined();
  });

  test('Code mode: filtered query by level returns results', async ({ page }) => {
    const responsePromise = page.waitForResponse(
      async (resp) => {
        if (!resp.url().includes('/api/ds/query')) {
          return false;
        }
        try {
          const body = await resp.json();
          return body?.results?.A !== undefined;
        } catch {
          return false;
        }
      },
      { timeout: 15000 }
    );

    await page.goto(exploreUrl('{job="e2e-test", level="error"}'));
    const response = await responsePromise;
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.results.A.frames).toBeDefined();
  });

  test('Code mode: metric query (rate) returns results', async ({ page }) => {
    const responsePromise = page.waitForResponse(
      async (resp) => {
        if (!resp.url().includes('/api/ds/query')) {
          return false;
        }
        try {
          const body = await resp.json();
          return body?.results?.A !== undefined;
        } catch {
          return false;
        }
      },
      { timeout: 15000 }
    );

    await page.goto(exploreUrl('rate({job="e2e-test"}[5m])'));
    const response = await responsePromise;
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.results.A.frames).toBeDefined();
  });
});
