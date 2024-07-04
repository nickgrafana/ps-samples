package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {

	o := otlptracehttp.WithEndpointURL(GetEnv("TEMPO_HOST", "http://localhost:4318/v1/traces"))

	return otlptracehttp.New(ctx, o)

}

func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.

	//r, err := resource.Merge(
	//	resource.Default(),
	//	resource.NewWithAttributes(
	//		semconv.SchemaURL,
	//		semconv.ServiceName(GetEnv("SERVICE", "service_name")),
	//	),
	//)

	r, err := resource.New(
		context.Background(),
		resource.WithFromEnv(),      // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithTelemetrySDK(), // Discover and provide information about the OpenTelemetry SDK used.
		resource.WithProcess(),      // Discover and provide process information.
		resource.WithOS(),           // Discover and provide OS information.
		resource.WithContainer(),    // Discover and provide container information.
		resource.WithHost(),         // Discover and provide host information.
		resource.WithAttributes(attribute.String("x", "y"),
			attribute.String("SchemaURL", semconv.SchemaURL),
			semconv.ServiceName(GetEnv("SERVICE", "service_name"))), // Add custom resource attributes.
		// resource.WithDetectors(thirdparty.Detector{}), // Bring your own external Detector implementation.
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

func ConfigureOpentelemetry(ctx context.Context, name string) func() {
	exp, err := newExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp := newTraceProvider(exp)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{}))

	// Finally, set the tracer that can be used for this package.
	tracer = tp.Tracer(name)

	return func() {

		_ = tp.Shutdown(ctx)
	}
}
