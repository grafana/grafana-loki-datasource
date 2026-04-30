import { expect, test } from '@grafana/plugin-e2e';

import { type LokiOptions } from '../../src/types';

test.describe('Config editor', () => {
  test.describe.configure({ mode: 'serial' });

  test.describe('rendering', () => {
    test('smoke: should render config editor', { tag: '@plugins' }, async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'loki' });

      await expect(page.getByText('Type: Loki', { exact: true })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
    });

    test('should render Authentication section', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'loki' });

      await expect(page.getByRole('heading', { name: 'Authentication', exact: true })).toBeVisible();
      await expect(page.getByRole('combobox', { name: 'Authentication method' })).toBeVisible();
    });

    test('should render Additional settings with Queries and Alerting subsections', async ({
      createDataSourceConfigPage,
      page,
    }) => {
      await createDataSourceConfigPage({ type: 'loki' });

      await expect(page.getByRole('heading', { name: 'Additional settings', exact: true })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Queries', exact: true })).toBeVisible();
      await expect(page.getByRole('spinbutton', { name: 'Maximum lines' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Alerting', exact: true })).toBeVisible();
    });
  });

  test.describe('provisioned datasource', () => {
    test('should load provisioned URL', async ({ readProvisionedDataSource, gotoDataSourceConfigPage, page }) => {
      const ds = await readProvisionedDataSource<LokiOptions>({ fileName: 'datasources.yml' });
      await gotoDataSourceConfigPage(ds.uid);

      const expectedUrl = process.env.DS_INSTANCE_HOST
        ? `http://${process.env.DS_INSTANCE_HOST}:${process.env.DS_INSTANCE_PORT ?? '3100'}`
        : 'http://loki:3100';
      await expect(page.getByRole('textbox', { name: 'Data source connection URL' })).toHaveValue(expectedUrl);
    });

    test('should load provisioned Maximum lines value', async ({
      readProvisionedDataSource,
      gotoDataSourceConfigPage,
      page,
    }) => {
      const ds = await readProvisionedDataSource<LokiOptions>({ fileName: 'datasources.yml' });
      await gotoDataSourceConfigPage(ds.uid);

      await expect(page.getByRole('spinbutton', { name: 'Maximum lines' })).toHaveValue('1000');
    });
  });

  test.describe('save & test', () => {
    test('should pass health check for provisioned datasource', async ({
      readProvisionedDataSource,
      gotoDataSourceConfigPage,
      page,
    }) => {
      const ds = await readProvisionedDataSource<LokiOptions>({ fileName: 'datasources.yml' });
      const configPage = await gotoDataSourceConfigPage(ds.uid);

      await page.getByRole('button', { name: 'Save & test' }).click();
      await expect(configPage).toHaveAlert('success', { hasNotText: 'Error' });
    });

    test('should show error alert when backend is unreachable', async ({ createDataSourceConfigPage, page }) => {
      const configPage = await createDataSourceConfigPage({ type: 'loki' });

      await page.getByRole('textbox', { name: 'Data source connection URL' }).fill('http://localhost:1');
      await page.getByRole('button', { name: 'Save & test' }).click();
      await expect(configPage).toHaveAlert('error');
    });
  });
});
