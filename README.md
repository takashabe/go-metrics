# go-stats

Collecting metrics. The metrics values can be categorized into several types.

## Usage

TODO

## metircs type

| Type      | Detail                                                                                                                                                       |
| ---       | ---                                                                                                                                                          |
| Counter   | Used to count things                                                                                                                                         |
| Gauge     | A particular value at a particular time                                                                                                                      |
| Histogram | Represents a statistical distribution of a series of values.<br> Each histogram are `count`, `average`, `minimum`, `maximum`, `median` and `95th percentile` |
| Set       | Used to count the value of unique in a group                                                                                                                 |

## reference

http://docs.datadoghq.com/guides/metrics/#overview
