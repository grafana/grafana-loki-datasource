// force timezone to UTC to allow tests to work regardless of local timezone
process.env.TZ = 'UTC';

module.exports = {
  ...require('./.config/jest.config'),
  moduleNameMapper: {
    ...require('./.config/jest.config').moduleNameMapper,
    '^monaco-editor$': '<rootDir>/src/__mocks__/monaco-editor.ts',
  },
  transformIgnorePatterns: [
    require('./.config/jest/utils').nodeModulesToTransform([
      ...require('./.config/jest/utils').grafanaESModules,
      'monaco-editor',
      '@openfeature/ofrep-web-provider',
      '@openfeature/web-sdk',
      '@lezer/lr',
      '@lezer/common',
      '@lezer/highlight',
      '@grafana/lezer-logql',
      '@marcbachmann/cel-js',
    ]),
  ],
};
