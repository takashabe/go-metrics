package forward

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/takashabe/go-metrics/collect"
	noticeio "github.com/takashabe/go-notice-io"
)

func createDummyConsoleWriterWithKeys(t *testing.T, keys ...string) *ConsoleWriter {
	sc := collect.NewSimpleCollector()
	for _, key := range keys {
		sc.Add(key, 1)
	}
	w, err := NewConsoleWriter(sc)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	cw, ok := w.(*ConsoleWriter)
	if !ok {
		t.Fatalf("want type *ConsoleWriter, got %v", reflect.TypeOf(w))
	}
	return cw
}

func TestAddMetrics(t *testing.T) {
	cw := createDummyConsoleWriterWithKeys(t, []string{"a", "b", "c"}...)

	cases := []struct {
		keys       []string
		expectKeys []string
		expectErr  error
	}{
		{
			[]string{},
			[]string{},
			nil,
		},
		{
			[]string{"b", "a"},
			[]string{"a", "b"},
			nil,
		},
		{
			[]string{"d"},
			[]string{"a", "b"},
			ErrNotExistMetrics,
		},
	}
	for i, c := range cases {
		err := cw.AddMetrics(c.keys...)
		if err != c.expectErr {
			t.Fatalf("#%d: want error %v, got %v", i, c.expectErr, err)
		}
		if !reflect.DeepEqual(cw.MetricsKeys, c.expectKeys) {
			t.Errorf("#%d: want %v, got %v", i, c.expectKeys, cw.MetricsKeys)
		}
	}
}

func TestRemoveMetrics(t *testing.T) {
	cw := createDummyConsoleWriterWithKeys(t, []string{"a", "b", "c"}...)
	cw.AddMetrics(cw.Source.GetMetricsKeys()...)

	cases := []struct {
		keys       []string
		expectKeys []string
		expectErr  error
	}{
		{
			[]string{},
			[]string{"a", "b", "c"},
			nil,
		},
		{
			[]string{"a"},
			[]string{"b", "c"},
			nil,
		},
		{
			[]string{"c", "d"},
			[]string{"b"},
			nil,
		},
	}
	for i, c := range cases {
		err := cw.RemoveMetrics(c.keys...)
		if err != c.expectErr {
			t.Fatalf("#%d: want error %v, got %v", i, c.expectErr, err)
		}
		if !reflect.DeepEqual(cw.MetricsKeys, c.expectKeys) {
			t.Errorf("#%d: want %v, got %v", i, c.expectKeys, cw.MetricsKeys)
		}
	}
}

func TestFlush(t *testing.T) {
	// setup mixed metrics collector
	sc := collect.NewSimpleCollector()
	sc.Add("a", 1)
	sc.Add("b", 1)
	sc.Histogram("h", 1)
	sc.Set("s", "A")
	sc.Set("s", "B")
	sc.Set("s2", "A'")
	w, err := NewConsoleWriter(sc)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	cw, ok := w.(*ConsoleWriter)
	if !ok {
		t.Fatalf("want type *ConsoleWriter, got %v", reflect.TypeOf(w))
	}
	cw.AddMetrics(cw.Source.GetMetricsKeys()...)

	// change writer for testing
	var buf bytes.Buffer
	if err := cw.SetDestination(&buf); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if err := cw.Flush(); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	// check valid json
	var dummy interface{}
	if err := json.Unmarshal(buf.Bytes(), &dummy); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	expect := []byte(`{"a":1.0,"b":1.0,"h.95percentile":0.0,"h.avg":1.0,"h.count":1.0,"h.max":1.0,"h.median":1.0,"s":["A","B"],"s2":["A'"]}`)
	if !reflect.DeepEqual(buf.Bytes(), expect) {
		t.Errorf("want %s, got %s", buf.Bytes(), expect)
	}
}

func TestStream(t *testing.T) {
	cw := createDummyConsoleWriterWithKeys(t, []string{"a", "b", "c"}...)
	cw.AddMetrics(cw.Source.GetMetricsKeys()...)

	nw := noticeio.NewBufferWithChannel(nil, make(chan error, 1))
	cw.SetDestination(nw)
	cw.Interval = 10 * time.Millisecond

	// testing cancel
	ctx, cancel := context.WithCancel(context.Background())
	cw.RunStream(ctx)
	before := runtime.NumGoroutine()
	for i := 0; i < 2; i++ {
		<-nw.WriteCh
	}
	cancel()
	time.Sleep(50 * time.Millisecond)
	if after := runtime.NumGoroutine(); before-1 != after {
		t.Errorf("want num %d, got %d", before-1, after)
	}

	// testing writer
	var buf bytes.Buffer
	_, err := io.Copy(&buf, nw)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if expectBuf := `{"a":1.0,"b"`; !strings.HasPrefix(buf.String(), expectBuf) {
		t.Errorf("want has prefix %s, got %s", expectBuf, buf.String()[:len(expectBuf)])
	}
}
