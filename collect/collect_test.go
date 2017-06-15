package collect

import "testing"

func TestCounter(t *testing.T) {
	// expected
	c := NewCollector()
	c.Incr("test.a")
	c.Incr("test.b")
}

func TestMix(t *testing.T) {
	c := NewCollector()
	c.Incr("test.c", 1)
	c.Histogram("test.h", 50)
	c.Histogram("test.h", 100)
	c.Set("test.s", "a")
	c.Set("test.s", "b")

	expect := nil
	if c.GetPrefix("test") != expect {
		t.Errorf("want %v, but %v", expect, c.GetPrefix("test"))
	}
}
