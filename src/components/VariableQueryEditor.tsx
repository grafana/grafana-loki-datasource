import React, { type FormEvent, useState, useEffect } from 'react';
import { usePrevious } from 'react-use';

import { type QueryEditorProps, type SelectableValue } from '@grafana/data';
import { InlineField, InlineFieldRow, Input, Select } from '@grafana/ui';

import { type LokiDatasource } from '../datasource';
import { migrateVariableQuery } from '../migrations/variableQueryMigrations';
import { type LokiOptions, type LokiQuery, type LokiVariableQuery, LokiVariableQueryType as QueryType } from '../types';

const variableOptions = [
  { label: 'Label names', value: QueryType.LabelNames },
  { label: 'Label values', value: QueryType.LabelValues },
  { label: 'Detected field values', value: QueryType.DetectedFieldValues },
];

export type Props = QueryEditorProps<LokiDatasource, LokiQuery, LokiOptions, LokiVariableQuery>;

const refId = 'LokiVariableQueryEditor-VariableQuery';

export const LokiVariableQueryEditor = ({ onChange, query, datasource, range }: Props) => {
  const [type, setType] = useState<number | undefined>(undefined);
  const [label, setLabel] = useState('');
  const [labelOptions, setLabelOptions] = useState<Array<SelectableValue<string>>>([]);
  const [stream, setStream] = useState('');
  const previousType = usePrevious(type);

  useEffect(() => {
    if (!query) {
      return;
    }

    const variableQuery = typeof query === 'string' ? migrateVariableQuery(query) : query;
    setType(variableQuery.type);
    setLabel(variableQuery.label || '');
    setStream(variableQuery.stream || '');
  }, [query]);

  useEffect(() => {
    // Fetch label names when the query type is LabelValues, and the previous type was not the same
    if (type !== QueryType.LabelValues || previousType === type) {
      return;
    }

    datasource.languageProvider.fetchLabels({ timeRange: range }).then((labelNames) => {
      setLabelOptions(labelNames.map((labelName) => ({ label: labelName, value: labelName })));
    });
  }, [datasource, type, range, previousType]);

  const onQueryTypeChange = (newType: SelectableValue<QueryType>) => {
    // Label and stream have different semantics for Detected field values, so reset them when entering or leaving it
    const resetFields = (type === QueryType.DetectedFieldValues) !== (newType.value === QueryType.DetectedFieldValues);
    setType(newType.value);
    if (resetFields) {
      setLabel('');
      setStream('');
    }
    if (newType.value !== undefined) {
      onChange({
        type: newType.value,
        label: resetFields ? '' : label,
        stream: resetFields ? '' : stream,
        refId,
      });
    }
  };

  const onLabelChange = (newLabel: SelectableValue<string>) => {
    setLabel(newLabel.value || '');
  };

  const onStreamChange = (e: FormEvent<HTMLInputElement>) => {
    setStream(e.currentTarget.value);
  };

  const handleBlur = () => {
    if (type !== undefined) {
      onChange({ type, label, stream, refId: 'LokiVariableQueryEditor-VariableQuery' });
    }
  };

  return (
    <>
      <InlineFieldRow>
        <InlineField label="Query type" labelWidth={20}>
          <Select
            aria-label="Query type"
            onChange={onQueryTypeChange}
            onBlur={handleBlur}
            value={type}
            options={variableOptions}
            width={16}
          />
        </InlineField>
        {(type === QueryType.LabelValues || type === QueryType.DetectedFieldValues) && (
          <>
            <InlineField label="Label" labelWidth={20}>
              <Select
                aria-label="Label"
                onChange={onLabelChange}
                onBlur={handleBlur}
                value={{ label: label, value: label }}
                options={type === QueryType.LabelValues ? labelOptions : []}
                width={16}
                allowCustomValue
              />
            </InlineField>
          </>
        )}
      </InlineFieldRow>
      {(type === QueryType.LabelValues || type === QueryType.DetectedFieldValues) && (
        <InlineFieldRow>
          <InlineField
            label={type === QueryType.DetectedFieldValues ? 'LogQL query' : 'Stream selector'}
            labelWidth={20}
            grow={true}
            tooltip={
              <div>
                {type === QueryType.DetectedFieldValues
                  ? 'Required. The LogQL query to detect field values from. Fields extracted by a parser need the parser pipeline in the query, for example: {app="x"} | pattern "<method> <path>". Values are observed from a sample of recent matching log lines, so rare values may be missing.'
                  : 'Optional. If defined, a list of values for the specified log stream selector is returned. For example: {label="value"} or {label="$variable"}'}
              </div>
            }
          >
            <Input
              type="text"
              aria-label={type === QueryType.DetectedFieldValues ? 'LogQL query' : 'Stream selector'}
              placeholder={
                type === QueryType.DetectedFieldValues ? 'LogQL query (required)' : 'Optional stream selector'
              }
              value={stream}
              onChange={onStreamChange}
              onBlur={handleBlur}
            />
          </InlineField>
        </InlineFieldRow>
      )}
    </>
  );
};
