import React, { type ReactNode, useState } from 'react';

import { type Field, FieldType, mapInternalLinkToExplore } from '@grafana/data';
import { getDataSourceSrv, getTemplateSrv } from '@grafana/runtime';
import { InlineField, TextArea } from '@grafana/ui';

import { type DerivedFieldConfig } from '../types';

type Props = {
  derivedFields?: DerivedFieldConfig[];
  className?: string;
};
export const DebugSection = (props: Props) => {
  const { derivedFields, className } = props;
  const [debugText, setDebugText] = useState('');

  let debugFields: DebugField[] = [];
  if (debugText && derivedFields) {
    debugFields = makeDebugFields(derivedFields, debugText);
  }

  return (
    <div className={className}>
      <InlineField label="Debug log message" labelWidth={24} grow>
        <TextArea
          type="text"
          aria-label="Loki query"
          placeholder="Paste an example log line here to test the regular expressions of your derived fields"
          value={debugText}
          onChange={(event) => setDebugText(event.currentTarget.value)}
        />
      </InlineField>
      {!!debugFields.length && <DebugFields fields={debugFields} />}
    </div>
  );
};

type DebugFieldItemProps = {
  fields: DebugField[];
};
const DebugFields = ({ fields }: DebugFieldItemProps) => {
  return (
    <table className={'filter-table'}>
      <thead>
        <tr>
          <th>Name</th>
          <th>Value</th>
          <th>Url</th>
        </tr>
      </thead>
      <tbody>
        {fields.map((field) => {
          let value: ReactNode = field.value;
          if (field.error && field.error instanceof Error) {
            value = field.error.message;
          } else if (field.href) {
            value = <a href={field.href}>{value}</a>;
          }
          return (
            <tr key={`${field.name}=${field.value}`}>
              <td>{field.name}</td>
              <td>{value}</td>
              <td>{field.href ? <a href={field.href}>{field.href}</a> : ''}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
};

type DebugField = {
  name: string;
  error?: unknown;
  value?: string;
  href?: string;
};

function makeDebugFields(derivedFields: DerivedFieldConfig[], debugText: string): DebugField[] {
  return derivedFields
    .filter((field) => field.name && field.matcherRegex)
    .map((field) => {
      try {
        const testMatch = debugText.match(field.matcherRegex);
        let href;
        const value = testMatch && testMatch[1];

        if (value) {
          const scopedVars = {
            __value: {
              value: {
                raw: value,
              },
              text: 'Raw value',
            },
          };
          const replaceVariables = (text: string) => getTemplateSrv().replace(text, scopedVars);

          if (field.datasourceUid) {
            const datasource = getDataSourceSrv().getInstanceSettings(field.datasourceUid);
            const queryType =
              datasource?.type === 'tempo'
                ? 'traceql'
                : datasource?.type === 'grafana-x-ray-datasource'
                  ? 'getTrace'
                  : undefined;
            const internalLink = {
              datasourceUid: field.datasourceUid,
              datasourceName: datasource?.name ?? 'Data source not found',
              query: {
                query: field.url,
                queryType,
              },
            };
            href = mapInternalLinkToExplore({
              link: { title: '', url: '', internal: internalLink },
              internalLink,
              scopedVars,
              field: { name: field.name, type: FieldType.string, config: {}, values: [value] } as Field,
              replaceVariables,
            }).href;
          } else {
            href = replaceVariables(field.url ?? '');
          }
        }
        const debugFiled: DebugField = {
          name: field.name,
          value: value || '<no match>',
          href,
        };
        return debugFiled;
      } catch (error) {
        return {
          name: field.name,
          error,
        };
      }
    });
}
