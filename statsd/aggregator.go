package statsd

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// metricAggregatorStats is a bookkeeping structure for statistics about a MetricAggregator
type metricAggregatorStats struct {
	BadLines       int
	LastMessage    time.Time
	LastFlush      time.Time
	LastFlushError time.Time
}

// MetricSender is an interface that can be implemented by objects which
// can provide metrics to a MetricAggregator
type MetricSender interface {
	SendMetrics(MetricMap) error
}

// MetricAggregator is an object that aggregates statsd metrics.
// The function NewMetricAggregator should be used to create the objects.
//
// Incoming metrics should be sent to the MetricChan channel.
type MetricAggregator struct {
	sync.Mutex
	MetricChan     chan Metric   // Channel on which metrics are received
	FlushInterval  time.Duration // How often to flush metrics to the sender
	Sender         MetricSender  // The sender to which metrics are flushed
	Stats          metricAggregatorStats
	Counters       MetricMap
	Gauges         MetricMap
	Timers         MetricListMap
	TimersCounters MetricMap
}

// NewMetricAggregator creates a new MetricAggregator object
func NewMetricAggregator(sender MetricSender, flushInterval time.Duration) MetricAggregator {
	a := MetricAggregator{}
	a.FlushInterval = flushInterval
	a.Sender = sender
	a.MetricChan = make(chan Metric)
	a.Counters = make(MetricMap)
	a.Gauges = make(MetricMap)
	a.Timers = make(MetricListMap)
	a.TimersCounters = make(MetricMap)
	return a
}

// flush prepares the contents of a MetricAggregator for sending via the Sender
func (a *MetricAggregator) flush() (metrics MetricMap) {
	defer a.Unlock()
	a.Lock()

	metrics = make(MetricMap)
	numStats := 0

	for k, v := range a.Counters {
		perSecond := v / a.FlushInterval.Seconds()
		metrics["stats.counters.rate."+k] = perSecond
		metrics["stats.counters.count."+k] = v
		numStats += 1
	}

	for k, v := range a.Gauges {
		metrics["stats.gauges."+k] = v
		numStats += 1
	}

	// TODO: add histogram
	pctThreshold := []int{95}
	timerData := make(map[string]map[string]float64, 10)
	for k, v := range a.Timers {
		if count := len(v); count > 0 {
			sort.Float64s(v)
			min := v[0]
			max := v[count-1]

			currTimerData := make(map[string]float64, 10)
			var sum, mean float64
			cumulativeValues := make([]float64, count)
			thresholdBoundary := max

			// 计算每个点的累计求和
			cumulativeValues[0] = v[0]
			for i := 1; i < count; i++ {
				cumulativeValues[i] = cumulativeValues[i-1] + v[i]
			}

			for _, pct := range pctThreshold {
				if count > 1 {
					numInThreshold := round(math.Abs(float64(pct)) * float64(count) / 100.0)
					if numInThreshold == 0 {
						continue
					}
					if pct > 0 {
						thresholdBoundary = v[numInThreshold-1]
						sum = cumulativeValues[numInThreshold-1]
					} else {
						thresholdBoundary = v[count-numInThreshold]
						sum = cumulativeValues[count-1] - cumulativeValues[count-numInThreshold]
					}
					mean = sum / float64(numInThreshold)
					cleanPct := strings.Replace(fmt.Sprintf("%d", pct), "-", "top", -1)
					var uplowPrefix string
					if pct > 0 {
						uplowPrefix = "upper_"
					} else {
						uplowPrefix = "lower_"
					}
					currTimerData["mean_"+cleanPct] = mean
					currTimerData[uplowPrefix+cleanPct] = thresholdBoundary
					currTimerData["sum_"+cleanPct] = sum
				}
			}

			sum = cumulativeValues[count-1]
			mean = sum / float64(count)
			sumOfDiffs := 0.0
			median := 0.0
			for i := 0; i < count; i++ {
				sumOfDiffs += (v[i] - mean) * (v[i] - mean)
			}

			mid := int64(math.Floor(float64(count) / 2.0))
			if count%2 == 1 {
				median = v[mid]
			} else {
				median = (v[mid-1] + v[mid]) / 2
			}
			stddev := math.Sqrt(sumOfDiffs / float64(count))
			currTimerData["std"] = stddev
			currTimerData["count_ps"] = a.TimersCounters[k] / a.FlushInterval.Seconds()
			currTimerData["sum"] = sum
			currTimerData["mean"] = mean
			currTimerData["median"] = median
			currTimerData["lower"] = min
			currTimerData["upper"] = max
			currTimerData["count"] = float64(count)

			numStats += 1
			timerData[k] = currTimerData
		}
		for k, v := range timerData {
			for k2, v2 := range v {
				metrics["stats.timers."+k+"."+k2] = v2
			}
		}
	}
	metrics["statsd.numStats"] = float64(numStats)
	// log.Println(metrics)
	return metrics
}

// Reset clears the contents of a MetricAggregator
func (a *MetricAggregator) Reset() {
	defer a.Unlock()
	a.Lock()

	for k := range a.Counters {
		a.Counters[k] = 0
	}

	for k := range a.Timers {
		a.Timers[k] = []float64{}
		a.TimersCounters[k] = 0
	}

	// No reset for gauges, they keep the last value
}

// receiveMetric is called for each incoming metric on MetricChan
func (a *MetricAggregator) receiveMetric(m Metric) {
	defer a.Unlock()
	a.Lock()

	switch m.Type {
	case COUNTER:
		v, ok := a.Counters[m.Bucket]
		value := m.Value
		if m.SampleRate < 1.0 {
			value = m.Value * (1 / m.SampleRate)
		}
		if ok {
			a.Counters[m.Bucket] = v + value
		} else {
			a.Counters[m.Bucket] = value
		}
	case GAUGE:
		a.Gauges[m.Bucket] = m.Value
	case TIMER:
		v, ok := a.Timers[m.Bucket]
		counterValue := 1.0
		if m.SampleRate < 1.0 {
			counterValue = 1.0 / m.SampleRate
		}
		if ok {
			v = append(v, m.Value)
			a.Timers[m.Bucket] = v
			a.TimersCounters[m.Bucket] += counterValue
		} else {
			a.Timers[m.Bucket] = []float64{m.Value}
			a.TimersCounters[m.Bucket] = counterValue
		}
	case ERROR:
		a.Stats.BadLines += 1
	}
	a.Stats.LastMessage = time.Now()
}

// Aggregate starts the MetricAggregator so it begins consuming metrics from MetricChan
// and flushing them periodically via its Sender
func (a *MetricAggregator) Aggregate() {
	flushChan := make(chan error)
	flushTimer := time.NewTimer(a.FlushInterval)

	for {
		select {
		case metric := <-a.MetricChan: // Incoming metrics
			a.receiveMetric(metric)
		case <-flushTimer.C: // Time to flush to graphite
			flushed := a.flush()
			go func() {
				flushChan <- a.Sender.SendMetrics(flushed)
			}()
			a.Reset()
			flushTimer = time.NewTimer(a.FlushInterval)
		case flushResult := <-flushChan:
			a.Lock()

			if flushResult != nil {
				log.Printf("Sending metrics to Graphite failed: %s", flushResult)
				a.Stats.LastFlushError = time.Now()
			} else {
				a.Stats.LastFlush = time.Now()
			}
			a.Unlock()
		}
	}

}
