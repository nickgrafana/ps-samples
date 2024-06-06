package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/carlmjohnson/requests"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"
)

const maxSleep = 0.35

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stderr)

}

type AutoGenerated struct {
	Time     string  `json:"time"`
	Duration float64 `json:"duration"`
}

func GetEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "localhost:8080"
	} else {
		return val
	}
}
func randFloat(min, max float64) float64 {
	return math.Round((min+rand.Float64()*(max-min))*1000) / 1000
}

var (
	summary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "request_summary_seconds",
		Help: "Time taken to complete a request.",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.99: 0.001,
		},
	},
		[]string{"path"},
	)
	summaryMemory = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "memory_summary_bytes",
		Help: "Memory usage of requests.",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.99: 0.001,
		},
	},
		[]string{"path"},
	)
	gauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "memory_total_bytes",
			Help: "Total memory usage in bytes.",
		},
		[]string{"path"},
	)
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paths_total_count",
			Help: "Path requested counter.",
		},
		[]string{"path"},
	)
	histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "request_seconds",
		Help:    "Time taken to complete a request.",
		Buckets: prometheus.LinearBuckets(0, 0.250, 5),
	},
		[]string{"path"},
	)
	histogramMemory = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "memory_bytes",
		Help:    "Memory usage of requests.",
		Buckets: prometheus.LinearBuckets(0, 250000, 5),
	},
		[]string{"path"},
	)
)

func measure(service string, local, total float64) {

	total = math.Round(total*100) / 100
	memory := math.Round((1000000 * local) / maxSleep)

	log.WithFields(log.Fields{
		"local":  local,
		"total":  total,
		"memory": memory,
	}).Info(service)
	counter.With(prometheus.Labels{"path": service}).Inc()
	histogram.With(prometheus.Labels{"path": service}).Observe(total)
	summary.With(prometheus.Labels{"path": service}).Observe(total)
	summaryMemory.With(prometheus.Labels{"path": service}).Observe(memory)
	histogramMemory.With(prometheus.Labels{"path": service}).Observe(memory)
	gauge.With(prometheus.Labels{"path": service}).Set(memory)

}

// databases
func db(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := time.Duration(randFloat(0, maxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)

	str := ""
	if r.URL.Query().Get("action") == "encode" {
		str = time.DateTime
	} else {
		str = base64.URLEncoding.EncodeToString([]byte(time.DateTime))
	}

	measure("db", sleep.Seconds(), time.Since(start).Seconds())

	_, err := w.Write([]byte(str))
	if err != nil {
		panic(err)
	}
}

// backends
func encode(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := time.Duration(randFloat(0, maxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)

	var str string
	err := requests.
		URL("http://" + GetEnv("DB_HOST") + "/db?action=encode").
		ToString(&str).
		Fetch(context.Background())
	if err != nil {
		panic(err)
	}

	measure("encode", sleep.Seconds(), time.Since(start).Seconds())

	_, err = w.Write([]byte(base64.StdEncoding.EncodeToString([]byte(str))))
	if err != nil {
		panic(err)
	}

}

func decode(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := time.Duration(randFloat(0, maxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)
	var s string
	err := requests.
		URL("http://" + GetEnv("DB_HOST") + "/db?action=decode").
		ToString(&s).
		Fetch(context.Background())
	if err != nil {
		panic(err)
	}

	measure("decode", sleep.Seconds(), time.Since(start).Seconds())
	data, _ := base64.StdEncoding.DecodeString(s)
	_, err = w.Write(data)
	if err != nil {
		panic(err)
	}
}

// frontend
func conceal(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := time.Duration(randFloat(0, maxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)

	var s string
	err := requests.
		URL("http://" + GetEnv("ENCODE_HOST") + "/encode").
		ToString(&s).
		Fetch(context.Background())
	if err != nil {
		panic(err)
	}

	measure("conceal", sleep.Seconds(), time.Since(start).Seconds())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res := AutoGenerated{s, time.Since(start).Seconds()}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}
}

func show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := time.Duration(randFloat(0, maxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)

	var s string
	err := requests.
		URL("http://" + GetEnv("DECODE_HOST") + "/decode").
		ToString(&s).
		Fetch(context.Background())
	if err != nil {
		panic(err)
	}

	measure("show", sleep.Seconds(), time.Since(start).Seconds())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res := AutoGenerated{s, time.Since(start).Seconds()}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

}

func main() {

	reg := prometheus.NewRegistry()
	reg.MustRegister(counter)
	reg.MustRegister(histogram)
	reg.MustRegister(histogramMemory)
	reg.MustRegister(gauge)
	reg.MustRegister(summary)
	reg.MustRegister(summaryMemory)

	http.HandleFunc("/conceal", conceal)
	http.HandleFunc("/show", show)
	http.HandleFunc("/encode", encode)
	http.HandleFunc("/decode", decode)
	http.HandleFunc("/db", db)
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
