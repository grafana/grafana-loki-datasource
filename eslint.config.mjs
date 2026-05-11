import { defineConfig } from 'eslint/config';
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
