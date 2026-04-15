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
    rules: {
      'react/react-in-jsx-scope': 'off',
      // Rules added in eslint-plugin-react-hooks v7 that did not exist in the grafana
      // monorepo's v5. Disable to match the effective behaviour of the source repo.
      'react-hooks/set-state-in-effect': 'off',
      'react-hooks/refs': 'off',
    },
  },
]);
