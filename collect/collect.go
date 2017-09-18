// Package collect implements collect metrics data
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

// Enum of MetricType
const (
	_ MetricType = iota
	TypeCounter
	TypeGauge
	TypeHistogram
	TypeSet
	TypeSnapshot
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
	case TypeSnapshot:
		return "snapshot"
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

// Aggregate return counter key and value
func (m *CounterMetrics) Aggregate() map[string]Data {
	m.value.mu.RLock()
	defer m.value.mu.RUnlock()
	return map[string]Data{
		m.key: m.value,
	}
}

// GetType return MetricType
func (m *CounterMetrics) GetType() MetricType {
	return TypeCounter
}

// GaugeMetrics is implemented Metrics for Gauge
type GaugeMetrics struct {
	key   string
	value *Float
}

// Aggregate return gauge key and value
func (m *GaugeMetrics) Aggregate() map[string]Data {
	return map[string]Data{
		m.key: m.value,
	}
}

// GetType return MetricType
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

// Aggregate returns aggregated histogram metrics
func (m *HistogramMetrics) Aggregate() map[string]Data {
	m.value.mu.Lock()
	sort.Float64s(m.value.v)
	m.value.mu.Unlock()

	return map[string]Data{
		m.key + ".count":        m.count(),
		m.key + ".avg":          m.average(),
		m.key + ".max":          m.max(),
		m.key + ".median":       m.median(),
		m.key + ".95percentile": m.percentile(0.95),
	}
}

// MarshalJSONWithOrder return keeped order json
func (m *HistogramMetrics) MarshalJSONWithOrder() ([]byte, error) {
	sortKeys := make([]string, 0)
	agg := m.Aggregate()
	for k := range agg {
		sortKeys = append(sortKeys, k)
	}
	sort.Strings(sortKeys)

	// create json
	var buf bytes.Buffer
	buf.Write([]byte("{"))
	for k, v := range sortKeys {
		if k != 0 {
			buf.Write([]byte(","))
		}
		value, err := agg[v].MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write([]byte(fmt.Sprintf("%q:%s", v, value)))
	}
	buf.Write([]byte("}"))
	return buf.Bytes(), nil
}

func (m *HistogramMetrics) count() Data {
	m.value.mu.RLock()
	defer m.value.mu.RUnlock()
	return &Float{
		f: float64(len(m.value.v)),
	}
}

func (m *HistogramMetrics) average() Data {
	m.value.mu.RLock()
	defer m.value.mu.RUnlock()
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
	m.value.mu.RLock()
	defer m.value.mu.RUnlock()
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
	m.value.mu.RLock()
	defer m.value.mu.RUnlock()
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
	m.value.mu.RLock()
	defer m.value.mu.RUnlock()
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

// GetType return MetricType
func (m *HistogramMetrics) GetType() MetricType {
	return TypeHistogram
}

// SetMetrics is implemented Metrics for Set
type SetMetrics struct {
	key   string
	value *Map
}

// Aggregate return sorted sort key and value
func (m *SetMetrics) Aggregate() map[string]Data {
	m.value.mu.RLock()
	defer m.value.mu.RUnlock()
	s := make([]string, 0)
	for k := range m.value.v {
		s = append(s, k)
	}
	sort.Strings(s)

	return map[string]Data{
		m.key: &StringSlice{
			s: s,
		},
	}
}

// GetType return MetricType
func (m *SetMetrics) GetType() MetricType {
	return TypeSet
}

// SnapshotMetrics is implemented Metrics for Snapshot
type SnapshotMetrics struct {
	key   string
	value *Map
}

// Aggregate return sorted sort key and value
func (m *SnapshotMetrics) Aggregate() map[string]Data {
	m.value.mu.RLock()
	defer m.value.mu.RUnlock()
	s := make([]string, 0)
	for k := range m.value.v {
		s = append(s, k)
	}
	sort.Strings(s)

	return map[string]Data{
		m.key: &StringSlice{
			s: s,
		},
	}
}

// GetType return MetricType
func (m *SnapshotMetrics) GetType() MetricType {
	return TypeSnapshot
}

// Data is a metrics value
type Data interface {
	MarshalJSON() ([]byte, error)
}

// Float is implemented Data
type Float struct {
	f  float64
	mu sync.RWMutex
}

// MarshalJSON return specific encoded json
func (f *Float) MarshalJSON() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return []byte(fmt.Sprintf("%.1f", f.f)), nil
}

func (f *Float) add(delta float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.f += delta
}

func (f *Float) set(delta float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.f = delta
}

// StringSlice is implemented Data
type StringSlice struct {
	s  []string
	mu sync.RWMutex
}

// MarshalJSON return specific encoded json
func (s *StringSlice) MarshalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
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

// FloatSlice is used by collect metrics
type FloatSlice struct {
	v  []float64
	mu sync.RWMutex
}

// Map is used by collect metrics
type Map struct {
	v  map[string]struct{}
	mu sync.RWMutex
}

func (m *Map) set(s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	mu      sync.RWMutex
}

// NewSimpleCollector return new SimpleCollector
func NewSimpleCollector() *SimpleCollector {
	return &SimpleCollector{
		metrics: make(map[string]Metrics),
	}
}

// GetMetrics returns json from encoded metrics
func (c *SimpleCollector) GetMetrics(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m, ok := c.metrics[key]
	if !ok {
		return nil, ErrNotFoundMetrics
	}

	// need sort keys?
	switch m := m.(type) {
	case *HistogramMetrics:
		return m.MarshalJSONWithOrder()
	default:
		return json.Marshal(m.Aggregate())
	}
}

// GetMetricsKeys returns keeps metrics keys
func (c *SimpleCollector) GetMetricsKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]string, 0)
	for k := range c.metrics {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

// Add add count for CounterMetrics
func (c *SimpleCollector) Add(key string, delta float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

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

// Snapshot add metrics for Snapshot
func (c *SimpleCollector) Snapshot(key string, deltas []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// add key and ignore existing key
	c.metrics[key] = &SnapshotMetrics{
		key: key,
		value: &Map{
			v: make(map[string]struct{}),
		},
	}

	// add Snapshot, ignore otherwise
	if v, ok := c.metrics[key].(*SnapshotMetrics); ok {
		for _, d := range deltas {
			v.value.set(d)
		}
	}
}
