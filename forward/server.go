package forward

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/takashabe/go-metrics/collect"
)

// NetWriter is forward to udp server
type NetWriter struct {
	Source      collect.Collector
	Destination io.Writer
	MetricsKeys []string
	Interval    time.Duration
}

// NewNetWriter return new NetWriter
func NewNetWriter(c collect.Collector, addr string) (MetricsWriter, error) {
	if c == nil {
		return nil, ErrInvalidCollector
	}

	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}

	w := &NetWriter{
		MetricsKeys: make([]string, 0),
		Interval:    time.Second,
	}
	w.SetSource(c)
	w.SetDestination(conn)
	return w, nil
}

// SetSource setting collector
func (cw *NetWriter) SetSource(c collect.Collector) {
	cw.Source = c
}

// SetDestination setting writer
func (cw *NetWriter) SetDestination(w io.Writer) {
	cw.Destination = w
}

// AddMetrics register metrics keys when exist collector
func (cw *NetWriter) AddMetrics(metrics ...string) error {
	s, err := addMetrics(cw.Source.GetMetricsKeys(), cw.MetricsKeys, metrics)
	if err != nil {
		return err
	}

	cw.MetricsKeys = s
	return nil
}

// RemoveMetrics remove metrics
func (cw *NetWriter) RemoveMetrics(metrics ...string) error {
	cw.MetricsKeys = subSlice(cw.MetricsKeys, metrics)
	return nil
}

// Flush write metrics data for destination writer
func (cw *NetWriter) Flush() error {
	return flush(cw.Source, cw.Destination, cw.MetricsKeys...)
}

// FlushWithKeys write specific metrics data for destination writer
func (cw *NetWriter) FlushWithKeys(keys ...string) error {
	return flush(cw.Source, cw.Destination, keys...)
}

// RunStream run Flush() goroutine
func (cw *NetWriter) RunStream(ctx context.Context) {
	go runStream(ctx, cw, cw.Interval)
}
