package collect

import (
	"fmt"
	"testing"
)

func TestCounter(t *testing.T) {
	sc := NewSimpleCollector()
	cases := []struct {
		key    string
		value  float64
		expect string
	}{
		{"a", 1, "1"},
		{"a", 2, "3"},
		{"b", 2, "2"},
	}
	for i, c := range cases {
		sc.Add(c.key, c.value)
		actual := sc.metrics[c.key].Aggregate()
		if got := actual[c.key].String(); got != c.expect {
			t.Errorf("#%d: want value %s, got %s", i, c.expect, got)
		}
	}
}

func TestGauge(t *testing.T) {
	sc := NewSimpleCollector()
	cases := []struct {
		key    string
		value  float64
		expect string
	}{
		{"a", 1, "1"},
		{"a", 2, "2"},
		{"b", 1, "1"},
	}
	for i, c := range cases {
		sc.Gauge(c.key, c.value)
		actual := sc.metrics[c.key].Aggregate()
		if got := actual[c.key].String(); got != c.expect {
			t.Errorf("#%d: want value %s, got %s", i, c.expect, got)
		}
	}
}

func TestHistogram(t *testing.T) {
	sc := NewSimpleCollector()
	cases := []struct {
		key    string
		values []float64
		expect map[string]float64
	}{
		{
			"a",
			[]float64{
				5, 5, 3, 3, 10,
			},
			map[string]float64{
				"a.count":        5,
				"a.avg":          5.2,
				"a.max":          10,
				"a.median":       5,
				"a.95percentile": 0,
			},
		},
		{
			"b",
			[]float64{
				10, 5, 5, 2, 3, 40, 10, 10, 10, 9,
			},
			map[string]float64{
				"b.count":        10,
				"b.avg":          10.4,
				"b.max":          40,
				"b.median":       10,
				"b.95percentile": 40,
			},
		},
	}
	for i, c := range cases {
		for _, v := range c.values {
			sc.Histogram(c.key, v)
		}
		actual := sc.metrics[c.key].Aggregate()
		if len(actual) != len(c.expect) {
			t.Fatalf("#%d: want size %d, got %d", i, len(c.expect), len(actual))
		}
		for ek, ev := range c.expect {
			av := actual[ek].String()
			if av != fmt.Sprint(ev) {
				t.Errorf("#%d-%s: want %s, got %s", i, ek, fmt.Sprint(ev), av)
			}
		}
	}
}

func TestMix(t *testing.T) {
	c := NewSimpleCollector()
	c.Add("test.c", 2)
	c.Histogram("test.h", 10.5)
	c.Histogram("test.h", 10)
	c.Set("test.s", "a")
	c.Set("test.s", "b")
}
