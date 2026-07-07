package schema_test

import (
	_ "embed"
	"testing"

	"github.com/grafana/dsconfig/schema"

	"github.com/grafana/grafana-loki-datasource/pkg/loki/models"
)

//go:embed dsconfig.json
var configSchemaJSON []byte

//go:generate go test -run TestPlugin -generateArtifacts
func TestPlugin(t *testing.T) {
	schema.RunPluginTests(t, schema.PluginUnderTest{
		ID:                models.PluginID,
		ConfigSchemaJSON:  configSchemaJSON,
		SettingsJSONModel: models.Config{},
		SecureKeys:        []string{"basicAuthPassword", "tlsCACert", "tlsClientCert", "tlsClientKey"},
	})
}
