package loki

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
	"github.com/grafana/loki/v3/pkg/logql/syntax"
	"github.com/stretchr/testify/require"
)

// boundaryCoverageContext enables the feature toggle that routes logs volume
// range queries through runLogsVolumeQuery.
func boundaryCoverageContext() context.Context {
	cfg := config.NewGrafanaCfg(map[string]string{featuretoggles.EnabledFeatures: flagLokiLogsVolumeBoundaryCoverage})
	return config.WithGrafanaConfig(context.Background(), cfg)
}

func TestLogsVolumeQueryPartRequests(t *testing.T) {
	query := lokiQuery{Expr: `sum by (level) (count_over_time({app="test"}[1m]))`, QueryType: QueryTypeRange, SupportingQueryType: SupportingQueryLogsVolume, Start: time.Unix(30, 0), End: time.Unix(210, 0), Step: time.Minute}
	parts, err := splitLogsVolumeQuery(query)
	require.NoError(t, err)
	require.Len(t, parts, 3)
	for _, tc := range []struct {
		name   string
		part   lokiQuery
		path   string
		params map[string]string
	}{
		{"first partial bucket", parts[0], "/loki/api/v1/query", map[string]string{"time": "60000000000", "start": "", "end": "", "step": "", "query": `sum by (level)(count_over_time({app="test"}[30s] offset 1ns))`}},
		{"middle full buckets", parts[1], "/loki/api/v1/query_range", map[string]string{"time": "", "start": "120000000000", "end": "180000000000", "step": "60000ms", "query": `sum by (level)(count_over_time({app="test"}[1m] offset 1ns))`}},
		{"last partial bucket", parts[2], "/loki/api/v1/query", map[string]string{"time": "210000000000", "start": "", "end": "", "step": "", "query": `sum by (level)(count_over_time({app="test"}[30s] offset 1ns))`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			api := makeMockedAPI(200, "application/json", []byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`), func(req *http.Request) {
				calls++
				require.Equal(t, tc.path, req.URL.Path)
				require.Equal(t, "Source=logvolhist", req.Header.Get("X-Query-Tags"))
				for name, expected := range tc.params {
					require.Equal(t, expected, req.URL.Query().Get(name), name)
				}
			})
			result, err := runSingleQuery(context.Background(), api, &tc.part, ResponseOpts{}, backend.NewLoggerWith("logger", "test"))
			require.NoError(t, err)
			require.NoError(t, result.Error)
			require.Equal(t, 1, calls)
		})
	}
}

func TestLogsVolumeMergesBucketsForEachLevel(t *testing.T) {
	frame := func(level string, times []time.Time, counts []float64) *data.Frame {
		f := data.NewFrame("", data.NewField("Time", nil, times), data.NewField("Value", data.Labels{"level": level}, counts))
		f.Meta = &data.FrameMeta{}
		return f
	}
	first := &backend.DataResponse{Frames: data.Frames{
		frame("info", []time.Time{time.Unix(60, 0)}, []float64{1}),
		frame("error", []time.Time{time.Unix(60, 0)}, []float64{5}),
	}}
	middle := &backend.DataResponse{Frames: data.Frames{
		frame("info", []time.Time{time.Unix(120, 0), time.Unix(180, 0)}, []float64{2, 3}),
	}}
	// Loki evaluates the trailing partial bucket at the real end (210s) so the
	// count matches the log lines; the merge reports it at the aligned bucket end.
	last := &backend.DataResponse{Frames: data.Frames{
		frame("info", []time.Time{time.Unix(210, 0)}, []float64{4}),
		frame("warn", []time.Time{time.Unix(210, 0)}, []float64{6}),
	}}
	query := lokiQuery{Expr: `sum by (level) (count_over_time({app="test"}[1m]))`, Start: time.Unix(30, 0), End: time.Unix(210, 0), Step: time.Minute}
	parts, err := splitLogsVolumeQuery(query)
	require.NoError(t, err)
	require.Len(t, parts, 3)
	merged := mergeLogsVolumeResults(query, parts, []*backend.DataResponse{first, middle, last})
	require.Len(t, merged.Frames, 3)
	for i, expected := range []struct {
		level  string
		times  []time.Time
		counts []float64
	}{
		{"info", []time.Time{time.Unix(60, 0), time.Unix(120, 0), time.Unix(180, 0), time.Unix(240, 0)}, []float64{1, 2, 3, 4}},
		{"error", []time.Time{time.Unix(60, 0)}, []float64{5}},
		{"warn", []time.Time{time.Unix(240, 0)}, []float64{6}},
	} {
		t.Run(expected.level, func(t *testing.T) {
			actual := merged.Frames[i]
			require.Equal(t, expected.level, actual.Fields[1].Labels["level"])
			require.Equal(t, data.FrameTypeTimeSeriesMulti, actual.Meta.Type)
			require.Equal(t, float64(60000), actual.Fields[0].Config.Interval)
			require.Equal(t, len(expected.counts), actual.Rows())
			for row, count := range expected.counts {
				require.Equal(t, expected.times[row], actual.Fields[0].At(row))
				require.Equal(t, count, actual.Fields[1].At(row))
			}
		})
	}
}

func TestLogsVolumeMergeKeepsSeriesWithCollidingLabelStrings(t *testing.T) {
	first := data.Labels{"a": "x, b=y"}
	second := data.Labels{"a": "x", "b": "y"}
	require.Equal(t, first.String(), second.String(), "test premise: the two label sets must render identically")

	frame := func(labels data.Labels, count float64) *data.Frame {
		f := data.NewFrame("", data.NewField("Time", nil, []time.Time{time.Unix(60, 0)}), data.NewField("Value", labels, []float64{count}))
		f.Meta = &data.FrameMeta{}
		return f
	}
	parts := []lokiQuery{{QueryType: QueryTypeRange, Step: time.Minute}}
	results := []*backend.DataResponse{{Frames: data.Frames{frame(first, 1), frame(second, 2)}}}

	merged := mergeLogsVolumeResults(lokiQuery{Step: time.Minute}, parts, results)

	require.Len(t, merged.Frames, 2, "distinct label sets must stay separate series")
	require.True(t, merged.Frames[0].Fields[1].Labels.Equals(first))
	require.Equal(t, 1, merged.Frames[0].Rows())
	require.Equal(t, float64(1), merged.Frames[0].Fields[1].At(0))
	require.True(t, merged.Frames[1].Fields[1].Labels.Equals(second))
	require.Equal(t, 1, merged.Frames[1].Rows())
	require.Equal(t, float64(2), merged.Frames[1].Fields[1].At(0))
}

func TestLogsVolumeBucketTime(t *testing.T) {
	for _, tc := range []struct {
		name     string
		end      time.Time
		expected time.Time
	}{
		{"aligned leading edge stays", time.Unix(60, 0), time.Unix(60, 0)},
		{"trailing edge moves to bucket end", time.Unix(210, 0), time.Unix(240, 0)},
		{"range inside one bucket", time.Unix(20, 0), time.Unix(60, 0)},
		{"one millisecond into a bucket", time.UnixMilli(60_001), time.Unix(120, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, logsVolumeBucketTime(lokiQuery{End: tc.end, Step: time.Minute}))
		})
	}
}

func TestLogsVolumeExecutedQueryString(t *testing.T) {
	query := lokiQuery{Expr: `sum by (level) (count_over_time({app="test"}[1m]))`, Start: time.Unix(30, 0).In(time.FixedZone("test", 3600)), End: time.Unix(210, 0), Step: time.Minute}
	parts, err := splitLogsVolumeQuery(query)
	require.NoError(t, err)
	frame := data.NewFrame("", data.NewField("Time", nil, []time.Time{time.Unix(120, 0)}), data.NewField("Value", nil, []float64{1}))
	frame.Meta = &data.FrameMeta{}
	merged := mergeLogsVolumeResults(query, parts, []*backend.DataResponse{{}, {Frames: data.Frames{frame}}, {}})
	require.Len(t, merged.Frames, 1)
	// The first two lines match other metric range frames so the frontend
	// recovers the user's expression; the remaining lines describe the requests.
	require.Equal(t, strings.Join([]string{
		`Expr: sum by (level) (count_over_time({app="test"}[1m]))`,
		`Step: 1m0s`,
		`Instant: sum by (level)(count_over_time({app="test"}[30s] offset 1ns)) time=1970-01-01T00:01:00Z`,
		`Range: sum by (level)(count_over_time({app="test"}[1m] offset 1ns)) start=1970-01-01T00:02:00Z end=1970-01-01T00:03:00Z`,
		`Instant: sum by (level)(count_over_time({app="test"}[30s] offset 1ns)) time=1970-01-01T00:03:30Z`,
	}, "\n"), merged.Frames[0].Meta.ExecutedQueryString)
}

func TestLogsVolumeLeavesOtherRangeQueriesUnchanged(t *testing.T) {
	for _, supporting := range []SupportingQueryType{SupportingQueryNone, SupportingQueryLogsSample, SupportingQueryDataSample, SupportingQueryInfiniteScroll, "grafana-explore"} {
		t.Run(string(supporting), func(t *testing.T) {
			calls := 0
			api := makeMockedAPI(200, "application/json", []byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`), func(req *http.Request) {
				calls++
				require.Equal(t, `sum(count_over_time({app="test"}[1m]))`, req.URL.Query().Get("query"))
				require.Equal(t, "30000000000", req.URL.Query().Get("start"))
				require.Equal(t, "210000000000", req.URL.Query().Get("end"))
			})
			query := lokiQuery{Expr: `sum(count_over_time({app="test"}[1m]))`, QueryType: QueryTypeRange, SupportingQueryType: supporting, Start: time.Unix(30, 0), End: time.Unix(210, 0), Step: time.Minute}
			_, err := runQuery(boundaryCoverageContext(), api, &query, ResponseOpts{}, backend.NewLoggerWith("logger", "test"))
			require.NoError(t, err)
			require.Equal(t, 1, calls)
		})
	}
}

func TestLogsVolumeLeavesDirectInstantQueriesUnchanged(t *testing.T) {
	calls := 0
	api := makeMockedAPI(200, "application/json", []byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`), func(req *http.Request) {
		calls++
		require.Equal(t, "/loki/api/v1/query", req.URL.Path)
		require.Equal(t, "210000000000", req.URL.Query().Get("time"))
		require.Equal(t, `sum(count_over_time({app="test"}[3m]))`, req.URL.Query().Get("query"))
	})
	query := lokiQuery{Expr: `sum(count_over_time({app="test"}[3m]))`, QueryType: QueryTypeInstant, SupportingQueryType: SupportingQueryLogsVolume, Start: time.Unix(30, 0), End: time.Unix(210, 0), Step: time.Minute}
	_, err := runQuery(boundaryCoverageContext(), api, &query, ResponseOpts{}, backend.NewLoggerWith("logger", "test"))
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

func TestLogsVolumePartFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("time") == "210000000000" {
			http.Error(w, "edge query failed", http.StatusTooManyRequests)
			return
		}
		_, err := fmt.Fprint(w, `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[60,"1"]}]}}`)
		require.NoError(t, err)
	}))
	defer server.Close()
	plog := backend.NewLoggerWith("logger", "test")
	api := newLokiAPI(server.Client(), server.URL, plog, tracing.DefaultTracer())
	query := lokiQuery{Expr: `sum(count_over_time({app="test"}[1m]))`, QueryType: QueryTypeRange, SupportingQueryType: SupportingQueryLogsVolume, Start: time.Unix(30, 0), End: time.Unix(210, 0), Step: time.Minute}
	result, err := runQuery(boundaryCoverageContext(), api, &query, ResponseOpts{}, plog)
	require.NoError(t, err)
	require.ErrorContains(t, result.Error, "edge query failed")
	require.Equal(t, backend.Status(http.StatusTooManyRequests), result.Status)
	require.Empty(t, result.Frames)
}

func TestLogsVolumeFallsBackToOriginalQuery(t *testing.T) {
	for _, tc := range []struct {
		name string
		expr string
	}{
		{"no count_over_time", `sum by (level) (bytes_over_time({app="test"}[1m]))`},
		{"unparsable", `sum by (level) (count_over_time({app="test"}[1m])`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			api := makeMockedAPI(200, "application/json", []byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`), func(req *http.Request) {
				calls++
				require.Equal(t, "/loki/api/v1/query_range", req.URL.Path)
				require.Equal(t, tc.expr, req.URL.Query().Get("query"))
				require.Equal(t, "30000000000", req.URL.Query().Get("start"))
				require.Equal(t, "210000000000", req.URL.Query().Get("end"))
			})
			query := lokiQuery{Expr: tc.expr, QueryType: QueryTypeRange, SupportingQueryType: SupportingQueryLogsVolume, Start: time.Unix(30, 0), End: time.Unix(210, 0), Step: time.Minute}
			result, err := runQuery(boundaryCoverageContext(), api, &query, ResponseOpts{}, backend.NewLoggerWith("logger", "test"))
			require.NoError(t, err)
			require.NoError(t, result.Error)
			require.Equal(t, 1, calls, "the unmodified query is sent to Loki exactly once")
		})
	}
}

func TestLogsVolumeZeroLengthRangeIsEmpty(t *testing.T) {
	calls := 0
	api := makeMockedAPI(200, "application/json", []byte(`{}`), func(*http.Request) { calls++ })
	query := lokiQuery{Expr: `sum(count_over_time({app="test"}[1m]))`, QueryType: QueryTypeRange, SupportingQueryType: SupportingQueryLogsVolume, Start: time.Unix(30, 0), End: time.Unix(30, 0), Step: time.Minute}
	result, err := runQuery(boundaryCoverageContext(), api, &query, ResponseOpts{}, backend.NewLoggerWith("logger", "test"))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, backend.StatusOK, result.Status)
	require.Empty(t, result.Frames)
	require.Zero(t, calls)
}

func TestLogsVolumeKeepsOriginalQueryWithoutFeatureToggle(t *testing.T) {
	calls := 0
	api := makeMockedAPI(200, "application/json", []byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`), func(req *http.Request) {
		calls++
		require.Equal(t, "/loki/api/v1/query_range", req.URL.Path)
		require.Equal(t, `sum by (level) (count_over_time({app="test"}[1m]))`, req.URL.Query().Get("query"))
		require.Equal(t, "30000000000", req.URL.Query().Get("start"))
		require.Equal(t, "210000000000", req.URL.Query().Get("end"))
	})
	query := lokiQuery{Expr: `sum by (level) (count_over_time({app="test"}[1m]))`, QueryType: QueryTypeRange, SupportingQueryType: SupportingQueryLogsVolume, Start: time.Unix(30, 0), End: time.Unix(210, 0), Step: time.Minute}
	result, err := runQuery(context.Background(), api, &query, ResponseOpts{}, backend.NewLoggerWith("logger", "test"))
	require.NoError(t, err)
	require.NoError(t, result.Error)
	require.Equal(t, 1, calls, "without the toggle the range query is sent to Loki unchanged, with no edge requests")
}

func TestLogsVolumeTruncatesNanosecondBounds(t *testing.T) {
	query := lokiQuery{Expr: `sum(count_over_time({app="test"}[1m]))`, Start: time.Unix(30, 500_000), End: time.Unix(210, 999_999_999), Step: time.Minute}
	parts, err := splitLogsVolumeQuery(query)
	require.NoError(t, err)
	require.Len(t, parts, 3)
	require.Equal(t, time.Unix(60, 0), parts[0].End)
	require.Contains(t, parts[0].Expr, "[30s]")
	require.Equal(t, time.UnixMilli(210_999), parts[2].End)
	require.Contains(t, parts[2].Expr, "[30s999ms]")

	t.Run("sub-millisecond range covers nothing", func(t *testing.T) {
		parts, err := splitLogsVolumeQuery(lokiQuery{Expr: `sum(count_over_time({app="test"}[1m]))`, Start: time.Unix(30, 100), End: time.Unix(30, 900), Step: time.Minute})
		require.NoError(t, err)
		require.Empty(t, parts)
	})
}

func TestLogsVolumeStatistics(t *testing.T) {
	makeResult := func(timestamp int64, count, bytes, lines, seconds float64) *backend.DataResponse {
		frame := data.NewFrame("", data.NewField("Time", nil, []time.Time{time.Unix(timestamp, 0)}), data.NewField("Value", nil, []float64{count}))
		frame.Meta = &data.FrameMeta{Stats: []data.QueryStat{
			makeStat(statSummaryTotalBytesProcessed, bytes, "decbytes"),
			makeStat(statSummaryExecTime, seconds, "s"),
			makeStat(statSummaryBytesProcessedPerSecond, bytes/seconds, "Bps"),
			makeStat(statSummaryTotalLinesProcessed, lines, ""),
			makeStat(statSummaryLinesProcessedPerSecond, lines/seconds, ""),
		}}
		return &backend.DataResponse{Frames: data.Frames{frame}}
	}
	merged := mergeLogsVolumeResults(lokiQuery{Step: time.Minute}, []lokiQuery{{QueryType: QueryTypeRange, Step: time.Minute}, {QueryType: QueryTypeRange, Step: time.Minute}}, []*backend.DataResponse{makeResult(60, 1, 100, 1000, 1), makeResult(120, 2, 500, 5000, 2)})
	require.Len(t, merged.Frames, 1)
	require.Equal(t, 2, merged.Frames[0].Rows())
	stats := merged.Frames[0].Meta.Stats
	require.Len(t, stats, 5)
	require.Equal(t, float64(600), stats[0].Value)
	require.Equal(t, float64(3), stats[1].Value)
	require.Equal(t, float64(200), stats[2].Value)
	require.Equal(t, float64(6000), stats[3].Value)
	require.Equal(t, float64(2000), stats[4].Value)
}

func FuzzLogsVolumeCoverage(f *testing.F) {
	f.Add(uint32(30_000), uint32(180_000), uint32(60_000))
	f.Add(uint32(123), uint32(866), uint32(100))
	f.Fuzz(func(t *testing.T, startMs, lengthMs, stepMs uint32) {
		step := time.Duration(stepMs%60_000+1) * time.Millisecond
		start := time.UnixMilli(int64(startMs))
		end := start.Add(time.Duration(lengthMs%600_000+1) * time.Millisecond)
		parts, err := splitLogsVolumeQuery(lokiQuery{Expr: `sum(count_over_time({app="test"}[1m]))`, Start: start, End: end, Step: step})
		require.NoError(t, err)
		require.LessOrEqual(t, len(parts), 3)
		coveredUntil := start
		for _, part := range parts {
			expr, err := syntax.ParseExpr(part.Expr)
			require.NoError(t, err)
			var window time.Duration
			expr.Walk(func(node syntax.Expr) bool {
				if e, ok := node.(*syntax.LogRangeExpr); ok {
					window = e.Interval
					require.Equal(t, time.Nanosecond, e.Offset)
				}
				return true
			})
			firstEvaluation := part.End
			if part.QueryType == QueryTypeRange {
				firstEvaluation = part.Start
				require.Zero(t, part.Start.UnixMilli()%step.Milliseconds())
				require.Zero(t, part.End.UnixMilli()%step.Milliseconds())
				require.Equal(t, step, window)
			}
			require.Equal(t, coveredUntil.UnixNano(), firstEvaluation.Add(-window).UnixNano(), strconv.Itoa(len(parts)))
			require.True(t, part.End.After(coveredUntil))
			coveredUntil = part.End
		}
		require.Equal(t, end.UnixNano(), coveredUntil.UnixNano())
	})
}

func TestSplitLogsVolumeQuery(t *testing.T) {
	for _, tc := range []struct {
		name          string
		start, end    int64
		types         []QueryType
		ends, windows []int64
	}{
		{"both edges", 30, 1830, []QueryType{QueryTypeInstant, QueryTypeRange, QueryTypeInstant}, []int64{60, 1800, 1830}, []int64{30, 60, 30}},
		{"aligned", 0, 1800, []QueryType{QueryTypeRange}, []int64{1800}, []int64{60}},
		{"left edge", 30, 1800, []QueryType{QueryTypeInstant, QueryTypeRange}, []int64{60, 1800}, []int64{30, 60}},
		{"right edge", 0, 1830, []QueryType{QueryTypeRange, QueryTypeInstant}, []int64{1800, 1830}, []int64{60, 30}},
		{"within one bucket", 10, 20, []QueryType{QueryTypeInstant}, []int64{20}, []int64{10}},
		{"crosses one boundary", 50, 70, []QueryType{QueryTypeInstant, QueryTypeInstant}, []int64{60, 70}, []int64{10, 10}},
		{"partial then one full bucket", 30, 120, []QueryType{QueryTypeInstant, QueryTypeRange}, []int64{60, 120}, []int64{30, 60}},
		{"partial ending on boundary", 30, 60, []QueryType{QueryTypeInstant}, []int64{60}, []int64{30}},
		{"one full bucket", 60, 120, []QueryType{QueryTypeRange}, []int64{120}, []int64{60}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query := lokiQuery{Expr: `sum by (level) (count_over_time({app="test"} |= "[1m]" [1m]))`, Start: time.Unix(tc.start, 0), End: time.Unix(tc.end, 0), Step: time.Minute, RefID: "A", SupportingQueryType: SupportingQueryLogsVolume}
			parts, err := splitLogsVolumeQuery(query)
			require.NoError(t, err)
			require.Len(t, parts, len(tc.types))
			coveredUntil := query.Start
			for i, part := range parts {
				require.Equal(t, tc.types[i], part.QueryType)
				require.Equal(t, time.Unix(tc.ends[i], 0), part.End)
				require.Equal(t, query.RefID, part.RefID)
				require.Equal(t, query.SupportingQueryType, part.SupportingQueryType)
				require.Contains(t, part.Expr, `|= "[1m]"`)
				expr, err := syntax.ParseExpr(part.Expr)
				require.NoError(t, err)
				expr.Walk(func(node syntax.Expr) bool {
					if window, ok := node.(*syntax.LogRangeExpr); ok {
						require.Equal(t, time.Duration(tc.windows[i])*time.Second, window.Interval)
						require.Equal(t, time.Nanosecond, window.Offset)
						if part.QueryType == QueryTypeRange {
							require.Equal(t, coveredUntil, part.Start.Add(-window.Interval))
						} else {
							require.Equal(t, coveredUntil, part.End.Add(-window.Interval))
						}
					}
					return true
				})
				coveredUntil = part.End
			}
			require.Equal(t, query.End, coveredUntil)
		})
	}
}

func TestLogsVolumePreservesExistingOffset(t *testing.T) {
	query := lokiQuery{Expr: `sum(count_over_time({app="test"}[1m] offset 1s))`, Start: time.Unix(30, 0), End: time.Unix(210, 0), Step: time.Minute}
	parts, err := splitLogsVolumeQuery(query)
	require.NoError(t, err)
	require.Len(t, parts, 3)
	for _, part := range parts {
		expr, err := syntax.ParseExpr(part.Expr)
		require.NoError(t, err)
		expr.Walk(func(node syntax.Expr) bool {
			if window, ok := node.(*syntax.LogRangeExpr); ok {
				require.Equal(t, time.Second+time.Nanosecond, window.Offset)
			}
			return true
		})
	}
}

func TestLogsVolumeDailyChunksCoverEveryBucket(t *testing.T) {
	for _, step := range []time.Duration{time.Minute, 7 * time.Minute} {
		t.Run(step.String(), func(t *testing.T) {
			start := time.Unix(30, 0)
			end := start.Add(73 * time.Hour)
			boundaries := []time.Time{start}
			for day := 1; day <= 3; day++ {
				boundaries = append(boundaries, alignLogsVolumeTime(time.Unix(0, 0).Add(time.Duration(day)*24*time.Hour), step))
			}
			boundaries = append(boundaries, end)
			coveredUntil := start
			instantCount := 0
			for i := 1; i < len(boundaries); i++ {
				parts, err := splitLogsVolumeQuery(lokiQuery{Expr: `sum(count_over_time({app="test"}[1m]))`, Start: boundaries[i-1], End: boundaries[i], Step: step})
				require.NoError(t, err)
				for _, part := range parts {
					if part.QueryType == QueryTypeInstant {
						instantCount++
						require.True(t, i == 1 || i == len(boundaries)-1)
					} else {
						require.Equal(t, coveredUntil, part.Start.Add(-step), "no bucket skipped at daily boundary")
					}
					coveredUntil = part.End
				}
			}
			require.Equal(t, 2, instantCount)
			require.Equal(t, end, coveredUntil)
		})
	}
}
