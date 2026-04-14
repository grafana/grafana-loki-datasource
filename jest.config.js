// force timezone to UTC to allow tests to work regardless of local timezone
process.env.TZ = 'UTC';

module.exports = {
  ...require('./.config/jest.config'),
  transformIgnorePatterns: [
    require('./.config/jest/utils').nodeModulesToTransform([
      ...require('./.config/jest/utils').grafanaESModules,
      'monaco-editor',
    ]),
  ],
};
