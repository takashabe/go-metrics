package forward

import (
	"io"
	"net"
	"sort"
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

func (cw *NetWriter) RemoveMetrics(metrics ...string) error {
	for _, m := range metrics {
		for k, v := range cw.MetricsKeys {
			if m == v {
				cw.MetricsKeys = append(cw.MetricsKeys[:k], cw.MetricsKeys[k+1:]...)
			}
		}
	}
	return nil
}

func (cw *NetWriter) Flush() error {
	buf, err := getMergedMetrics(cw.Source, cw.MetricsKeys...)
	if err != nil {
		return err
	}
	_, err = io.Copy(cw.Destination, buf)
	return err
}
