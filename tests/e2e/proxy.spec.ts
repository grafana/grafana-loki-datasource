import { expect, test } from '@grafana/plugin-e2e';

import { isCloudRun } from './env';

test.describe('Data source proxy', () => {
  test.beforeEach(() => {
    test.skip(isCloudRun, 'These tests require the locally provisioned Loki instance.');
  });

  for (const prefix of ['', '/loki']) {
    test(`supports form-encoded queries with ${prefix || '/'} in the data source URL`, async ({
      createDataSource,
      readProvisionedDataSource,
      request,
    }) => {
      const provisioned = await readProvisionedDataSource({ fileName: 'datasources.yml' });
      const dataSource = await createDataSource({
        type: provisioned.type,
        access: 'proxy',
        url: `${provisioned.url}${prefix}`,
      });
      const proxyPath = `/api/datasources/proxy/uid/${dataSource.uid}${prefix ? '' : '/loki'}/api/v1`;

      try {
        for (const [endpoint, form] of [
          ['query', { query: 'vector(1)' }],
          ['query_range', { query: 'vector(1)', step: '60' }],
          ['labels', {}],
          ['label/job/values', {}],
          ['series', { 'match[]': '{job="proxy-regression"}' }],
        ] as const) {
          await test.step(`GET and POST ${endpoint}`, async () => {
            const getResponse = await request.get(`${proxyPath}/${endpoint}`, { params: form });
            expect(getResponse.status(), await getResponse.text()).toBe(200);
            expect(await getResponse.json()).toMatchObject({ status: 'success' });

            const postResponse = await request.post(`${proxyPath}/${endpoint}`, { form });
            expect(postResponse.status(), await postResponse.text()).toBe(200);
            expect(await postResponse.json()).toMatchObject({ status: 'success' });
          });
        }

        for (const [method, endpoint] of [
          ['POST', 'push'],
          ['PUT', 'query'],
          ['DELETE', 'query'],
        ]) {
          await test.step(`rejects ${method} ${endpoint}`, async () => {
            const response = await request.fetch(`${proxyPath}/${endpoint}`, { method });
            expect(response.status(), await response.text()).toBe(403);
          });
        }
      } finally {
        const response = await request.delete(`/api/datasources/uid/${dataSource.uid}`);
        expect(response.status(), await response.text()).toBe(200);
      }
    });
  }
});
