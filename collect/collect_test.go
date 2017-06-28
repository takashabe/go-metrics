package collect

import (
	"reflect"
	"testing"
)

func TestCounter(t *testing.T) {
	sc := NewSimpleCollector()
	cases := []struct {
		key    string
		value  float64
		expect []byte
	}{
		{"a", 1, []byte("1.0")},
		{"a", 2, []byte("3.0")},
		{"b", 2, []byte("2.0")},
	}
	for i, c := range cases {
		sc.Add(c.key, c.value)
		if got := sc.metrics[c.key].GetType(); got != TypeCounter {
			t.Fatalf("#%d: want type %s, got %s", i, TypeCounter, got)
		}
		agg := sc.metrics[c.key].Aggregate()
		got, err := agg[c.key].MarshalJSON()
		if err != nil {
			t.Fatalf("#%d: want no error, got %v", i, err)
		}
		if !reflect.DeepEqual(got, c.expect) {
			t.Errorf("#%d: want value %s, got %s", i, c.expect, got)
		}
	}
}

func TestGauge(t *testing.T) {
	sc := NewSimpleCollector()
	cases := []struct {
		key    string
		value  float64
		expect []byte
	}{
		{"a", 1, []byte("1.0")},
		{"a", 2, []byte("2.0")},
		{"b", 1, []byte("1.0")},
	}
	for i, c := range cases {
		sc.Gauge(c.key, c.value)
		if got := sc.metrics[c.key].GetType(); got != TypeGauge {
			t.Fatalf("#%d: want type %s, got %s", i, TypeGauge, got)
		}
		agg := sc.metrics[c.key].Aggregate()
		got, err := agg[c.key].MarshalJSON()
		if err != nil {
			t.Fatalf("#%d: want no error, got %v", i, err)
		}
		if !reflect.DeepEqual(got, c.expect) {
			t.Errorf("#%d: want value %s, got %s", i, c.expect, got)
		}
	}
}

func TestHistogram(t *testing.T) {
	sc := NewSimpleCollector()
	cases := []struct {
		key    string
		values []float64
		expect map[string][]byte
	}{
		{
			"a",
			[]float64{
				5,
			},
			map[string][]byte{
				"a.count":        []byte("1.0"),
				"a.avg":          []byte("5.0"),
				"a.max":          []byte("5.0"),
				"a.median":       []byte("5.0"),
				"a.95percentile": []byte("0.0"),
			},
		},
		{
			"b",
			[]float64{
				1, 2,
			},
			map[string][]byte{
				"b.count":        []byte("2.0"),
				"b.avg":          []byte("1.5"),
				"b.max":          []byte("2.0"),
				"b.median":       []byte("2.0"),
				"b.95percentile": []byte("1.9"),
			},
		},
		{
			"c",
			[]float64{
				10, 5, 5, 2, 3, 40, 10, 10, 10, 9,
			},
			map[string][]byte{
				"c.count":        []byte("10.0"),
				"c.avg":          []byte("10.4"),
				"c.max":          []byte("40.0"),
				"c.median":       []byte("10.0"),
				"c.95percentile": []byte("26.5"),
			},
		},
	}
	for i, c := range cases {
		for _, v := range c.values {
			sc.Histogram(c.key, v)
		}
		if got := sc.metrics[c.key].GetType(); got != TypeHistogram {
			t.Fatalf("#%d: want type %s, got %s", i, TypeHistogram, got)
		}
		agg := sc.metrics[c.key].Aggregate()
		if len(agg) != len(c.expect) {
			t.Fatalf("#%d: want size %d, got %d", i, len(c.expect), len(agg))
		}
		for ek, ev := range c.expect {
			av, err := agg[ek].MarshalJSON()
			if err != nil {
				t.Fatalf("#%d-%s: want no error, got %v", i, ek, err)
			}
			if !reflect.DeepEqual(av, ev) {
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
		expect []byte
	}{
		{
			"a",
			[]string{
				"1", "1", "2",
			},
			[]byte(`["1","2"]`),
		},
		{
			"b",
			[]string{
				"1",
			},
			[]byte(`["1"]`),
		},
	}
	for i, c := range cases {
		for _, v := range c.values {
			sc.Set(c.key, v)
		}
		if got := sc.metrics[c.key].GetType(); got != TypeSet {
			t.Fatalf("#%d: want type %s, got %s", i, TypeSet, got)
		}
		agg := sc.metrics[c.key].Aggregate()
		got, err := agg[c.key].MarshalJSON()
		if err != nil {
			t.Fatalf("#%d: want no error, got %v", i, err)
		}
		if !reflect.DeepEqual(got, c.expect) {
			t.Errorf("#%d: want value %s, got %s", i, c.expect, got)
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
	expectGauge := []byte("5.0")
	agg := c.metrics["g"].Aggregate()
	got, err := agg["g"].MarshalJSON()
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if !reflect.DeepEqual(got, expectGauge) {
		t.Errorf("want %s, got %s", expectGauge, got)
	}

	expectSet := []byte(`["a","b"]`)
	agg = c.metrics["s"].Aggregate()
	got, err = agg["s"].MarshalJSON()
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if !reflect.DeepEqual(got, expectSet) {
		t.Errorf("want %s, got %s", expectSet, got)
	}

	// expect: can't overwrite at exist keys
	c.Gauge("c", 1)
	if got := c.metrics["c"].GetType(); got != TypeCounter {
		t.Errorf("want %s, got %s", TypeCounter, got)
	}
}
