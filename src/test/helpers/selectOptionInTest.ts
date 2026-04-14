// Core Grafana history https://github.com/grafana/grafana/blob/v11.0.0-preview/packages/grafana-ui/src/components/Select/SelectBase.tsx
import { waitFor } from '@testing-library/react';
import { select } from 'react-select-event';

export const selectOptionInTest = async (
  input: HTMLElement,
  optionOrOptions: string | RegExp | Array<string | RegExp>
) => await waitFor(() => select(input, optionOrOptions, { container: document.body }));
