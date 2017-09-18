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

func createDummySimpleWriterWithKeys(t *testing.T, w io.Writer, keys ...string) *SimpleWriter {
	sc := collect.NewSimpleCollector()
	for _, key := range keys {
		sc.Add(key, 1)
	}
	mw, err := NewSimpleWriter(sc, w)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	sw, ok := mw.(*SimpleWriter)
	if !ok {
		t.Fatalf("want type *SimpleWriter, got %v", reflect.TypeOf(mw))
	}
	return sw
}

func TestAddMetrics(t *testing.T) {
	cw := createDummySimpleWriterWithKeys(t, nil, []string{"a", "b", "c"}...)

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
	cw := createDummySimpleWriterWithKeys(t, nil, []string{"a", "b", "c"}...)
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

func registerDummyMetrics(t *testing.T, c collect.Collector) {
	c.Add("a", 1)
	c.Add("b", 1)
	c.Histogram("h", 1)
	c.Set("s", "A")
	c.Set("s", "B")
	c.Set("s2", "A'")
}

func TestFlush(t *testing.T) {
	// setup mixed metrics collector
	sc := collect.NewSimpleCollector()
	registerDummyMetrics(t, sc)
	w, err := NewSimpleWriter(sc, nil)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	cw, ok := w.(*SimpleWriter)
	if !ok {
		t.Fatalf("want type *SimpleWriter, got %v", reflect.TypeOf(w))
	}
	cw.AddMetrics(cw.Source.GetMetricsKeys()...)

	// change writer for testing
	var buf bytes.Buffer
	cw.SetDestination(&buf)
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

func TestFlushWithKeys(t *testing.T) {
	// setup mixed metrics collector
	sc := collect.NewSimpleCollector()
	registerDummyMetrics(t, sc)
	w, err := NewSimpleWriter(sc, nil)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	cw, ok := w.(*SimpleWriter)
	if !ok {
		t.Fatalf("want type *SimpleWriter, got %v", reflect.TypeOf(w))
	}
	cw.AddMetrics(cw.Source.GetMetricsKeys()...)

	// change writer for testing
	var buf bytes.Buffer
	cw.SetDestination(&buf)
	if err := cw.FlushWithKeys("unknown", "a", "unknown", "b", "s2", "unknown"); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	// check valid json
	var dummy interface{}
	if err := json.Unmarshal(buf.Bytes(), &dummy); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	expect := []byte(`{"a":1.0,"b":1.0,"s2":["A'"]}`)
	if !reflect.DeepEqual(buf.Bytes(), expect) {
		t.Errorf("want %s, got %s", buf.Bytes(), expect)
	}
}

func TestStream(t *testing.T) {
	expect := `{"a":1.0,"b":1.0,"c":1.0}`
	cw := createDummySimpleWriterWithKeys(t, nil, []string{"a", "b", "c"}...)
	cw.AddMetrics(cw.Source.GetMetricsKeys()...)

	nw := noticeio.NewBufferWithChannel(nil, make(chan error, 1))
	cw.SetDestination(nw)
	cw.Interval = 10 * time.Millisecond

	// testing cancel
	ctx, cancel := context.WithCancel(context.Background())
	cw.RunStream(ctx)
	before := runtime.NumGoroutine()
	for i := 0; i < 2; i++ {
		// waiting write twice
		<-nw.WriteCh
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
	if after := runtime.NumGoroutine(); before-1 != after {
		t.Errorf("want num %d, got %d", before-1, after)
	}

	// testing writer
	bs := make([]byte, 1024)
	n, err := nw.Read(bs)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	message := string(bs[:n])
	if !strings.HasPrefix(message, expect) {
		t.Errorf("want has prefix %s, got %s", expect, message[:len(expect)])
	}
}
