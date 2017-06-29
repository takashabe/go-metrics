package forward

import (
	"reflect"
	"testing"

	"github.com/takashabe/go-metrics/collect"
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
