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
      // React 17+ new JSX transform doesn't require React in scope
      'react/react-in-jsx-scope': 'off',
    },
  },
]);
