package collect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/pkg/errors"
)

// about metrics errors
var (
	ErrNotFoundMetrics = errors.New("not found metrics")
)

// MetricType is metric types
type MetricType int

const (
	_ MetricType = iota
	TypeCounter
	TypeGauge
	TypeHistogram
	TypeSet
)

func (m MetricType) String() string {
	switch m {
	case TypeCounter:
		return "counter"
	case TypeGauge:
		return "gauge"
	case TypeHistogram:
		return "histogram"
	case TypeSet:
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
	m.value.RLock()
	defer m.value.RUnlock()
	return map[string]Data{
		m.key: m.value,
	}
}

func (m *CounterMetrics) GetType() MetricType {
	return TypeCounter
}

// GaugeMetrics is implemented Metrics for Gauge
type GaugeMetrics struct {
	key   string
	value *Float
}

func (m *GaugeMetrics) Aggregate() map[string]Data {
	return map[string]Data{
		m.key: m.value,
	}
}

func (m *GaugeMetrics) GetType() MetricType {
	return TypeGauge
}

// HistogramMetrics is implemented Metirics for Histogram
type HistogramMetrics struct {
	key   string
	value *FloatSlice
}

// minPercentileSize is minimum number of size for percentile analysis
const minPercentileSize = 2

func (m *HistogramMetrics) Aggregate() map[string]Data {
	m.value.Lock()
	sort.Float64s(m.value.v)
	m.value.Unlock()

	return map[string]Data{
		m.key + ".count":        m.count(),
		m.key + ".avg":          m.average(),
		m.key + ".max":          m.max(),
		m.key + ".median":       m.median(),
		m.key + ".95percentile": m.percentile(0.95),
	}
}

func (m *HistogramMetrics) count() Data {
	m.value.RLock()
	defer m.value.RUnlock()
	return &Float{
		f: float64(len(m.value.v)),
	}
}

func (m *HistogramMetrics) average() Data {
	m.value.RLock()
	defer m.value.RUnlock()
	var total float64
	for _, v := range m.value.v {
		total += v
	}
	return &Float{
		f: total / float64(len(m.value.v)),
	}
}

// note: expect sorted list
func (m *HistogramMetrics) max() Data {
	m.value.RLock()
	defer m.value.RUnlock()
	var max float64
	if size := len(m.value.v); size > 0 {
		max = m.value.v[size-1]
	}
	return &Float{
		f: max,
	}
}

// note: expect sorted list
func (m *HistogramMetrics) median() Data {
	m.value.RLock()
	defer m.value.RUnlock()
	var median float64
	if size := len(m.value.v); size > 0 {
		median = m.value.v[size/2]
	}
	return &Float{
		f: median,
	}
}

// note: expect sorted list
func (m *HistogramMetrics) percentile(n float64) Data {
	m.value.RLock()
	defer m.value.RUnlock()
	list := m.value.v
	if 1.0 <= n || len(list) < minPercentileSize {
		return &Float{}
	}

	// use linear interpolation
	// see type R-7: https://en.wikipedia.org/wiki/Quantile
	r := 1 + float64((len(list)-1))*n
	rFloor := int(math.Floor(r))
	rCeil := int(math.Ceil(r))
	// -1 means in accordance with slice index
	q := list[rFloor-1] + (r-float64(rFloor))*(list[rCeil-1]-list[rFloor-1])

	return &Float{
		f: q,
	}
}

func (m *HistogramMetrics) GetType() MetricType {
	return TypeHistogram
}

// SetMetrics is implemented Metrics for Set
type SetMetrics struct {
	key   string
	value *Map
}

func (m *SetMetrics) Aggregate() map[string]Data {
	m.value.RLock()
	defer m.value.RUnlock()
	s := make([]string, 0)
	for k, _ := range m.value.v {
		s = append(s, k)
	}
	sort.Strings(s)

	return map[string]Data{
		m.key: &StringSlice{
			s: s,
		},
	}
}

func (m *SetMetrics) GetType() MetricType {
	return TypeSet
}

// Data is a metrics value
type Data interface {
	MarshalJSON() ([]byte, error)
}

// Float is implemented Data
type Float struct {
	f float64
	sync.RWMutex
}

func (f *Float) MarshalJSON() ([]byte, error) {
	f.RLock()
	defer f.RUnlock()
	return []byte(fmt.Sprintf("%.1f", f.f)), nil
}

func (f *Float) add(delta float64) {
	f.Lock()
	defer f.Unlock()
	f.f += delta
}

func (f *Float) set(delta float64) {
	f.Lock()
	defer f.Unlock()
	f.f = delta
}

// StringSlice is implemented Data
type StringSlice struct {
	s []string
	sync.RWMutex
}

func (s *StringSlice) MarshalJSON() ([]byte, error) {
	s.RLock()
	defer s.RUnlock()
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "[")
	for k, v := range s.s {
		if k != 0 {
			fmt.Fprintf(&buf, ",")
		}
		fmt.Fprintf(&buf, "%q", v)
	}
	fmt.Fprintf(&buf, "]")
	return buf.Bytes(), nil
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

// Collector is collect metrics interface
type Collector interface {
	// metrics accessor
	GetMetrics(string) ([]byte, error)
	GetMetricsKeys() []string

	// collect metrics functions
	Add(string, float64)
	Gauge(string, float64)
	Histogram(string, float64)
	Set(string, string)
}

// SimpleCollector is implemented Collector
type SimpleCollector struct {
	metrics map[string]Metrics
	sync.RWMutex
}

func NewSimpleCollector() *SimpleCollector {
	return &SimpleCollector{
		metrics: make(map[string]Metrics),
	}
}

func (c *SimpleCollector) GetMetrics(key string) ([]byte, error) {
	c.RLock()
	defer c.RUnlock()

	m, ok := c.metrics[key]
	if !ok {
		return nil, ErrNotFoundMetrics
	}
	return json.Marshal(m.Aggregate())
}

func (c *SimpleCollector) GetMetricsKeys() []string {
	c.RLock()
	defer c.RUnlock()
	res := make([]string, 0)
	for k, _ := range c.metrics {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
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
		v.value.add(delta)
	}
}

// Gauge set metrics for GaugeMetrics
func (c *SimpleCollector) Gauge(key string, delta float64) {
	c.Lock()
	defer c.Unlock()

	// add key
	if _, dup := c.metrics[key]; !dup {
		c.metrics[key] = &GaugeMetrics{
			key:   key,
			value: &Float{},
		}
	}

	// incremental counter, ignore otherwise
	if v, ok := c.metrics[key].(*GaugeMetrics); ok {
		v.value.set(delta)
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
