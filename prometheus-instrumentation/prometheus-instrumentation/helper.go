package main

import (
	"math"
	"math/rand"
	"os"
	"time"
)

const MB = 1024 * 1024

type Output struct {
	Output string `json:"output"`
}

func GetEnv(key string, def string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def
	} else {
		return val
	}
}

func randFloatExclusiveMinInclusiveMax(min, max float64) float64 {
	rangeSize := max - min
	randomNumber := min + rand.Float64()*rangeSize
	return math.Round(randomNumber*1000) / 1000
}

func sleepTime(min, max float64) time.Duration {
	return time.Duration(randFloatExclusiveMinInclusiveMax(min, max)*1000) * time.Millisecond
}

func getResponseTime(time ResponseTime) time.Duration {
	switch time {
	case Fast:
		return sleepTime(0, 0.1)
	case Medium:
		return sleepTime(0.1, 0.2)
	case Slow:
		return sleepTime(0.2, 0.350)
	default:
		return sleepTime(1, 2)
	}
}
