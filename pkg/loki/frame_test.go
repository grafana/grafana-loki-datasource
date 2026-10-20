package loki

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestFormatName(t *testing.T) {
	t.Run("converting metric name", func(t *testing.T) {
		metric := map[string]string{
			"app":    "backend",
			"device": "mobile",
		}

		query := &lokiQuery{
			LegendFormat: "legend {{app}} {{ device }} {{broken}}",
		}

		require.Equal(t, "legend backend mobile ", formatName(metric, query))
	})

	t.Run("build full series name", func(t *testing.T) {
		metric := map[string]string{
			"app":    "backend",
			"device": "mobile",
		}

		query := &lokiQuery{
			LegendFormat: "",
		}

		require.Equal(t, `{app="backend", device="mobile"}`, formatName(metric, query))
	})
}

func TestAdjustFrame(t *testing.T) {
	t.Run("logs-frame metadata should be set correctly", func(t *testing.T) {
		time1 := time.Date(2022, 1, 2, 3, 4, 5, 6, time.UTC)
		time2 := time.Date(2022, 1, 2, 3, 5, 5, 6, time.UTC)
		time3 := time.Date(2022, 1, 2, 3, 5, 5, 6, time.UTC)
		time4 := time.Date(2022, 1, 2, 3, 6, 5, 6, time.UTC)

		timeNs1 := strconv.FormatInt(time1.UnixNano(), 10)
		timeNs2 := strconv.FormatInt(time2.UnixNano(), 10)
		timeNs3 := strconv.FormatInt(time3.UnixNano(), 10)
		timeNs4 := strconv.FormatInt(time4.UnixNano(), 10)

		makeFrame := func() *data.Frame {
			return data.NewFrame("",
				data.NewField("__labels", nil, []json.RawMessage{
					json.RawMessage(`{"level":"info"}`),
					json.RawMessage(`{"level":"error"}`),
					json.RawMessage(`{"level":"error"}`),
					json.RawMessage(`{"level":"info"}`),
				}),
				data.NewField("Time", nil, []time.Time{
					time1, time2, time3, time4,
				}),
				data.NewField("Line", nil, []string{"line1", "line2", "line2", "line3"}),
				data.NewField("TS", nil, []string{
					timeNs1, timeNs2, timeNs3, timeNs4,
				}),
			)
		}

		query := &lokiQuery{
			Expr:      `{type="important"}`,
			QueryType: QueryTypeRange,
			RefID:     "A",
		}

		verifyFrame := func(frame *data.Frame) {
			fields := frame.Fields

			idField := fields[len(fields)-1]
			require.Equal(t, "id", idField.Name)
			require.Equal(t, data.FieldTypeString, idField.Type())
			require.Equal(t, 4, idField.Len())
			require.Equal(t, "1641092645000000006_a36f4e1b", idField.At(0))
			require.Equal(t, "1641092705000000006_1d77c9ca", idField.At(1))
			require.Equal(t, "1641092705000000006_1d77c9ca_1", idField.At(2))
			require.Equal(t, "1641092765000000006_948c1a7d", idField.At(3))
		}

		frame := makeFrame()

		err := adjustFrame(frame, query, true, false)
		require.NoError(t, err)
		verifyFrame(frame)

		frame = makeFrame() // we need to reset the frame, because adjustFrame mutates it
		err = adjustFrame(frame, query, false, false)
		require.NoError(t, err)
		verifyFrame(frame)

		frame = makeFrame() // we need to reset the frame, because adjustFrame mutates it
		err = adjustFrame(frame, query, true, true)
		require.NoError(t, err)
		verifyFrame(frame)

		frame = makeFrame() // we need to reset the frame, because adjustFrame mutates it
		err = adjustFrame(frame, query, false, true)
		require.NoError(t, err)
		verifyFrame(frame)
	})

	t.Run("naming inside metric fields should be correct", func(t *testing.T) {
		makeFrame := func() *data.Frame {
			field1 := data.NewField("", nil, make([]time.Time, 0))
			field2 := data.NewField("", nil, make([]float64, 0))
			field2.Labels = data.Labels{"app": "Application", "tag2": "tag2"}

			frame := data.NewFrame("", field1, field2)
			frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesMulti})
			return frame
		}

		verifyFrame := func(frame *data.Frame, expectedFrameName string) {
			require.Equal(t, frame.Name, expectedFrameName)
			require.Equal(t, frame.Meta.ExecutedQueryString, "Expr: up(ALERTS)\nStep: 42s")
			require.Equal(t, frame.Fields[0].Config.Interval, float64(42000))
			require.Equal(t, frame.Fields[1].Config.DisplayNameFromDS, "legend Application")
		}

		query := &lokiQuery{
			Expr:         "up(ALERTS)",
			QueryType:    QueryTypeRange,
			LegendFormat: "legend {{app}}",
			Step:         time.Second * 42,
		}

		frame := makeFrame()
		err := adjustFrame(frame, query, true, false)
		require.NoError(t, err)

		verifyFrame(frame, "legend Application")

		frame = makeFrame()
		err = adjustFrame(frame, query, false, false)
		require.NoError(t, err)

		verifyFrame(frame, "")

		frame = makeFrame()
		err = adjustFrame(frame, query, true, true)
		require.NoError(t, err)

		verifyFrame(frame, "legend Application")

		frame = makeFrame()
		err = adjustFrame(frame, query, false, true)
		require.NoError(t, err)

		verifyFrame(frame, "")
	})

	t.Run("should set interval-attribute in response", func(t *testing.T) {
		query := &lokiQuery{
			Step:      time.Second * 42,
			QueryType: QueryTypeRange,
		}

		makeFrame := func() *data.Frame {
			field1 := data.NewField("", nil, make([]time.Time, 0))
			field2 := data.NewField("", nil, make([]float64, 0))

			frame := data.NewFrame("", field1, field2)
			frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesMulti})
			return frame
		}

		verifyFrame := func(frame *data.Frame) {
			// to keep the test simple, we assume the
			// first field is the time-field
			timeField := frame.Fields[0]
			require.NotNil(t, timeField)
			require.Equal(t, data.FieldTypeTime, timeField.Type())

			timeFieldConfig := timeField.Config
			require.NotNil(t, timeFieldConfig)
			require.Equal(t, float64(42000), timeFieldConfig.Interval)
		}

		frame := makeFrame()

		err := adjustFrame(frame, query, true, false)
		require.NoError(t, err)
		verifyFrame(frame)

		err = adjustFrame(frame, query, false, false)
		require.NoError(t, err)
		verifyFrame(frame)

		err = adjustFrame(frame, query, true, true)
		require.NoError(t, err)
		verifyFrame(frame)

		err = adjustFrame(frame, query, false, true)
		require.NoError(t, err)
		verifyFrame(frame)
	})

	t.Run("should parse response stats", func(t *testing.T) {
		stats := map[string]any{
			"summary": map[string]any{
				"bytesProcessedPerSecond": 1,
				"linesProcessedPerSecond": 2,
				"totalBytesProcessed":     3,
				"totalLinesProcessed":     4,
				"execTime":                5.5,
			},

			"store": map[string]any{
				"totalChunksRef":        6,
				"totalChunksDownloaded": 7,
				"chunksDownloadTime":    8.8,
				"headChunkBytes":        9,
				"headChunkLines":        10,
				"decompressedBytes":     11,
				"decompressedLines":     12,
				"compressedBytes":       13,
				"totalDuplicates":       14,
			},

			"ingester": map[string]any{
				"totalReached":       15,
				"totalChunksMatched": 16,
				"totalBatches":       17,
				"totalLinesSent":     18,
				"headChunkBytes":     19,
				"headChunkLines":     20,
				"decompressedBytes":  21,
				"decompressedLines":  22,
				"compressedBytes":    23,
				"totalDuplicates":    24,
			},
		}

		meta := data.FrameMeta{
			Custom: map[string]any{
				"stats": stats,
			},
		}

		expected := []data.QueryStat{
			{FieldConfig: data.FieldConfig{DisplayName: "Summary: bytes processed per second", Unit: "Bps"}, Value: 1},
			{FieldConfig: data.FieldConfig{DisplayName: "Summary: lines processed per second", Unit: ""}, Value: 2},
			{FieldConfig: data.FieldConfig{DisplayName: "Summary: total bytes processed", Unit: "decbytes"}, Value: 3},
			{FieldConfig: data.FieldConfig{DisplayName: "Summary: total lines processed", Unit: ""}, Value: 4},
			{FieldConfig: data.FieldConfig{DisplayName: "Summary: exec time", Unit: "s"}, Value: 5.5},

			{FieldConfig: data.FieldConfig{DisplayName: "Store: total chunks ref", Unit: ""}, Value: 6},
			{FieldConfig: data.FieldConfig{DisplayName: "Store: total chunks downloaded", Unit: ""}, Value: 7},
			{FieldConfig: data.FieldConfig{DisplayName: "Store: chunks download time", Unit: "s"}, Value: 8.8},
			{FieldConfig: data.FieldConfig{DisplayName: "Store: head chunk bytes", Unit: "decbytes"}, Value: 9},
			{FieldConfig: data.FieldConfig{DisplayName: "Store: head chunk lines", Unit: ""}, Value: 10},
			{FieldConfig: data.FieldConfig{DisplayName: "Store: decompressed bytes", Unit: "decbytes"}, Value: 11},
			{FieldConfig: data.FieldConfig{DisplayName: "Store: decompressed lines", Unit: ""}, Value: 12},
			{FieldConfig: data.FieldConfig{DisplayName: "Store: compressed bytes", Unit: "decbytes"}, Value: 13},
			{FieldConfig: data.FieldConfig{DisplayName: "Store: total duplicates", Unit: ""}, Value: 14},

			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: total reached", Unit: ""}, Value: 15},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: total chunks matched", Unit: ""}, Value: 16},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: total batches", Unit: ""}, Value: 17},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: total lines sent", Unit: ""}, Value: 18},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: head chunk bytes", Unit: "decbytes"}, Value: 19},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: head chunk lines", Unit: ""}, Value: 20},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: decompressed bytes", Unit: "decbytes"}, Value: 21},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: decompressed lines", Unit: ""}, Value: 22},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: compressed bytes", Unit: "decbytes"}, Value: 23},
			{FieldConfig: data.FieldConfig{DisplayName: "Ingester: total duplicates", Unit: ""}, Value: 24},
		}

		result := parseStats(meta.Custom)

		// NOTE: i compare it item-by-item otherwise the test-fail-error-message is very hard to read
		require.Len(t, result, len(expected))

		for i := 0; i < len(result); i++ {
			require.Equal(t, expected[i], result[i])
		}
	})
}

func TestParseLogline(t *testing.T) {
	custom := func(logline map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{"stats": map[string]interface{}{"logline": logline}}
	}

	t.Run("data skipped", func(t *testing.T) {
		c := custom(map[string]interface{}{
			"outcome":          "data_skipped",
			"hintStatus":       "ok",
			"hintLookupTime":   412000000.0,
			"hintRanges":       18.0,
			"skippedRequests":  20.0,
			"narrowedRequests": 4.0,
			"plannedChunks":    120000.0,
			"scannedChunks":    4800.0,
			"chunkFilterRatio": 0.96,
			"objectRequests":   37.0,
			"ioBytes":          9437184.0,
			"ioWait":           210000000.0,
			"hintWaitTime":     0.0,
		})

		byName := map[string]float64{}
		units := map[string]string{}
		for _, s := range parseStats(c) {
			byName[s.DisplayName] = s.Value
			units[s.DisplayName] = s.Unit
		}
		require.InDelta(t, 96.0, byName["Full-Text Index: chunk filter ratio"], 1e-9)
		require.Equal(t, "percent", units["Full-Text Index: chunk filter ratio"])
		require.Equal(t, 120000.0, byName["Full-Text Index: planned chunks"])
		require.Equal(t, 4800.0, byName["Full-Text Index: scanned chunks"])
		require.Equal(t, 20.0, byName["Full-Text Index: skipped requests"])
		require.Equal(t, 4.0, byName["Full-Text Index: narrowed requests"])

		// only the five stats above are exposed, the other fields in the response are ignored
		var loglineStats int
		for _, s := range parseStats(c) {
			if strings.HasPrefix(s.DisplayName, "Full-Text Index: ") {
				loglineStats++
			}
		}
		require.Equal(t, 5, loglineStats)

		notice := parseLoglineNotice(c)
		require.NotNil(t, notice)
		require.Equal(t, data.NoticeSeverityInfo, notice.Severity)
		require.Equal(t, "Full-Text Index: 96% of data skipped", notice.Text)
	})

	t.Run("not used", func(t *testing.T) {
		notice := parseLoglineNotice(custom(map[string]interface{}{"outcome": "recent_data_only"}))
		require.NotNil(t, notice)
		require.Equal(t, data.NoticeSeverityInfo, notice.Severity)
		require.Contains(t, notice.Text, "only covers recent data")
	})

	t.Run("outcomes without a notice", func(t *testing.T) {
		// these say nothing useful to the user, so no notice is created
		for _, outcome := range []string{"unsupported_query", "query_too_small"} {
			require.Nil(t, parseLoglineNotice(custom(map[string]interface{}{"outcome": outcome})), outcome)
		}
	})

	t.Run("key absent", func(t *testing.T) {
		c := map[string]interface{}{"stats": map[string]interface{}{"summary": map[string]interface{}{}}}
		require.Nil(t, parseLoglineNotice(c))
		for _, s := range parseStats(c) {
			require.NotContains(t, s.DisplayName, "Full-Text Index")
		}
	})

	t.Run("unknown outcome has no notice", func(t *testing.T) {
		require.Nil(t, parseLoglineNotice(custom(map[string]interface{}{"outcome": "something_new"})))
	})
}
