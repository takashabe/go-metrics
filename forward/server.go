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

func (cw *NetWriter) SetSource(c collect.Collector) {
	cw.Source = c
}

func (cw *NetWriter) SetDestination(w io.Writer) {
	cw.Destination = w
}

func (cw *NetWriter) AddMetrics(metrics ...string) error {
	s, err := addMetrics(cw.Source.GetMetricsKeys(), cw.MetricsKeys, metrics)
	if err != nil {
		return err
	}

	cw.MetricsKeys = s
	return nil
}

func (cw *NetWriter) RemoveMetrics(metrics ...string) error {
	cw.MetricsKeys = subSlice(cw.MetricsKeys, metrics)
	return nil
}

func (cw *NetWriter) Flush() error {
	return flush(cw.Source, cw.Destination, cw.MetricsKeys...)
}

func (cw *NetWriter) RunStream(ctx context.Context) {
	go runStream(ctx, cw, cw.Interval)
}
