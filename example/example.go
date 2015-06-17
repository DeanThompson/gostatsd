package main

import (
	"github.com/DeanThompson/gostatsd/statsd"
	"log"
)

func main() {
	f := func(m statsd.Metric) {
		log.Printf("%s", m)
	}
	r := statsd.MetricReceiver{":8125", HanlderFunc(f)}
	r.ListenAndReceive()
}
