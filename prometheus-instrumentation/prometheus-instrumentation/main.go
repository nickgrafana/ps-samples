package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/carlmjohnson/requests"
	"github.com/mroth/weightedrand/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

var MaxSleep, _ = strconv.ParseFloat(GetEnv("MAX_SLEEP", "0.35"), 64)
var MaxError, _ = strconv.ParseFloat(GetEnv("MAX_ERR", "0.065"), 64)

func init() {

	log.SetOutput(os.Stderr)

}

type Output struct {
	Time     string  `json:"time"`
	Duration float64 `json:"duration"`
}

func GetEnv(key string, def string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def
	} else {
		return val
	}
}
func randFloat(min, max float64) float64 {
	return math.Round((min+rand.Float64()*(max-min))*1000) / 1000
}

var (
	timeSpentSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "time_spent_summary_seconds",
		Help: "Time taken to complete a request.",
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
		Help: "Total time spent waiting to complete request.",
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
	errorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paths_total_error_count",
			Help: "Path error requested counter.",
		},
		[]string{"path"},
	)
	okCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paths_total_ok_count",
			Help: "Path ok requested counter.",
		},
		[]string{"path"},
	)
	histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "request_seconds",
		Help:    "Time taken to complete a request.",
		Buckets: []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1},
	},
		[]string{"path"},
	)
	histogramLocal = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "request_local_seconds",
		Help:    "Time taken to complete a request.",
		Buckets: []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1},
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

func measure(service string, timeSpent, totalTimeSpent float64) {

	totalTimeSpent = math.Round(totalTimeSpent*100) / 100
	memory := math.Round((1000000 * timeSpent) / MaxSleep)

	if MaxError >= rand.Float64() {
		errorCounter.With(prometheus.Labels{"path": service}).Inc()
		fmt.Println(fmt.Sprintf("%s - ERROR %s: time_spent: %f, total_time_spent: %f, memory: %f", time.Now().Format(time.RFC3339), service, timeSpent, totalTimeSpent, memory))
		log.SetFormatter(&log.JSONFormatter{})
		log.WithFields(log.Fields{
			"time_spent":       timeSpent,
			"total_time_spent": totalTimeSpent,
			"memory":           memory,
		}).Error(service)

		log.SetFormatter(&log.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
		log.WithFields(log.Fields{
			"time_spent":       timeSpent,
			"total_time_spent": totalTimeSpent,
			"memory":           memory,
		}).Error(service)
	} else {
		okCounter.With(prometheus.Labels{"path": service}).Inc()
		fmt.Println(fmt.Sprintf("%s - OK %s: time_spent: %f, total_time_spent: %f, memory: %f", time.Now().Format(time.RFC3339), service, timeSpent, totalTimeSpent, memory))
		log.SetFormatter(&log.JSONFormatter{})
		log.WithFields(log.Fields{
			"time_spent":       timeSpent,
			"total_time_spent": totalTimeSpent,
			"memory":           memory,
		}).Info(service)
		log.SetFormatter(&log.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
		log.WithFields(log.Fields{
			"time_spent":       timeSpent,
			"total_time_spent": totalTimeSpent,
			"memory":           memory,
		}).Info(service)
	}

	counter.With(prometheus.Labels{"path": service}).Inc()     //request number
	gauge.With(prometheus.Labels{"path": service}).Set(memory) //memory over time

	timeSpentSummary.With(prometheus.Labels{"path": service}).Observe(totalTimeSpent)
	summaryMemory.With(prometheus.Labels{"path": service}).Observe(memory)
	totalTimeSpentSummary.With(prometheus.Labels{"path": service}).Observe(timeSpent)

	histogram.With(prometheus.Labels{"path": service}).Observe(totalTimeSpent)
	histogramMemory.With(prometheus.Labels{"path": service}).Observe(memory)
	histogramLocal.With(prometheus.Labels{"path": service}).Observe(timeSpent)

}

func measure_nolog(service string, timeSpent, totalTimeSpent, memory float64) {

	counter.With(prometheus.Labels{"path": service}).Inc()     //request number
	gauge.With(prometheus.Labels{"path": service}).Set(memory) //memory over time

	timeSpentSummary.With(prometheus.Labels{"path": service}).Observe(totalTimeSpent)
	summaryMemory.With(prometheus.Labels{"path": service}).Observe(memory)
	totalTimeSpentSummary.With(prometheus.Labels{"path": service}).Observe(timeSpent)

	histogram.With(prometheus.Labels{"path": service}).Observe(totalTimeSpent)
	histogramMemory.With(prometheus.Labels{"path": service}).Observe(memory)
	histogramLocal.With(prometheus.Labels{"path": service}).Observe(timeSpent)

}

// databases
func db(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := time.Duration(randFloat(0, MaxSleep)*1000) * time.Millisecond
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
	sleep := time.Duration(randFloat(0, MaxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)

	var str string
	err := requests.
		URL("http://" + GetEnv("DB_HOST", "localhost:8080") + "/db?action=encode").
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
	sleep := time.Duration(randFloat(0, MaxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)
	var s string
	err := requests.
		URL("http://" + GetEnv("DB_HOST", "localhost:8080") + "/db?action=decode").
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
	sleep := time.Duration(randFloat(0, MaxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)

	var s string
	err := requests.
		URL("http://" + GetEnv("ENCODE_HOST", "localhost:8080") + "/encode").
		ToString(&s).
		Fetch(context.Background())
	if err != nil {
		panic(err)
	}

	measure("conceal", sleep.Seconds(), time.Since(start).Seconds())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res := Output{s, time.Since(start).Seconds()}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}
}

func show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := time.Duration(randFloat(0, MaxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)

	var s string
	err := requests.
		URL("http://" + GetEnv("DECODE_HOST", "localhost:8080") + "/decode").
		ToString(&s).
		Fetch(context.Background())
	if err != nil {
		panic(err)
	}

	measure("show", sleep.Seconds(), time.Since(start).Seconds())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res := Output{s, time.Since(start).Seconds()}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

}

func logs(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := time.Duration(randFloat(0, MaxSleep)*1000) * time.Millisecond
	time.Sleep(sleep)

	log.SetFormatter(&log.JSONFormatter{})
	m := make([]string, 0)
	m = append(m,
		"POST",
		"GET",
		"PATCH",
		"PUT")

	h := make([]string, 0)
	h = append(h,
		"node1",
		"node2",
		"node3",
		"node4",
		"node5",
		"node6")

	c := make([]string, 0)
	c = append(c,
		"200",
		"201",
		"400",
		"401",
		"403",
		"500",
		"502")

	route := make([]string, 0)
	route = append(route,
		"/products",
		"/users",
		"/cart",
		"/checkout",
		"/login",
		"/logout",
		"/search",
		"/view")

	log.SetFormatter(&log.JSONFormatter{})

	log.SetLevel(log.TraceLevel)

	x := log.WithFields(log.Fields{
		"service_time":    strconv.FormatInt(time.Now().UTC().UnixNano(), 10),
		"server_pid":      rand.Int31n(1000),
		"server_hostname": h[rand.Intn(len(h))],
		"http_method":     m[rand.Intn(len(m))],
		"http_route":      route[rand.Intn(len(route))],
		"http_code":       c[rand.Intn(len(c))],
	})

	chooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice(log.TraceLevel, 1),
		weightedrand.NewChoice(log.DebugLevel, 2),
		weightedrand.NewChoice(log.InfoLevel, 3),
		weightedrand.NewChoice(log.WarnLevel, 2),
		weightedrand.NewChoice(log.ErrorLevel, 2),
	)

	choice := chooser.Pick()

	switch choice {
	case log.TraceLevel:
		x.Trace()
	case log.DebugLevel:
		x.Debug()
	case log.InfoLevel:
		x.Info()
	case log.WarnLevel:
		x.Warn()
	case log.ErrorLevel:
		x.Error()
	default:
		x.Panic()
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res := Output{strconv.FormatInt(time.Now().UTC().UnixNano(), 10), time.Since(start).Seconds()}
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

}

func staticLog(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	minMemory := 1
	maxMemory := 128000

	minTime := 0.01
	maxTime := 1.05
	memory := rand.Intn(maxMemory-minMemory+1) + minMemory
	totalTime := minTime + rand.Float64()*(maxTime-minTime)
	localTime := minTime + rand.Float64()*(maxTime-minTime)
	measure_nolog("static_log", localTime, totalTime, float64(memory))

	log.SetFormatter(&log.JSONFormatter{})

	log.SetLevel(log.InfoLevel)
	h := make([]string, 0)
	h = append(h,
		"node1",
		"node2",
		"node3",
		"node4",
		"node5",
		"node6")

	chooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice("200", 8),
		weightedrand.NewChoice("201", 2),
	)
	x := log.WithFields(log.Fields{
		"service_time":    strconv.FormatInt(time.Now().UTC().UnixNano(), 10),
		"server_pid":      rand.Int31n(1000),
		"server_hostname": h[rand.Intn(len(h))],
		"http_method":     "GET",
		"http_route":      "/view",
		"http_code":       chooser.Pick(),
		"response_time":   totalTime,
		"memory_usage":    memory,
	})

	x.Info()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res := Output{strconv.FormatInt(time.Now().UTC().UnixNano(), 10), time.Since(start).Seconds()}
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

}
func main() {

	reg := prometheus.NewRegistry()
	reg.MustRegister(counter)
	reg.MustRegister(okCounter)
	reg.MustRegister(errorCounter)
	reg.MustRegister(histogram)
	reg.MustRegister(histogramMemory)
	reg.MustRegister(histogramLocal)
	reg.MustRegister(gauge)
	reg.MustRegister(timeSpentSummary)
	reg.MustRegister(summaryMemory)
	reg.MustRegister(totalTimeSpentSummary)

	http.HandleFunc("/conceal", conceal)
	http.HandleFunc("/show", show)
	http.HandleFunc("/encode", encode)
	http.HandleFunc("/decode", decode)
	http.HandleFunc("/db", db)
	http.HandleFunc("/logs", logs)
	http.HandleFunc("/static_log", staticLog)
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
