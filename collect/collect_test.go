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
		{"a", 1, "1.0"},
		{"a", 2, "3.0"},
		{"b", 2, "2.0"},
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
		{"a", 1, "1.0"},
		{"a", 2, "2.0"},
		{"b", 1, "1.0"},
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
		expect map[string]string
	}{
		{
			"a",
			[]float64{
				5,
			},
			map[string]string{
				"a.count":        "1.0",
				"a.avg":          "5.0",
				"a.max":          "5.0",
				"a.median":       "5.0",
				"a.95percentile": "0.0",
			},
		},
		{
			"b",
			[]float64{
				1, 2,
			},
			map[string]string{
				"b.count":        "2.0",
				"b.avg":          "1.5",
				"b.max":          "2.0",
				"b.median":       "2.0",
				"b.95percentile": "1.9",
			},
		},
		{
			"c",
			[]float64{
				10, 5, 5, 2, 3, 40, 10, 10, 10, 9,
			},
			map[string]string{
				"c.count":        "10.0",
				"c.avg":          "10.4",
				"c.max":          "40.0",
				"c.median":       "10.0",
				"c.95percentile": "26.5",
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
			if av != ev {
				t.Errorf("#%d-%s: want %s, got %s", i, ek, ev, av)
			}
		}
	}
}

func TestSet(t *testing.T) {
	sc := NewSimpleCollector()
	cases := []struct {
		key    string
		values []string
		expect []string
	}{
		{
			"a",
			[]string{
				"1", "1", "2",
			},
			[]string{
				"1", "2",
			},
		},
		{
			"b",
			[]string{
				"1",
			},
			[]string{
				"1",
			},
		},
	}
	for i, c := range cases {
		for _, v := range c.values {
			sc.Set(c.key, v)
		}
		actual := sc.metrics[c.key].Aggregate()
		if actual[c.key].String() != fmt.Sprint(c.expect) {
			t.Errorf("#%d: want %s, got %s", i, c.expect, actual[c.key].String())
		}
	}
}

func TestMix(t *testing.T) {
	c := NewSimpleCollector()
	c.Add("c", 2)
	c.Gauge("g", 5)
	c.Histogram("h", 10.5)
	c.Histogram("h", 10)
	c.Set("s", "a")
	c.Set("s", "b")

	// expect: choosable a metrics in mixed metrics collector
	expectGauge := "5.0"
	if got := c.metrics["g"].Aggregate(); got["g"].String() != expectGauge {
		t.Errorf("want %v, got %v", expectGauge, got["g"].String())
	}
	expectSet := fmt.Sprint([]string{"a", "b"})
	if got := c.metrics["s"].Aggregate(); got["s"].String() != expectSet {
		t.Errorf("want %v, got %v", expectSet, got["s"].String())
	}

	// expect: can't overwrite at exist keys
	c.Gauge("c", 1)
	if got := c.metrics["c"].GetType(); got != TypeCounter {
		t.Errorf("want %s, got %s", TypeCounter, got)
	}
}
