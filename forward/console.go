package forward

import (
	"bytes"
	"io"
	"os"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/takashabe/go-metrics/collect"
)

var (
	ErrInvalidCollector = errors.New("invalid collector source")
	ErrNotExistMetrics  = errors.New("not exist metrics key")
)

// MetricsWriter is write metrics interface
type MetricsWriter interface {
	SetSource(c collect.Collector)
	SetDestination(w io.Writer) error
	AddMetrics(metrics ...string) error
	RemoveMetrics(metrics ...string) error
	Flush() error
}

// ConsoleWriter is implemented MetricsWriter. forward to console
type ConsoleWriter struct {
	Source      collect.Collector
	Destination io.Writer
	MetricsKeys []string
	Interval    time.Duration
}

func NewConsoleWriter(c collect.Collector) (MetricsWriter, error) {
	if c == nil {
		return nil, ErrInvalidCollector
	}

	cw := &ConsoleWriter{
		MetricsKeys: make([]string, 0),
		Interval:    time.Second,
	}
	cw.SetSource(c)
	cw.SetDestination(os.Stdout)
	return cw, nil
}

func (cw *ConsoleWriter) SetSource(c collect.Collector) {
	cw.Source = c
}

func (cw *ConsoleWriter) SetDestination(w io.Writer) error {
	cw.Destination = w
	return nil
}

func (cw *ConsoleWriter) AddMetrics(metrics ...string) error {
	exists := cw.Source.GetMetricsKeys()
	existMap := make(map[string]struct{})
	for _, v := range exists {
		existMap[v] = struct{}{}
	}

	for _, key := range metrics {
		if _, ok := existMap[key]; !ok {
			return ErrNotExistMetrics
		}
	}

	cw.MetricsKeys = append(cw.MetricsKeys, metrics...)
	sort.Strings(cw.MetricsKeys)
	return nil
}

func (cw *ConsoleWriter) RemoveMetrics(metrics ...string) error {
	for _, m := range metrics {
		for k, v := range cw.MetricsKeys {
			if m == v {
				cw.MetricsKeys = append(cw.MetricsKeys[:k], cw.MetricsKeys[k+1:]...)
			}
		}
	}
	return nil
}

func (cw *ConsoleWriter) Flush() error {
	keys := cw.MetricsKeys
	max := len(keys)
	var buf bytes.Buffer
	buf.WriteByte('{')
	for k, v := range keys {
		b, err := cw.Source.GetMetrics(v)
		if err != nil {
			return err
		}
		// trim "{}" and merge all metrics
		b = bytes.TrimLeft(b, "{")
		b = bytes.TrimRight(b, "}")
		buf.Write(b)
		if k != max-1 {
			buf.WriteByte(',')
		}
	}
	buf.WriteByte('}')
	_, err := cw.Destination.Write(buf.Bytes())
	return err
}
