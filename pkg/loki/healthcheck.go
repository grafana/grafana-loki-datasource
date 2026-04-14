package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/grafana/grafana-loki-datasource/pkg/loki/kinds/dataquery"
)

const (
	refID = "__healthcheck__"
)

func (ds *DataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult,
	error) {
	logger := ds.logger.With("endpoint", "checkHealth")
	hc := healthcheck(ctx, req, ds, logger)
	return hc, nil
}

func healthcheck(ctx context.Context, req *backend.CheckHealthRequest, ds *DataSource, logger log.Logger) *backend.CheckHealthResult {
	step := "1s"
	qt := "instant"
	qm := dataquery.LokiDataQuery{
		Expr:      "vector(1)+vector(1)",
		Step:      &step,
		QueryType: &qt,
	}
	b, _ := json.Marshal(&qm)

	query := backend.DataQuery{
		RefID: refID,
		TimeRange: backend.TimeRange{
			From: time.Unix(1, 0).UTC(),
			To:   time.Unix(4, 0).UTC(),
		},
		JSON: b,
	}
	resp, err := ds.QueryData(ctx, &backend.QueryDataRequest{
		PluginContext: req.PluginContext,
		Queries:       []backend.DataQuery{query},
	})

	if err != nil {
		return getHealthCheckMessage(fmt.Errorf("error received while querying loki: %w", err), logger)
	}

	if resp.Responses[refID].Error != nil {
		return getHealthCheckMessage(fmt.Errorf("error from loki: %w", resp.Responses[refID].Error), logger)
	}

	frameLen := len(resp.Responses[refID].Frames)
	if frameLen != 1 {
		return getHealthCheckMessage(fmt.Errorf("invalid dataframe length, expected %d got %d", 1, frameLen), logger)
	}

	fieldLen := len(resp.Responses[refID].Frames[0].Fields)
	if fieldLen != 2 {
		return getHealthCheckMessage(fmt.Errorf("invalid dataframe field length, expected %d got %d", 2, fieldLen), logger)
	}

	fieldValueLen := resp.Responses[refID].Frames[0].Fields[0].Len()
	if fieldValueLen != 1 {
		return getHealthCheckMessage(fmt.Errorf("invalid dataframe field value length, expected %d got %d", 1, fieldValueLen), logger)
	}

	rspValue := resp.Responses[refID].Frames[0].Fields[1].At(0).(float64)
	if rspValue != 2 {
		return getHealthCheckMessage(fmt.Errorf("invalid response value, expected %d got %f", 2, rspValue), logger)
	}

	return getHealthCheckMessage(nil, logger)
}

func getHealthCheckMessage(err error, logger log.Logger) *backend.CheckHealthResult {
	if err == nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "Data source successfully connected.",
		}
	}

	logger.Error("Loki health check failed", "error", err)
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: "Unable to connect with Loki. Please check the server logs for more details.",
	}
}
