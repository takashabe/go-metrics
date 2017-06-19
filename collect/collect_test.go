package collect

import "testing"

func TestCounter(t *testing.T) {
	c := NewSimpleCollector()
	c.Add("test.a", 1)
	c.Add("test.b", 2)
}

func TestGauge(t *testing.T) {
	c := NewSimpleCollector()
	c.Gauge("t", 1)
	c.Gauge("t", 2)
}

func TestMix(t *testing.T) {
	c := NewSimpleCollector()
	c.Add("test.c", 2)
	c.Histogram("test.h", 10.5)
	c.Histogram("test.h", 10)
	c.Set("test.s", "a")
	c.Set("test.s", "b")
}
