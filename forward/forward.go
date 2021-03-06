// Package forward implements collected metrics data forward to other resouce
package forward

import (
	"bytes"
	"context"
	"io"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/takashabe/go-metrics/collect"
)

// Error variables
var (
	ErrInvalidCollector = errors.New("invalid collector source")
	ErrNotExistMetrics  = errors.New("not exist metrics key")
)

// MetricsWriter is write metrics interface
type MetricsWriter interface {
	SetSource(c collect.Collector)
	SetDestination(w io.Writer)
	AddMetrics(metrics ...string) error
	RemoveMetrics(metrics ...string) error
	Flush() error
	FlushWithKeys(keys ...string) error
	RunStream(ctx context.Context)
}

// SimpleWriter is implemented MetricsWriter. forward to any io.Writer
type SimpleWriter struct {
	Source      collect.Collector
	Destination io.Writer
	MetricsKeys []string
	Interval    time.Duration
}

// NewSimpleWriter return new SimpleWriter
func NewSimpleWriter(c collect.Collector, w io.Writer) (MetricsWriter, error) {
	if c == nil {
		return nil, ErrInvalidCollector
	}

	cw := &SimpleWriter{
		MetricsKeys: make([]string, 0),
		Interval:    time.Second,
	}
	cw.SetSource(c)
	cw.SetDestination(w)
	return cw, nil
}

// SetSource setting collector
func (cw *SimpleWriter) SetSource(c collect.Collector) {
	cw.Source = c
}

// SetDestination setting writer
func (cw *SimpleWriter) SetDestination(w io.Writer) {
	cw.Destination = w
}

// AddMetrics register metrics keys when exist collector
func (cw *SimpleWriter) AddMetrics(metrics ...string) error {
	s, err := addMetrics(cw.Source.GetMetricsKeys(), cw.MetricsKeys, metrics)
	if err != nil {
		return err
	}

	cw.MetricsKeys = s
	return nil
}

func addMetrics(src []string, dst []string, adds []string) ([]string, error) {
	existMap := make(map[string]struct{})
	for _, v := range src {
		existMap[v] = struct{}{}
	}

	for _, key := range adds {
		if _, ok := existMap[key]; !ok {
			return nil, ErrNotExistMetrics
		}
	}

	dst = append(dst, adds...)
	sort.Strings(dst)
	return dst, nil
}

// RemoveMetrics remove metrics
func (cw *SimpleWriter) RemoveMetrics(metrics ...string) error {
	cw.MetricsKeys = subSlice(cw.MetricsKeys, metrics)
	return nil
}

func subSlice(source []string, removes []string) []string {
	for _, m := range removes {
		for k, v := range source {
			if m == v {
				source = append(source[:k], source[k+1:]...)
			}
		}
	}
	return source
}

// Flush write metrics data for destination writer
func (cw *SimpleWriter) Flush() error {
	return flush(cw.Source, cw.Destination, cw.MetricsKeys...)
}

// FlushWithKeys write specific metrics data for destination writer
func (cw *SimpleWriter) FlushWithKeys(keys ...string) error {
	return flush(cw.Source, cw.Destination, keys...)
}

// flush write to Destination writer from collector
func flush(c collect.Collector, w io.Writer, keys ...string) error {
	buf, err := getMergedMetrics(c, keys...)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, buf)
	return err
}

// getMergedMetrics return a merged metrics data
func getMergedMetrics(c collect.Collector, keys ...string) (*bytes.Buffer, error) {
	var (
		buf      bytes.Buffer
		existKey bool
	)
	buf.WriteByte('{')
	for k, v := range keys {
		b, err := c.GetMetrics(v)
		if err != nil {
			// ignore case
			if errors.Cause(err) == collect.ErrNotFoundMetrics {
				if k == len(keys)-1 {
					buf.Truncate(buf.Len() - 1)
				}
				continue
			}
			return nil, err
		}
		existKey = true

		// trim "{}" and merge all metrics
		b = bytes.TrimLeft(b, "{")
		b = bytes.TrimRight(b, "}")
		buf.Write(b)
		if k != len(keys)-1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte('}')

	if !existKey {
		return nil, collect.ErrNotFoundMetrics
	}
	return &buf, nil
}

// RunStream run Flush() goroutine
func (cw *SimpleWriter) RunStream(ctx context.Context) {
	go runStream(ctx, cw, cw.Interval)
}

func runStream(ctx context.Context, writer MetricsWriter, interval time.Duration) {
	t := time.NewTicker(interval)
	for {
		select {
		case <-t.C:
			writer.Flush()
		case <-ctx.Done():
			t.Stop()
			return
		}
	}
}
