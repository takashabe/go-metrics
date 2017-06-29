package forward

import (
	"reflect"
	"testing"

	"github.com/takashabe/go-metrics/collect"
)

func TestAddMetrics(t *testing.T) {
	sc := collect.NewSimpleCollector()
	sc.Add("a", 1)
	sc.Add("b", 1)
	sc.Add("c", 1)
	cw := NewConsoleWriter()
	cw.SetSource(sc)

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
	keys := []string{"a", "b", "c"}
	sc := collect.NewSimpleCollector()
	for _, v := range keys {
		sc.Add(v, 1)
	}
	cw := NewConsoleWriter()
	cw.SetSource(sc)
	err := cw.AddMetrics(keys...)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

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
