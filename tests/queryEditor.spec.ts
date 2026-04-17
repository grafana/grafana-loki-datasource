import { expect, test } from '@grafana/plugin-e2e';

/**
 * Builds a Grafana Explore URL with the given datasource and query state.
 * Encoding the datasource in the URL means the query editor loads on navigation
 * without needing any datasource-picker interaction.
 */
function exploreUrl(datasourceUID: string, query: Record<string, unknown> = {}): string {
  const panes = {
    '000': {
      datasource: datasourceUID,
      queries: [
        {
          refId: 'A',
          datasource: { type: 'loki', uid: datasourceUID },
          ...query,
        },
      ],
      range: { from: 'now-1h', to: 'now' },
    },
  };
  return `/explore?orgId=1&schemaVersion=1&panes=${encodeURIComponent(JSON.stringify(panes))}`;
}

test.describe('Query editor', () => {
  test.beforeEach(async ({ readProvisionedDataSource, page }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await page.goto(exploreUrl(ds.uid));
  });

  test.describe('rendering', () => {
    test(
      'smoke: should render Builder and Code mode options',
      { tag: '@plugins' },
      async ({ page }) => {
        await expect(page.getByRole('radio', { name: 'Builder' })).toBeVisible();
        await expect(page.getByRole('radio', { name: 'Code' })).toBeVisible();
      }
    );

    test('should render query editor header buttons', async ({ page }) => {
      await expect(page.getByRole('button', { name: 'Kick start your query' })).toBeVisible();
      await expect(page.getByTestId('label-browser-button')).toBeVisible();
    });
  });

  test.describe('Code mode', () => {
    test('should show Monaco editor in Code mode', async ({ page }) => {
      await page.getByRole('radio', { name: 'Code' }).click();
      await expect(page.getByRole('radio', { name: 'Code' })).toBeChecked();
      // Loki uses Monaco for the LogQL editor; [role="code"] scopes to the editor area
      await expect(page.locator('[role="code"]')).toBeVisible();
    });

    test('should accept a LogQL expression in Code mode', async ({ page }) => {
      await page.getByRole('radio', { name: 'Code' }).click();
      await expect(page.getByRole('radio', { name: 'Code' })).toBeChecked();

      const editor = page.locator('[role="code"]');
      await editor.click();
      await page.keyboard.press('ControlOrMeta+a');
      await page.keyboard.type('{job="grafana"}');
      await expect(editor).toContainText('{job="grafana"}');
    });
  });

  test.describe('Builder mode', () => {
    test('should show visual query builder with label filters', async ({ page }) => {
      await page.getByRole('radio', { name: 'Builder' }).click();
      await expect(page.getByRole('radio', { name: 'Builder' })).toBeChecked();
      await expect(page.getByText('Label filters')).toBeVisible();
    });
  });
});
