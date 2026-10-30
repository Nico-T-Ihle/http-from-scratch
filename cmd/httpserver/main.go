package main

import (
	"context"
	"fmt"
	"html"
	"httpServer/internal/http1"
	"httpServer/internal/mux1"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const addr = ":8282"

func addDigits(num int) int {

	if num < 10 {
		return num
	}
	// some leet code to get a fresh head lol
	convertInt2String := strconv.Itoa(num)
	numLength := len(convertInt2String)
	countSumm := 0

	println(numLength)
	for _, r := range convertInt2String {
		if numLength > 1 {
			println(int(r))
			convertBack, _ := strconv.Atoi(string(r))
			println("convertBack", convertBack)
			countSumm += convertBack
		}
	}
	// 38 3 +8 = 11 count leng > 1 repeat
	if countSumm >= 10 {
		return addDigits(countSumm)
	}
	return countSumm
}

func main() {
	var x = addDigits(38)
	println("FLQM:", x)
	println("the new IMAGE")

	ctx := context.Background()
	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatalf("Tracer Init failed: %v", err)
	}

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Fehler beim Tracer Shutdown: %v", err)
		}
	}()

	router := mux1.NewServerMux()

	tr := otel.Tracer("http-server")

	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("HANDLER HELLO WURDE AUFGERUFEN")

		_, span := tr.Start(r.Context(), "GET /hello")
		defer span.End()

		traceID := span.SpanContext().TraceID().String()
		fmt.Println("TRACE ID ERZEUGT:", traceID)

		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	router.HandleFunc("/hello/", func(w http.ResponseWriter, r *http.Request) {
		_, span := tr.Start(r.Context(), "GET /hello/*")
		defer span.End()

		traceID := span.SpanContext().TraceID().String()
		log.Printf("action=hello_slash trace_id=%s path=%s", traceID, r.URL.Path)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Willkommen im Hello-Ordner! Pfad: " + r.URL.Path + "\n"))
	})

	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	if err := http1.ListenAndServe(addr, router); err != nil {
		panic(err)
	}
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "jaeger.monitoring.svc.cluster.local:4317"
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, fmt.Errorf("exporter error: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("http-server"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("resource error: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
