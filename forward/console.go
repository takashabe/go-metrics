package forward

import (
	"io"

	"github.com/takashabe/go-metrics/collect"
)

// MetricsWriter is write metrics interface
type MetricsWriter interface {
	SetSource(c *collect.Collector, metrics ...string)
	SetDestination(w io.Writer) error
	AddMetrics(metrics ...string) error
	RemoveMetrics(metrics ...string) error
	Flush() ([]byte, error)
}
