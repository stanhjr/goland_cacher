// Package main contains the main entry point and configuration for the back office cache proxy server.
package main

import (
	"back_office_cacher/utils"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"net/http"
	"net/url"
	"os"
)

// Prometheus metric definition for tracking total HTTP requests.
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)
)

// Initialization function to set up the Redis client and register Prometheus metrics.
func init() {
	prometheus.MustRegister(httpRequestsTotal)
}

// proxyHandler handles incoming HTTP requests by either retrieving responses
// from the cache or proxying requests to the backend server.
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Increment Prometheus metric for total HTTP requests.
	httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()

	// Define the URL of the backend server to proxy requests to.
	proxyURL := os.Getenv("PROXY_URL")
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		utils.WriteJSONError(w, "Failed to parse remote URL.", http.StatusInternalServerError)
		return
	}

	// Modify the request to use the backend server's URL.
	r.URL.Host = proxy.Host
	r.URL.Scheme = proxy.Scheme
	r.Host = proxy.Host
	fmt.Println("proxy.Host", proxy.Host)
	fmt.Println("proxyURL", proxyURL)

	// Make a request to the backend server.
	resp, err := http.DefaultTransport.RoundTrip(r)

	// Handle errors during the request to the backend server.
	if err != nil {
		utils.WriteJSONError(w, "Failed to make request to remote server.", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response body from the backend server.
	body, err := io.ReadAll(resp.Body)

	// Handle errors while reading the response body.
	if err != nil {
		utils.WriteJSONError(w, "Failed to read response body.", http.StatusInternalServerError)
		return
	}

	utils.MakeHeaders(w, resp.Header, resp.StatusCode)
	w.Write(body)
}

// main is the entry point of the back_office proxy server.
func main() {
	// Handle Prometheus metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())

	// Handle all other requests using the proxyHandler.
	http.HandleFunc("/", proxyHandler)

	// Define the port to listen on.
	port := 8080

	// Start the proxy server and log any errors.
	fmt.Printf("Proxy server listening on :%d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Error starting proxy server:", err)
	}
}
