package main

import (
	"fmt"
	"html"
	"httpServer/internal/http1"
	"httpServer/internal/mux1"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const addr = ":8282"

func addDigits(num int) int {
	// some leet code to get a fresh head lol
	// ... //
	return num
}

type httpRequestsTotal struct {
	Typ  *prometheus.CounterVec
	Name string "http_request_totl"
}

func main() {

	var x = addDigits(38)
	println("FLQM:", x)

	router := mux1.NewServerMux()

	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	router.HandleFunc("/hello/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Willkommen im Hello-Ordner! Pfad: " + r.URL.Path + "\n"))
	})

	// TODO: do i have to create some new endpoints for the victoria metrics?

	// how do i do this...??!
	if err := http1.ListenAndServe(addr, router); err != nil {
		panic(err)
	}
}
