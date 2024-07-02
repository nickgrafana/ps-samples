package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

var (
	timeSpentSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "time_spent_summary_seconds",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.99: 0.001,
		},
	},
		[]string{"path"},
	)
	totalTimeSpentSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "total_time_spent_summary_seconds",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.99: 0.001,
		},
	},
		[]string{"path"},
	)
	timeSpentHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "time_spent_histogram_seconds",
		Buckets: []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1},
	},
		[]string{"path"},
	)
	totalTimeSpentHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "total_time_spent_histogram_seconds",
		Buckets: []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1},
	},
		[]string{"path"},
	)
	memoryUsageSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "memory_usage_summary_bytes",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.99: 0.001,
		},
	},
		[]string{"path"},
	)
	memoryUsageGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "memory_usage_gauge_bytes",
		},
		[]string{"path"},
	)
	memoryUsageHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "memory_usage_histogram_bytes",
		Buckets: prometheus.LinearBuckets(0, 250000, 5),
	},
		[]string{"path"},
	)
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_count",
		},
		[]string{"path"},
	)
	errorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_error_count",
		},
		[]string{"path"},
	)
	okCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_ok_count",
		},
		[]string{"path"},
	)
)

func measure(service string, timeSpent, totalTimeSpent, memory float64, status int) {

	if status == 200 {
		okCounter.With(prometheus.Labels{"path": service}).Inc()
	} else {
		errorCounter.With(prometheus.Labels{"path": service}).Inc()
	}
	requestCounter.With(prometheus.Labels{"path": service}).Inc()
	memoryUsageGauge.With(prometheus.Labels{"path": service}).Set(memory)

	timeSpentSummary.With(prometheus.Labels{"path": service}).Observe(totalTimeSpent)
	memoryUsageSummary.With(prometheus.Labels{"path": service}).Observe(memory)
	totalTimeSpentSummary.With(prometheus.Labels{"path": service}).Observe(timeSpent)

	totalTimeSpentHistogram.With(prometheus.Labels{"path": service}).Observe(totalTimeSpent)
	memoryUsageHistogram.With(prometheus.Labels{"path": service}).Observe(memory)
	timeSpentHistogram.With(prometheus.Labels{"path": service}).Observe(timeSpent)

	fields := log.Fields{
		"service_time":        strconv.FormatInt(time.Now().UTC().UnixNano(), 10),
		"server_pid":          rand.Int31n(1000),
		"http_method":         "GET",
		"http_route":          fmt.Sprintf("/%s", service),
		"memory_bytes":        int64(memory),
		"response_time":       fmt.Sprintf("%.3f", timeSpent),
		"response_time_total": fmt.Sprintf("%.3f", totalTimeSpent),
		"http_code":           status,
	}

	switch status {

	case 200:
		x := log.WithFields(fields)
		x.Info()
	case 400:
		x := log.WithFields(fields)
		x.Warn(service)
	case 500:
		x := log.WithFields(fields)
		x.Error(service)
	default:
		fields["http_code"] = 503
		x := log.WithFields(fields)
		x.Panic()
	}

}
