# go-metrics

Collecting metrics. The metrics values can be categorized into several types.

## Installation

```
go get -u github.com/takashabe/go-metrics
```

## Usage

* saveMetrics
  * collect metrics
* forwardConsole, forwardUDP
  * forward to the specified io.Writer

detail see at [example/main.go](example/main.go)

```go
package main

import (
  "context"
  "os"
  "time"

  "github.com/takashabe/go-metrics/collect"
  "github.com/takashabe/go-metrics/forward"
)

func main() {
  // collect metrics
  collector := collect.NewSimpleCollector()
  for i := 0; i < 10; i++ {
    saveMetrics(collector)
  }

  // metrics send to console
  cctx, ccancel := context.WithCancel(context.Background())
  forwardConsole(cctx, collector)

  // metrics send to udp server
  // must running server
  uctx, ucancel := context.WithCancel(context.Background())
  forwardUDP(uctx, collector)

  time.Sleep(2 * time.Second)
  ccancel()
  ucancel()

  // output console and udp socket (prepare reformat by jq):
  `
{
  "cnt": 10,
  "histogram.95percentile": 1499763733746145300,
  "histogram.avg": 1499763733746128600,
  "histogram.count": 10,
  "histogram.max": 1499763733746146600,
  "histogram.median": 1499763733746140200,
  "history": [
    "2017-07-11 18:02:13.746027874 +0900 JST",
    "2017-07-11 18:02:13.746132309 +0900 JST",
    "2017-07-11 18:02:13.74613555 +0900 JST",
    "2017-07-11 18:02:13.746137325 +0900 JST",
    "2017-07-11 18:02:13.746138707 +0900 JST",
    "2017-07-11 18:02:13.746140146 +0900 JST",
    "2017-07-11 18:02:13.746141455 +0900 JST",
    "2017-07-11 18:02:13.746142766 +0900 JST",
    "2017-07-11 18:02:13.746144055 +0900 JST",
    "2017-07-11 18:02:13.746146669 +0900 JST"
  ],
  "recent": 1499763733746146600
}`
}

func saveMetrics(c collect.Collector) {
  now := time.Now()

  c.Add("cnt", 1)
  c.Gauge("recent", float64(now.UnixNano()))
  c.Histogram("histogram", float64(now.UnixNano()))
  c.Set("history", now.String())
}

func forwardConsole(ctx context.Context, c collect.Collector) {
  writer, err := forward.NewSimpleWriter(c, os.Stdout)
  if err != nil {
    panic(err)
  }
  writer.AddMetrics(c.GetMetricsKeys()...)
  writer.RunStream(ctx) // metrics will be sent every seconds
}

func forwardUDP(ctx context.Context, c collect.Collector) {
  writer, err := forward.NewNetWriter(c, ":1234")
  if err != nil {
    panic(err)
  }
  writer.AddMetrics(c.GetMetricsKeys()...)
  writer.RunStream(ctx) // metrics will be sent every seconds
}
```

## Metircs type

| Type      | Detail                                                                                                                                                       |
| ---       | ---                                                                                                                                                          |
| Counter   | Used to count things                                                                                                                                         |
| Gauge     | A particular value at a particular time                                                                                                                      |
| Histogram | Represents a statistical distribution of a series of values.<br> Each histogram are `count`, `average`, `minimum`, `maximum`, `median` and `95th percentile` |
| Set       | Used to count the value of unique in a group                                                                                                                 |
