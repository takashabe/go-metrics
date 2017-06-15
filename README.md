# go-metrics

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

## Memo

require doing:

* app -> stats.push(value)
* stats.show()

components:

* collect values
  * via import this package
  * via HTTP
* show metrics
  * format: json
  * get
    * via import this package
    * via HTTP
  * flush regularly
    * console
    * server
