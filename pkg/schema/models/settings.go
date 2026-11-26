package models

// Settings mirrors the plugin-specific jsonData keys for the schema conformance test.
// Test-only; the backend does not read it.
type Settings struct {
	EnableSecureSocksProxy bool           `json:"enableSecureSocksProxy"`
	ManageAlerts           bool           `json:"manageAlerts"`
	MaxLines               string         `json:"maxLines"`
	DerivedFields          []DerivedField `json:"derivedFields"`
}

// DerivedField is one derived field entry.
type DerivedField struct {
	Name            string `json:"name"`
	MatcherType     string `json:"matcherType"`
	MatcherRegex    string `json:"matcherRegex"`
	URL             string `json:"url"`
	URLDisplayLabel string `json:"urlDisplayLabel"`
	DatasourceUID   string `json:"datasourceUid"`
	TargetBlank     bool   `json:"targetBlank"`
}
