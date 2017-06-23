package collect

import "testing"

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

func TestMix(t *testing.T) {
	c := NewSimpleCollector()
	c.Add("test.c", 2)
	c.Histogram("test.h", 10.5)
	c.Histogram("test.h", 10)
	c.Set("test.s", "a")
	c.Set("test.s", "b")
}
