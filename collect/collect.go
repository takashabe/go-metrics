package collect

import (
	"fmt"
	"sync"
)

// MetricType is metric types
type MetricType int

const (
	_ MetricType = iota
	Counter
	Gauge
	Histogram
	Set
)

func (m MetricType) String() string {
	switch m {
	case Counter:
		return "counter"
	case Gauge:
		return "gauge"
	case Histogram:
		return "histogram"
	case Set:
		return "set"
	default:
		return "not supported metric type"
	}
}

// Metrics is a metrics key and value
type Metrics interface {
	Aggregate() map[string]Data
	GetType() MetricType
}

// CounterMetrics is implemented Metirics for Counter
type CounterMetrics struct {
	key   string
	value *Float
}

func (m *CounterMetrics) Aggregate() map[string]Data {
	return map[string]Data{
		m.key: m.value,
	}
}

func (m *CounterMetrics) GetType() MetricType {
	return Counter
}

// HistogramMetrics is implemented Metirics for Histogram
type HistogramMetrics struct {
	key   string
	value *FloatSlice
}

func (m *HistogramMetrics) Aggregate() map[string]Data {
	// TODO
	return map[string]Data{
		m.key + ".count":        m.count(),
		m.key + ".avg":          m.count(),
		m.key + ".max":          m.count(),
		m.key + ".median":       m.count(),
		m.key + ".95percentile": m.count(),
	}
}

func (m *HistogramMetrics) count() Data {
	return &Float{
		f: float64(len(m.value.v)),
	}
}

func (m *HistogramMetrics) GetType() MetricType {
	return Histogram
}

// SetMetrics is implemented Metrics for Set
type SetMetrics struct {
	key   string
	value *Map
}

func (m *SetMetrics) Aggregate() map[string]Data {
	m.value.RLock()
	defer m.value.Unlock()
	s := make([]string, len(m.value.v))
	for k, _ := range m.value.v {
		s = append(s, k)
	}

	return map[string]Data{
		m.key: &StringSlice{
			s: s,
		},
	}
}

func (m *SetMetrics) GetType() MetricType {
	return Set
}

// Data is a metrics value
type Data interface {
	String() string
}

// Float is implemented Data
type Float struct {
	f float64
	sync.RWMutex
}

func (f *Float) String() string {
	f.RLock()
	defer f.RUnlock()
	return fmt.Sprint(f.f)
}

func (f *Float) Add(delta float64) {
	f.Lock()
	defer f.Unlock()
	f.f += delta
}

// StringSlice is implemented Data
type StringSlice struct {
	s []string
	sync.RWMutex
}

func (s *StringSlice) String() string {
	s.Lock()
	defer s.Unlock()
	return fmt.Sprint(s.s)
}

type FloatSlice struct {
	v []float64
	sync.RWMutex
}

type Map struct {
	v map[string]struct{}
	sync.RWMutex
}

func (m *Map) set(s string) {
	m.Lock()
	defer m.Unlock()
	m.v[s] = struct{}{}
}

type SimpleCollector struct {
	metrics map[string]Metrics
	sync.RWMutex
}

func NewSimpleCollector() *SimpleCollector {
	return &SimpleCollector{
		metrics: make(map[string]Metrics),
	}
}

// Add add count for CounterMetrics
func (c *SimpleCollector) Add(key string, delta float64) {
	c.Lock()
	defer c.Unlock()

	// add key
	if _, dup := c.metrics[key]; !dup {
		c.metrics[key] = &CounterMetrics{
			key:   key,
			value: &Float{},
		}
	}

	// incremental counter, ignore otherwise
	if v, ok := c.metrics[key].(*CounterMetrics); ok {
		v.value.Add(delta)
	}
}

// Histogram add metrics for Histogram
func (c *SimpleCollector) Histogram(key string, delta float64) {
	c.Lock()
	defer c.Unlock()

	// add key
	if _, dup := c.metrics[key]; !dup {
		c.metrics[key] = &HistogramMetrics{
			key: key,
			value: &FloatSlice{
				v: make([]float64, 0),
			},
		}
	}

	// add histogram, ignore otherwise
	if v, ok := c.metrics[key].(*HistogramMetrics); ok {
		v.value.v = append(v.value.v, delta)
	}
}

// Set add metrics for Set
func (c *SimpleCollector) Set(key string, delta string) {
	c.Lock()
	defer c.Unlock()

	// add key
	if _, dup := c.metrics[key]; !dup {
		c.metrics[key] = &SetMetrics{
			key: key,
			value: &Map{
				v: make(map[string]struct{}),
			},
		}
	}

	// add set, ignore otherwise
	if v, ok := c.metrics[key].(*SetMetrics); ok {
		v.value.set(delta)
	}
}
