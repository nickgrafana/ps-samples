package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"github.com/gorilla/mux"
	"github.com/mroth/weightedrand/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	span := trace.SpanFromContext(r.Context())
	defer span.End()

	fmt.Println(time.Duration(randFloatExclusiveMinInclusiveMax(0, 0.1)*1000) * time.Millisecond)
	start := time.Now()

	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)

	span.AddEvent("computation")
	str := ""
	if r.URL.Query().Get("action") == "encode" {
		str = time.DateTime
	} else {
		str = base64.URLEncoding.EncodeToString([]byte(time.DateTime))
	}
	span.AddEvent("end computation")

	measure("db", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode, span)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	span.SetStatus(codes.Ok, "ok")
	if statusCode > 200 {
		span.SetStatus(codes.Error, "failed downstream")
	}

	res := Output{str}
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}
}

// backends
func encode(w http.ResponseWriter, r *http.Request) {
	//_, span := tracer.Start(r.Context(), "encode")
	span := trace.SpanFromContext(r.Context())
	defer span.End()
	start := time.Now()
	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)

	span.AddEvent("upstream")
	var s Output
	err := requests.
		URL("http://" + GetEnv("DB_HOST", "localhost:8080") + "/db?action=encode").
		Transport(otelhttp.NewTransport(http.DefaultTransport)).
		CheckStatus(200).
		ToJSON(&s).
		Fetch(r.Context())
	span.AddEvent("end upstream")
	if err != nil {
		var responseError *requests.ResponseError
		switch {
		case errors.As(err, &responseError):
			span.SetStatus(codes.Error, "failed upstream")
			span.RecordError(err)

			measure("encode", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, responseError.StatusCode, span)
			w.WriteHeader(responseError.StatusCode)
			return

		default:
			panic(err)
		}
	}

	measure("encode", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode, span)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	span.SetStatus(codes.Ok, "ok")
	if statusCode > 200 {
		span.SetStatus(codes.Error, "failed downstream")
	}
	res := Output{base64.StdEncoding.EncodeToString([]byte(s.Output))}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

}

func decode(w http.ResponseWriter, r *http.Request) {

	span := trace.SpanFromContext(r.Context())

	defer span.End()
	start := time.Now()
	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)
	span.AddEvent("upstream")
	var s Output
	err := requests.
		URL("http://" + GetEnv("DB_HOST", "localhost:8080") + "/db?action=decode").
		Transport(otelhttp.NewTransport(http.DefaultTransport)).
		CheckStatus(200).
		ToJSON(&s).
		Fetch(r.Context())
	span.AddEvent("end upstream")
	if err != nil {
		var responseError *requests.ResponseError
		switch {
		case errors.As(err, &responseError):
			span.SetStatus(codes.Error, "failed upstream")
			span.RecordError(err)

			measure("encode", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, responseError.StatusCode, span)
			w.WriteHeader(responseError.StatusCode)
			return

		default:
			panic(err)
		}
	}

	measure("decode", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode, span)
	data, _ := base64.StdEncoding.DecodeString(s.Output)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	span.SetStatus(codes.Ok, "ok")
	if statusCode > 200 {
		span.SetStatus(codes.Error, "failed downstream")
	}
	res := Output{string(data)}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

}

// frontend
func conceal(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "conceal")
	defer span.End()
	start := time.Now()
	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)

	span.AddEvent("upstream")
	var s Output
	err := requests.
		URL("http://" + GetEnv("ENCODE_HOST", "localhost:8080") + "/encode").
		Transport(otelhttp.NewTransport(http.DefaultTransport)).
		CheckStatus(200).
		ToJSON(&s).
		Fetch(ctx)
	span.AddEvent("end upstream")
	if err != nil {
		var responseError *requests.ResponseError
		switch {
		case errors.As(err, &responseError):
			span.SetStatus(codes.Error, "failed upstream")
			span.RecordError(err)

			measure("conceal", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, responseError.StatusCode, span)
			w.WriteHeader(responseError.StatusCode)
			return

		default:
			panic(err)
		}
	}

	measure("conceal", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode, span)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	span.SetStatus(codes.Ok, "ok")
	if statusCode > 200 {
		span.SetStatus(codes.Error, "failed downstream")
	}
	res := Output{s.Output}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}
}

func show(w http.ResponseWriter, r *http.Request) {

	ctx, span := tracer.Start(r.Context(), "show")
	defer span.End()
	span.AddEvent("start")
	span.SetAttributes(attribute.Bool("isTrue", true), attribute.String("stringAttr", "hi!"))

	start := time.Now()
	sleep := getResponseTime((*ChooseRuntime).Pick())
	statusCode := (*ChooseStatus).Pick()
	time.Sleep(sleep)
	span.AddEvent("upstream")
	//childCtx, span := tracer.Start(ctx, "B")
	//defer span.End()

	var s Output
	err := requests.
		URL("http://" + GetEnv("DECODE_HOST", "localhost:8080") + "/decode").
		CheckStatus(200).
		Transport(otelhttp.NewTransport(http.DefaultTransport)).
		ToJSON(&s).
		Fetch(ctx)
	span.AddEvent("end upstream")
	if err != nil {
		var responseError *requests.ResponseError
		switch {
		case errors.As(err, &responseError):
			span.SetStatus(codes.Error, "failed upstream")
			span.RecordError(err)

			measure("show", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, responseError.StatusCode, span)
			w.WriteHeader(responseError.StatusCode)
			return

		default:
			panic(err)
		}
	}

	measure("show", sleep.Seconds(), time.Since(start).Seconds(), sleep.Seconds()*MB, statusCode, span)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	span.SetStatus(codes.Ok, "ok")
	if statusCode > 200 {
		span.SetStatus(codes.Error, "failed downstream")
	}
	res := Output{s.Output}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}

	span.AddEvent("end")

}

func main() {

	ctx := context.Background()
	telemetry := ConfigureOpentelemetry(ctx, GetEnv("SERVICE", "library_name"))
	defer telemetry()

	reg := prometheus.NewRegistry()
	reg.MustRegister(requestCounter)
	reg.MustRegister(errorCounter)
	reg.MustRegister(totalTimeSpentHistogram)
	reg.MustRegister(memoryUsageHistogram)
	reg.MustRegister(timeSpentHistogram)
	reg.MustRegister(memoryUsageGauge)
	reg.MustRegister(timeSpentSummary)
	reg.MustRegister(memoryUsageSummary)
	reg.MustRegister(totalTimeSpentSummary)

	r := mux.NewRouter()

	r.Handle("/conceal", otelhttp.NewHandler(http.HandlerFunc(conceal), "conceal"))
	r.Handle("/show", otelhttp.NewHandler(http.HandlerFunc(show), "show"))
	r.Handle("/encode", otelhttp.NewHandler(http.HandlerFunc(encode), "encode"))
	r.Handle("/decode", otelhttp.NewHandler(http.HandlerFunc(decode), "decode"))

	r.Handle("/db", otelhttp.NewHandler(http.HandlerFunc(db), "db"))

	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
