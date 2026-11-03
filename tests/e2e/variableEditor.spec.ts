import { expect, test } from '@grafana/plugin-e2e';

import { isCloudRun, resolveDataSourceUid } from './env';

test.describe('Variable editor', () => {
  test('keeps a typed field name without pressing Enter and previews its values', async ({
    variableEditPage,
    page,
  }) => {
    const dsResponse = await page.request.get(`/api/datasources/uid/${await resolveDataSourceUid(page)}`);
    expect(dsResponse.ok()).toBeTruthy();
    const { name: dsName } = await dsResponse.json();

    await variableEditPage.setVariableType('Query');
    await variableEditPage.datasource.set(dsName);

    // Grafana versions that still bundle loki as a core plugin serve their own loki frontend,
    // and a Cloud instance may run a plugin release older than this feature. Neither offers
    // the Detected field values query type, so skip when the option is missing.
    await page.getByLabel('Query type', { exact: true }).click();
    await page.getByRole('option', { name: 'Label values', exact: true }).waitFor();
    const detectedFieldValues = page.getByRole('option', { name: 'Detected field values', exact: true });
    test.skip((await detectedFieldValues.count()) === 0, 'served loki plugin has no Detected field values query type');
    await detectedFieldValues.click();

    // Type the field name and move on without pressing Enter: blur via an inert element,
    // then fill the LogQL query.
    await page.getByLabel('Field', { exact: true }).click();
    await page.keyboard.type('code');
    await page.getByText('Query type', { exact: true }).click();

    await page.getByLabel('LogQL query', { exact: true }).click();
    await page.keyboard.type('{job="e2e-test"} | logfmt');
    await page.keyboard.press('Tab');
    await variableEditPage.runQuery();

    await expect(page.getByLabel('Field', { exact: true })).toHaveValue('code');

    // The code=200/500 fixture logs are pushed only into the local Loki (tests/e2e/fixtures/load.py).
    if (!isCloudRun) {
      await expect(variableEditPage).toDisplayPreviews(['200', '500']);
    }
  });
});
