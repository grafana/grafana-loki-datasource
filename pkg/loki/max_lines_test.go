package loki

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

// Queries that reach the backend without the panel query path, such as SQL
// expressions, alerting, and API calls, often have no maxLines. Without a
// limit parameter Loki returns only 100 lines, so the backend must send the
// data source's line limit, as the frontend does.
func TestQueryDataLineLimit(t *testing.T) {
	for _, tc := range []struct {
		name      string
		jsonData  string
		queryJSON string
		want      string
	}{
		{"no saved limit uses default", `{}`, `{"expr":"{job=\"a\"}","queryType":"range"}`, "1000"},
		{"no saved limit uses data source setting", `{"maxLines":"500"}`, `{"expr":"{job=\"a\"}","queryType":"range"}`, "500"},
		{"numeric data source setting", `{"maxLines":250}`, `{"expr":"{job=\"a\"}","queryType":"range"}`, "250"},
		{"invalid data source setting uses default", `{"maxLines":"abc"}`, `{"expr":"{job=\"a\"}","queryType":"range"}`, "1000"},
		{"zero saved limit uses data source setting", `{"maxLines":"500"}`, `{"expr":"{job=\"a\"}","queryType":"range","maxLines":0}`, "500"},
		{"saved limit wins", `{"maxLines":"500"}`, `{"expr":"{job=\"a\"}","queryType":"range","maxLines":20}`, "20"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotLimit := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				gotLimit <- req.URL.Query().Get("limit")
				_, _ = fmt.Fprint(w, `{"status":"success","data":{"resultType":"streams","result":[]}}`)
			}))
			defer server.Close()

			inst, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{
				URL:      server.URL,
				JSONData: []byte(tc.jsonData),
			})
			require.NoError(t, err)

			_, err = inst.(*DataSource).QueryData(context.Background(), &backend.QueryDataRequest{
				Queries: []backend.DataQuery{{
					RefID:     "A",
					JSON:      []byte(tc.queryJSON),
					TimeRange: backend.TimeRange{From: time.Unix(0, 0), To: time.Unix(600, 0)},
					Interval:  time.Second,
				}},
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, <-gotLimit)
		})
	}
}
