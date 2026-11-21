package loki

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/grafana/dskit/concurrency"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/loki/v3/pkg/logql/syntax"
)

func runLogsVolumeQuery(ctx context.Context, api *LokiAPI, query *lokiQuery, opts ResponseOpts, plog log.Logger) (*backend.DataResponse, error) {
	if query.End.Equal(query.Start) {
		// A zero-length range covers no log lines, so the histogram is empty.
		return &backend.DataResponse{Status: backend.StatusOK}, nil
	}
	parts, err := splitLogsVolumeQuery(*query)
	if err != nil {
		// Non-standard volume queries that cannot be rewritten are sent to Loki unchanged.
		plog.Debug("Running logs volume query without edge adjustment", "error", err, "query", query.Expr)
		return runSingleQuery(ctx, api, query, opts, plog)
	}
	// There are at most three requests. Preserve chronological order when
	// merging, regardless of the order in which the requests finish.
	results := make([]*backend.DataResponse, len(parts))
	var failedMu sync.Mutex
	var failed *backend.DataResponse
	err = concurrency.ForEachJob(ctx, len(parts), len(parts), func(ctx context.Context, i int) error {
		result, err := runSingleQuery(ctx, api, &parts[i], opts, plog)
		if err != nil {
			return err
		}
		if result.Error != nil {
			// A partial histogram would present an incorrect total as success,
			// so returning the error cancels the remaining parts. Cancelled
			// parts report errors of their own; keep the one that caused it.
			failedMu.Lock()
			if failed == nil {
				failed = result
			}
			failedMu.Unlock()
			return result.Error
		}
		results[i] = result
		return nil
	})
	if err != nil {
		if failed != nil && errors.Is(err, failed.Error) {
			failed.Frames = nil
			return failed, nil
		}
		return nil, err
	}
	return mergeLogsVolumeResults(*query, parts, results), nil
}

func mergeLogsVolumeResults(query lokiQuery, parts []lokiQuery, results []*backend.DataResponse) *backend.DataResponse {
	merged := &backend.DataResponse{Status: backend.StatusOK}
	series := make(map[data.Fingerprint]*data.Frame)
	var stats []data.QueryStat
	statIndexes := make(map[string]int)
	for i, result := range results {
		for _, frame := range result.Frames {
			for _, stat := range frame.Meta.Stats {
				if index, ok := statIndexes[stat.DisplayName]; ok {
					stats[index].Value += stat.Value
				} else {
					statIndexes[stat.DisplayName] = len(stats)
					stats = append(stats, stat)
				}
			}
			frame.Meta.Stats = nil
			if parts[i].QueryType == QueryTypeInstant {
				setFieldTimes(frame.Fields[0], logsVolumeBucketTime(parts[i]))
			}
			// Adjusted metric frames keep their labels on the value field.
			key := frame.Fields[1].Labels.Fingerprint()
			if existing, ok := series[key]; ok {
				for row := 0; row < frame.Rows(); row++ {
					existing.AppendRow(frame.Fields[0].At(row), frame.Fields[1].At(row))
				}
				existing.Meta.Notices = append(existing.Meta.Notices, frame.Meta.Notices...)
			} else {
				frame.Meta.Type = data.FrameTypeTimeSeriesMulti
				if frame.Fields[0].Config == nil {
					frame.Fields[0].Config = &data.FieldConfig{}
				}
				frame.Fields[0].Config.Interval = float64(parts[i].Step.Milliseconds())
				series[key] = frame
				merged.Frames = append(merged.Frames, frame)
			}
		}
	}
	executed := logsVolumeExecutedQueryString(query, parts)
	for _, frame := range merged.Frames {
		frame.Meta.ExecutedQueryString = executed
	}
	// Rates must be recomputed from total work and execution time, not summed.
	if index, ok := statIndexes[statSummaryExecTime]; ok && stats[index].Value > 0 {
		bytesRate, hasBytesRate := statIndexes[statSummaryBytesProcessedPerSecond]
		bytesTotal, hasBytesTotal := statIndexes[statSummaryTotalBytesProcessed]
		if hasBytesRate && hasBytesTotal {
			stats[bytesRate].Value = stats[bytesTotal].Value / stats[index].Value
		}
		linesRate, hasLinesRate := statIndexes[statSummaryLinesProcessedPerSecond]
		linesTotal, hasLinesTotal := statIndexes[statSummaryTotalLinesProcessed]
		if hasLinesRate && hasLinesTotal {
			stats[linesRate].Value = stats[linesTotal].Value / stats[index].Value
		}
	}
	if len(merged.Frames) > 0 {
		merged.Frames[0].Meta.Stats = stats
	}
	return merged
}

// logsVolumeExecutedQueryString keeps the "Expr: <query>\nStep: <step>" shape
// of other metric range frames, which the frontend reads, and lists the
// requests actually sent to Loki after it.
func logsVolumeExecutedQueryString(query lokiQuery, parts []lokiQuery) string {
	lines := []string{"Expr: " + query.Expr, "Step: " + query.Step.String()}
	for _, part := range parts {
		if part.QueryType == QueryTypeInstant {
			lines = append(lines, fmt.Sprintf("Instant: %s time=%s", part.Expr, part.End.UTC().Format(time.RFC3339Nano)))
		} else {
			lines = append(lines, fmt.Sprintf("Range: %s start=%s end=%s", part.Expr, part.Start.UTC().Format(time.RFC3339Nano), part.End.UTC().Format(time.RFC3339Nano)))
		}
	}
	return strings.Join(lines, "\n")
}

// logsVolumeBucketTime returns the aligned end of the bucket a partial edge
// evaluation belongs to. The evaluation itself happens at the real range edge
// so the count matches the log lines, but every point is reported on the
// step grid so the histogram bars keep uniform spacing.
func logsVolumeBucketTime(part lokiQuery) time.Time {
	return alignLogsVolumeTime(part.End.Add(-time.Nanosecond), part.Step).Add(part.Step)
}

func setFieldTimes(field *data.Field, timestamp time.Time) {
	for row := 0; row < field.Len(); row++ {
		switch field.Type() {
		case data.FieldTypeTime:
			field.Set(row, timestamp)
		case data.FieldTypeNullableTime:
			field.Set(row, &timestamp)
		}
	}
}

// splitLogsVolumeQuery partitions [Start, End) into disjoint count windows.
// Only the interior uses query_range so Loki's step alignment cannot expand
// either partial edge. The interior remains aligned for result caching.
func splitLogsVolumeQuery(query lokiQuery) ([]lokiQuery, error) {
	if query.Step < time.Millisecond {
		return nil, fmt.Errorf("logs volume step must be at least one millisecond")
	}
	if !query.End.After(query.Start) {
		return nil, fmt.Errorf("logs volume end must be after start")
	}
	// api.go serializes the HTTP step parameter in whole milliseconds. Bucket
	// boundaries and lookbacks use the same precision, so nanosecond bounds
	// (startNs/endNs) are truncated and the histogram may differ from the log
	// lines by less than a millisecond at either edge.
	query.Step = query.Step.Truncate(time.Millisecond)
	query.Start = query.Start.Truncate(time.Millisecond)
	query.End = query.End.Truncate(time.Millisecond)
	parts := make([]lokiQuery, 0, 3)
	if !query.End.After(query.Start) {
		return parts, nil
	}
	start := query.Start
	alignedStart := alignLogsVolumeTime(start, query.Step)
	if !start.Equal(alignedStart) {
		end := alignedStart.Add(query.Step)
		if end.After(query.End) {
			end = query.End
		}
		part, err := logsVolumeInstantPart(query, end, end.Sub(start))
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
		start = end
	}
	alignedEnd := alignLogsVolumeTime(query.End, query.Step)
	if start.Before(alignedEnd) {
		// Evaluations mark bucket ends: the first full lookback starts at
		// start, so its evaluation must be one step later.
		part, err := logsVolumeRangePart(query, start.Add(query.Step), alignedEnd)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
		start = alignedEnd
	}
	if start.Before(query.End) {
		part, err := logsVolumeInstantPart(query, query.End, query.End.Sub(start))
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func logsVolumeInstantPart(query lokiQuery, evaluationTime time.Time, lookback time.Duration) (lokiQuery, error) {
	expression, err := logsVolumeExpression(query.Expr, lookback)
	if err != nil {
		return lokiQuery{}, err
	}
	query.Expr = expression
	// The instant endpoint uses End as its time parameter and has no start.
	query.Start = time.Time{}
	query.End = evaluationTime
	query.QueryType = QueryTypeInstant
	return query, nil
}

func logsVolumeRangePart(query lokiQuery, start, end time.Time) (lokiQuery, error) {
	expression, err := logsVolumeExpression(query.Expr, query.Step)
	if err != nil {
		return lokiQuery{}, err
	}
	query.Expr = expression
	query.Start, query.End = start, end
	query.QueryType = QueryTypeRange
	return query, nil
}

func logsVolumeExpression(expression string, window time.Duration) (string, error) {
	expr, err := syntax.ParseExpr(expression)
	if err != nil {
		return "", fmt.Errorf("parse logs volume query: %w", err)
	}
	found := false
	expr.Walk(func(node syntax.Expr) bool {
		if aggregation, ok := node.(*syntax.RangeAggregationExpr); ok && aggregation.Operation == syntax.OpRangeTypeCount {
			aggregation.Left.Interval = window
			// Shift coverage from (start, end] to [start, end) without changing
			// the lookback duration or the aligned evaluation timestamps.
			aggregation.Left.Offset += time.Nanosecond
			found = true
		}
		return true
	})
	if !found {
		return "", fmt.Errorf("logs volume query must contain count_over_time")
	}
	return expr.String(), nil
}

// alignLogsVolumeTime matches Loki's queryrangebase.StepAlignMiddleware:
// (timestamp.UnixMilli() / stepMilliseconds) * stepMilliseconds.
// Callers validate and truncate step to at least one whole millisecond.
func alignLogsVolumeTime(timestamp time.Time, step time.Duration) time.Time {
	stepMilliseconds := step.Milliseconds()
	return time.UnixMilli(timestamp.UnixMilli() / stepMilliseconds * stepMilliseconds)
}
