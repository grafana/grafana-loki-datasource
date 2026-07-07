package models

// PluginID is the datasource plugin type, matching plugin.json's id field
// (src/plugin.json:4 in the upstream repo).
const PluginID = "loki"

// DerivedFieldMatcherType is the discriminator for a derived field's
// extraction strategy — either a regex against the log line or a label name.
// Mirrors the two string values written by the editor Select at
// src/configuration/DerivedField.tsx:99-100.
type DerivedFieldMatcherType string

const (
	// DerivedFieldMatcherRegex extracts the field by applying a regular
	// expression to the log message. Default in src/configuration/DerivedField.tsx:61.
	DerivedFieldMatcherRegex DerivedFieldMatcherType = "regex"
	// DerivedFieldMatcherLabel extracts the field by reading a label of the
	// log stream (src/configuration/DerivedField.tsx:100).
	DerivedFieldMatcherLabel DerivedFieldMatcherType = "label"
)

// SecureJsonDataKey is a strictly-typed name of a secret stored in
// secureJsonData (write-only; read existing config via secureJsonFields).
type SecureJsonDataKey string

const (
	// SecureJsonDataKeyBasicAuthPassword is the Basic-auth password, set when
	// root.basicAuth is true.
	SecureJsonDataKeyBasicAuthPassword SecureJsonDataKey = "basicAuthPassword"
	// SecureJsonDataKeyTLSCACert is the custom CA PEM, set when
	// jsonData.tlsAuthWithCACert is true.
	SecureJsonDataKeyTLSCACert SecureJsonDataKey = "tlsCACert"
	// SecureJsonDataKeyTLSClientCert is the mTLS client certificate PEM, set
	// when jsonData.tlsAuth is true.
	SecureJsonDataKeyTLSClientCert SecureJsonDataKey = "tlsClientCert"
	// SecureJsonDataKeyTLSClientKey is the mTLS client key PEM, set when
	// jsonData.tlsAuth is true.
	SecureJsonDataKeyTLSClientKey SecureJsonDataKey = "tlsClientKey"
)

// SecureJsonDataConfig lists the secret key names stored in secureJsonData.
type SecureJsonDataConfig []SecureJsonDataKey

// SecureJsonDataKeys are the secret keys used by the plugin.
//
// Note: @grafana/plugin-ui's CustomHeaders component also writes indexed
// httpHeaderValue<N> secrets when the user configures custom HTTP headers.
// Those keys are not represented here because they are dynamic (see README).
var SecureJsonDataKeys = SecureJsonDataConfig{
	SecureJsonDataKeyBasicAuthPassword,
	SecureJsonDataKeyTLSCACert,
	SecureJsonDataKeyTLSClientCert,
	SecureJsonDataKeyTLSClientKey,
}

// DerivedFieldConfig mirrors the frontend DerivedFieldConfig type
// (src/types.ts:56-64) that the config editor writes into
// jsonData.derivedFields and the frontend result transformer consumes at
// src/datasource.ts:397. The Loki backend never reads this field.
type DerivedFieldConfig struct {
	Name            string                  `json:"name"`
	MatcherRegex    string                  `json:"matcherRegex"`
	MatcherType     DerivedFieldMatcherType `json:"matcherType,omitempty"`
	URL             string                  `json:"url,omitempty"`
	URLDisplayLabel string                  `json:"urlDisplayLabel,omitempty"`
	DatasourceUID   string                  `json:"datasourceUid,omitempty"`
	TargetBlank     bool                    `json:"targetBlank,omitempty"`
}

// Config is the fully loaded configuration of a Loki datasource instance.
//
// The Loki backend's only server-side consumption of settings is
// pkg/loki/loki.go:48-72, which reads settings.URL directly and hands
// backend.DataSourceInstanceSettings to the SDK's HTTPClientOptions to build
// the HTTP client (that call is what pulls in root basicAuth / TLS fields /
// custom headers / cookies). No jsonData field is unmarshaled server-side.
//
// The jsonData fields on this struct therefore represent the shape the
// frontend writes and the SDK reads, not a plugin-owned upstream settings
// model — the Loki plugin does not ship a pkg/models/settings.go. `URL`,
// `BasicAuth`, and `BasicAuthUser` are carried as root fields so callers get
// a single flat Config that mirrors what a client of the Loki datasource
// would need to authenticate.
type Config struct {
	// Root-level fields (json:"-" on the struct because they don't live in jsonData).
	// URL is read by the Loki backend directly (pkg/loki/loki.go:66). BasicAuth /
	// BasicAuthUser / WithCredentials are populated by the editor and consumed by
	// the SDK's HTTPClientOptions() call (pkg/loki/loki.go:51) — the Loki code
	// itself never touches them by name.
	URL             string `json:"-"`
	BasicAuth       bool   `json:"-"`
	BasicAuthUser   string `json:"-"`
	WithCredentials bool   `json:"-"`

	// jsonData fields — the subset the editor writes and/or the SDK reads.
	// Custom HTTP header pairs (jsonData.httpHeaderName<N> /
	// secureJsonData.httpHeaderValue<N>) are not modeled here because they are
	// dynamically indexed.
	TLSAuth           bool                 `json:"tlsAuth,omitempty"`
	TLSAuthWithCACert bool                 `json:"tlsAuthWithCACert,omitempty"`
	TLSSkipVerify     bool                 `json:"tlsSkipVerify,omitempty"`
	ServerName        string               `json:"serverName,omitempty"`
	Timeout           float64              `json:"timeout,omitempty"`
	KeepCookies       []string             `json:"keepCookies,omitempty"`
	OauthPassThru     bool                 `json:"oauthPassThru,omitempty"`
	ManageAlerts      bool                 `json:"manageAlerts,omitempty"`
	MaxLines          string               `json:"maxLines,omitempty"`
	DerivedFields     []DerivedFieldConfig `json:"derivedFields,omitempty"`

	// DecryptedSecureJSONData holds the decrypted secure values by key
	// (basicAuthPassword, tlsCACert, tlsClientCert, tlsClientKey).
	DecryptedSecureJSONData map[SecureJsonDataKey]string `json:"-"`
}
