import '@grafana/data';

// Augment FeatureToggles with toggles not yet in the published @grafana/data version.
declare module '@grafana/data' {
  interface FeatureToggles {
    lokiAlignedQuerySplitting?: boolean;
  }
}
