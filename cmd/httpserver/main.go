package main

import (
	"fmt"
	"html"
	"httpServer/internal/http1"
	"httpServer/internal/mux1"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	for i, r := range convertInt2String {
		println(i, string(r))
		if numLength > 1 {
			println(int(r))
			convertBack, _ := strconv.Atoi(string(r))
			println("convertBack", convertBack)
			countSumm += convertBack
		}
	}
	println("countSumm", countSumm)
	// 38 3 +8 = 11 count leng > 1 repeat
	if countSumm >= 10 {
		return addDigits(countSumm)
	}
	return countSumm
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
		// custom messages?! TODO
	})

	router.HandleFunc("/hello/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Willkommen im Hello-Ordner! Pfad: " + r.URL.Path + "\n"))
	})

	if err := http1.ListenAndServe(addr, router); err != nil {
		panic(err)
	}
}
