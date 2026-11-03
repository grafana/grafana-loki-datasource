import { defineConfig } from 'eslint/config';
import grafanaI18nPlugin from '@grafana/i18n/eslint-plugin';
import baseConfig from './.config/eslint.config.mjs';

export default defineConfig([
  {
    ignores: [
      '**/node_modules/',
      '**/dist/',
      '**/coverage/',
      '**/artifacts/',
      'test-results/',
      'playwright-report/',
      '**/.eslintcache',
    ],
  },
  ...baseConfig,
  {
    plugins: {
      '@grafana/i18n': grafanaI18nPlugin,
    },
    // TODO: re-enable and refactor the affected components (LokiQueryEditor,
    // VariableQueryEditor, DerivedField, MonacoQueryField). These rules ship
    // with the React Compiler hooks plugin and flag pre-existing patterns
    // inherited from grafana/grafana that work correctly today.
    rules: {
      'react-hooks/set-state-in-effect': 'off',
      'react-hooks/refs': 'off',
    },
  },
]);
