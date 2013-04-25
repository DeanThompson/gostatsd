package statsd

/* import (
	"math"
)
*/

// round rounds a number to its nearest integer value
/*
func round(v float64) float64 {
	return math.Floor(v + 0.5)
}
*/

func round(value float64) int {
	if value < 0.0 {
		value -= 0.5
	} else {
		value += 0.5
	}
	return int(value)
}

// average computes the average (mean) of a list of numbers
func average(vals []float64) float64 {
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
