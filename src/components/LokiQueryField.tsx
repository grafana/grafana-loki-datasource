import React, { PureComponent, type ReactNode } from 'react';

import { type QueryEditorProps } from '@grafana/data';

import { type LokiDatasource } from '../datasource';
import { shouldRefreshLabels } from '../languageUtils';
import { type LokiQuery, type LokiOptions } from '../types';

import { MonacoQueryFieldWrapper } from './monaco-query-field/MonacoQueryFieldWrapper';

export interface LokiQueryFieldProps extends QueryEditorProps<LokiDatasource, LokiQuery, LokiOptions> {
  ExtraFieldElement?: ReactNode;
  placeholder?: string;
  'data-testid'?: string;
}

export class LokiQueryField extends PureComponent<LokiQueryFieldProps> {
  componentDidMount() {
    this.props.datasource.languageProvider.start(this.props.range);
  }

  componentDidUpdate(prevProps: LokiQueryFieldProps) {
    const {
      range,
      datasource: { languageProvider },
    } = this.props;
    const refreshLabels = shouldRefreshLabels(range, prevProps.range);
    // We want to refresh labels when range changes (we round up intervals to a minute)
    if (refreshLabels) {
      languageProvider.fetchLabels({ timeRange: range });
    }
  }

  onChangeQuery = (value: string, override?: boolean) => {
    // Send text change to parent
    const { query, onChange, onRunQuery } = this.props;
    if (onChange) {
      const nextQuery = { ...query, expr: value };
      onChange(nextQuery);

      if (override && onRunQuery) {
        onRunQuery();
      }
    }
  };

  render() {
    const { ExtraFieldElement, query, datasource, history, onRunQuery, range } = this.props;
    const placeholder = this.props.placeholder ?? 'Enter a Loki query (run with Shift+Enter)';

    return (
      <>
        <div
          className="gf-form-inline gf-form-inline--xs-view-flex-column flex-grow-1"
          data-testid={this.props['data-testid']}
        >
          <div className="gf-form--grow flex-shrink-1 min-width-15">
            <MonacoQueryFieldWrapper
              datasource={datasource}
              history={history ?? []}
              onChange={this.onChangeQuery}
              onRunQuery={onRunQuery}
              initialValue={query.expr ?? ''}
              placeholder={placeholder}
              timeRange={range}
            />
          </div>
        </div>
        {ExtraFieldElement}
      </>
    );
  }
}
