import { test, expect } from '@grafana/plugin-e2e';

import { type LokiOptions } from '../src/types';

test.describe('Config editor', () => {
  test.describe('rendering', () => {
    test('smoke: should render config editor', { tag: '@plugins' }, async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'loki' });
      await expect(page.getByText('Type: Loki', { exact: true })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
    });

    test('should render Loki URL field with placeholder', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'loki' });
      await expect(page.getByPlaceholder('http://localhost:3100')).toBeVisible();
    });

    test('should render Maximum lines field', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'loki' });
      await expect(page.locator('#loki_config_maxLines')).toBeVisible();
    });
  });

  test.describe('provisioned datasource', () => {
    test('should load provisioned datasource config page', async ({ readProvisionedDataSource, gotoDataSourceConfigPage, page }) => {
      const ds = await readProvisionedDataSource<LokiOptions>({ fileName: 'datasources.yml' });
      await gotoDataSourceConfigPage(ds.uid);
      await expect(page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
    });
  });

  test.describe('save & test', () => {
    test('should show error alert when backend is unreachable', async ({ createDataSourceConfigPage, page }) => {
      const configPage = await createDataSourceConfigPage({ type: 'loki' });
      await page.getByRole('textbox', { name: 'URL' }).fill('http://localhost:1');
      await expect(configPage.saveAndTest()).not.toBeOK();
      await expect(configPage).toHaveAlert('error');
    });
  });
});
