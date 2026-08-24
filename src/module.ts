import { DashboardLoadedEvent, DataSourcePlugin } from '@grafana/data';
import { initPluginTranslations } from '@grafana/i18n';
import { getAppEvents } from '@grafana/runtime';

import LokiCheatSheet from './components/LokiCheatSheet';
import LokiQueryEditorByApp from './components/LokiQueryEditorByApp';
import { ConfigEditor } from './configuration/ConfigEditor';
import { LokiDatasource } from './datasource';
import pluginJson from './plugin.json';
import { onDashboardLoadedHandler } from './tracking';
import { type LokiQuery } from './types';

initPluginTranslations(pluginJson.id);

export const plugin = new DataSourcePlugin(LokiDatasource)
  .setQueryEditor(LokiQueryEditorByApp)
  .setConfigEditor(ConfigEditor)
  .setQueryEditorHelp(LokiCheatSheet);

// Subscribe to on dashboard loaded event so that we can track plugin adoption
getAppEvents().subscribe<DashboardLoadedEvent<LokiQuery>>(DashboardLoadedEvent, onDashboardLoadedHandler);
