package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"github.com/mroth/weightedrand/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

type ResponseTime string

const (
	Fast   ResponseTime = "fast"
	Medium ResponseTime = "medium"
	Slow   ResponseTime = "slow"
)

// var ChooseRuntime *weightedrand.Chooser[time.Duration, int]
var ChooseRuntime *weightedrand.Chooser[ResponseTime, int]
var ChooseStatus *weightedrand.Chooser[int, int]

func init() {
	log.SetOutput(os.Stderr)
	//log.SetFormatter(&log.JSONFormatter{})
	ChooseRuntime, _ = weightedrand.NewChooser(
		weightedrand.NewChoice(Fast, 8),
		weightedrand.NewChoice(Medium, 1),
		weightedrand.NewChoice(Slow, 1),
	)

	ChooseStatus, _ = weightedrand.NewChooser(
		weightedrand.NewChoice(200, 90),
		weightedrand.NewChoice(400, 7),
		weightedrand.NewChoice(500, 3),
	)

}

// databases
func db(w http.ResponseWriter, r *http.Request) {

	fmt.Println(time.Duration(randFloatExclusiveMinInclusiveMax(0, 0.1)*1000) * time.Millisecond)
	start := time.Now()

	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)

	str := ""
	if r.URL.Query().Get("action") == "encode" {
		str = time.DateTime
	} else {
		str = base64.URLEncoding.EncodeToString([]byte(time.DateTime))
	}

	measure("db", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	res := Output{str}
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}
}

// backends
func encode(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)

	var s Output
	err := requests.
		URL("http://" + GetEnv("DB_HOST", "localhost:8080") + "/db?action=encode").
		CheckStatus(200).
		ToJSON(&s).
		Fetch(context.Background())
	if err != nil {
		var responseError *requests.ResponseError
		switch {
		case errors.As(err, &responseError):
			measure("encode", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, responseError.StatusCode)
			w.WriteHeader(responseError.StatusCode)
			return

		default:
			panic(err)
		}
	}

	measure("encode", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	res := Output{base64.StdEncoding.EncodeToString([]byte(s.Output))}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

}

func decode(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)
	var s Output
	err := requests.
		URL("http://" + GetEnv("DB_HOST", "localhost:8080") + "/db?action=decode").
		CheckStatus(200).
		ToJSON(&s).
		Fetch(context.Background())
	if err != nil {
		var responseError *requests.ResponseError
		switch {
		case errors.As(err, &responseError):
			measure("encode", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, responseError.StatusCode)
			w.WriteHeader(responseError.StatusCode)
			return

		default:
			panic(err)
		}
	}

	measure("decode", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode)
	data, _ := base64.StdEncoding.DecodeString(s.Output)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	res := Output{string(data)}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}
}

// frontend
func conceal(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)

	var s Output
	err := requests.
		URL("http://" + GetEnv("ENCODE_HOST", "localhost:8080") + "/encode").
		CheckStatus(200).
		ToJSON(&s).
		Fetch(context.Background())
	if err != nil {
		var responseError *requests.ResponseError
		switch {
		case errors.As(err, &responseError):
			measure("conceal", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, responseError.StatusCode)
			w.WriteHeader(responseError.StatusCode)
			return

		default:
			panic(err)
		}
	}

	measure("conceal", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	res := Output{s.Output}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}
}

func show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)

	var s Output
	err := requests.
		URL("http://" + GetEnv("DECODE_HOST", "localhost:8080") + "/decode").
		CheckStatus(200).
		ToJSON(&s).
		Fetch(context.Background())
	if err != nil {
		var responseError *requests.ResponseError
		switch {
		case errors.As(err, &responseError):
			measure("show", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, responseError.StatusCode)
			w.WriteHeader(responseError.StatusCode)
			return

		default:
			panic(err)
		}
	}

	measure("show", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	res := Output{s.Output}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

}

func main() {

	reg := prometheus.NewRegistry()
	reg.MustRegister(requestCounter)
	reg.MustRegister(okCounter)
	reg.MustRegister(errorCounter)
	reg.MustRegister(totalTimeSpentHistogram)
	reg.MustRegister(memoryUsageHistogram)
	reg.MustRegister(timeSpentHistogram)
	reg.MustRegister(memoryUsageGauge)
	reg.MustRegister(timeSpentSummary)
	reg.MustRegister(memoryUsageSummary)
	reg.MustRegister(totalTimeSpentSummary)

	http.HandleFunc("/conceal", conceal)
	http.HandleFunc("/show", show)
	http.HandleFunc("/encode", encode)
	http.HandleFunc("/decode", decode)
	http.HandleFunc("/db", db)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
