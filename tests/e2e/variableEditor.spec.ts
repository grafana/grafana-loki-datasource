import { readFile } from 'node:fs/promises';
import { join } from 'node:path';

import { expect, test } from '@grafana/plugin-e2e';

test.describe('Variable editor', () => {
  test('keeps a typed field name without pressing Enter and previews its values', async ({
    variableEditPage,
    page,
    readProvisionedDataSource,
  }) => {
    // Grafana versions that still bundle loki as a core plugin serve their own loki frontend
    // (this build loses the duplicate-plugin-id resolution), so its variable editor and the
    // Detected field values query type are not reachable there. Skip only on a verified
    // mismatch between the served bundle and this build; anything else is a real failure.
    const servedModule = await page.request.get('/public/plugins/loki/module.js');
    expect(servedModule.ok()).toBeTruthy();
    const localModule = await readFile(join(__dirname, '../../dist/module.js'), 'utf8');
    test.skip((await servedModule.text()) !== localModule, 'host Grafana is not serving this build');

    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await variableEditPage.setVariableType('Query');
    await variableEditPage.datasource.set(ds.name);

    await page.getByLabel('Query type', { exact: true }).click();
    await page.getByText('Detected field values', { exact: true }).click();

    // Type the field name and move on without pressing Enter: blur via an inert element,
    // then fill the LogQL query. The fixture logs carry code=200/500 (tests/e2e/fixtures/load.py).
    await page.getByLabel('Field', { exact: true }).click();
    await page.keyboard.type('code');
    await page.getByText('Query type', { exact: true }).click();

    await page.getByLabel('LogQL query', { exact: true }).click();
    await page.keyboard.type('{job="e2e-test"} | logfmt');
    await page.keyboard.press('Tab');
    await variableEditPage.runQuery();

    await expect(variableEditPage).toDisplayPreviews(['200', '500']);
  });
});
