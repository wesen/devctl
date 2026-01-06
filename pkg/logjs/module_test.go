package logjs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTempScript(t *testing.T, dir, contents string) string {
	t.Helper()
	p := filepath.Join(dir, "parser.js")
	require.NoError(t, os.WriteFile(p, []byte(contents), 0o644))
	return p
}

func TestModule_Parse_Filter_Transform(t *testing.T) {
	tmp := t.TempDir()

	scriptPath := writeTempScript(t, tmp, `
register({
  name: "t",
  parse(line, ctx) {
    const obj = log.parseJSON(line);
    if (!obj) return null;
    return { message: obj.msg, level: obj.level, trace_id: obj.trace_id, tags: ["a",""] };
  },
  filter(event, ctx) { return event.level !== "DEBUG"; },
  transform(event, ctx) { event.fields = { x: 1 }; return event; },
});
`)

	m, err := LoadFromFile(context.Background(), scriptPath, Options{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close(context.Background()) })

	ev, err := m.ProcessLine(context.Background(), `{"msg":"hi","level":"INFO","trace_id":"abc"}`+"\n", "stdin", 1)
	require.NoError(t, err)
	require.NotNil(t, ev)
	require.Equal(t, "INFO", ev.Level)
	require.Equal(t, "hi", ev.Message)
	require.Equal(t, "stdin", ev.Source)
	require.Equal(t, `{"msg":"hi","level":"INFO","trace_id":"abc"}`, ev.Raw)
	require.Equal(t, int64(1), ev.LineNumber)
	require.Equal(t, []string{"a"}, ev.Tags)
	require.Equal(t, map[string]any{"x": int64(1), "trace_id": "abc"}, ev.Fields)

	ev, err = m.ProcessLine(context.Background(), `{"msg":"no","level":"DEBUG"}`+"\n", "stdin", 2)
	require.NoError(t, err)
	require.Nil(t, ev)

	st := m.Stats()
	require.Equal(t, int64(2), st.LinesProcessed)
	require.Equal(t, int64(1), st.EventsEmitted)
	require.Equal(t, int64(1), st.LinesDropped)
}

func TestModule_Timestamp_Date_ToISOString(t *testing.T) {
	tmp := t.TempDir()

	scriptPath := writeTempScript(t, tmp, `
register({
  name: "t",
  parse(line, ctx) {
    return { timestamp: new Date("2020-01-01T00:00:00Z"), message: "x" };
  },
});
`)

	m, err := LoadFromFile(context.Background(), scriptPath, Options{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close(context.Background()) })

	ev, err := m.ProcessLine(context.Background(), "ignored\n", "stdin", 1)
	require.NoError(t, err)
	require.NotNil(t, ev)
	require.NotNil(t, ev.Timestamp)
	require.Equal(t, "2020-01-01T00:00:00.000Z", *ev.Timestamp)
}

func TestModule_Timeout(t *testing.T) {
	tmp := t.TempDir()

	scriptPath := writeTempScript(t, tmp, `
register({
  name: "t",
  parse(line, ctx) {
    while(true) {}
  },
  onError(err, payload, ctx) {
    // swallow
  }
});
`)

	m, err := LoadFromFile(context.Background(), scriptPath, Options{HookTimeout: "10ms"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Close(context.Background()) })

	ev, err := m.ProcessLine(context.Background(), "x\n", "stdin", 1)
	require.NoError(t, err)
	require.Nil(t, ev)

	st := m.Stats()
	require.Equal(t, int64(1), st.HookTimeouts)
	require.Equal(t, int64(1), st.HookErrors)
}
