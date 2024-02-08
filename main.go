// Package main contains the main entry point and configuration for the back office cache proxy server.
package main

import (
	"back_office_cacher/services"
	"back_office_cacher/utils"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Global variable to hold the Redis client instance.
var redisClient *redis.Client

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
	redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	prometheus.MustRegister(httpRequestsTotal)
}

// proxyHandler handles incoming HTTP requests by either retrieving responses
// from the cache or proxying requests to the backend server.
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Increment Prometheus metric for total HTTP requests.
	httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path).Inc()

	partnerID := r.Header.Get("Partner-Id")
	authorization := r.Header.Get("Authorization")

	// Check if authentication credentials are provided.
	if partnerID == "" || authorization == "" {
		utils.WriteJSONError(w, "Authentication credentials were not provided.", http.StatusUnauthorized)
		return
	}

	// Create a CacheKeyService instance to generate cache keys based on request parameters.
	cacheKeyService := services.NewCacheKeyService(partnerID, authorization, r.URL.Path)
	cacheKey, err := cacheKeyService.GetCacheKey()
	if err != nil {
		utils.WriteJSONError(w, "Resource not found.", http.StatusNotFound)
		return
	}

	// Attempt to retrieve the response from the cache.
	cachedResponse, err := services.RetrieveFromCache(redisClient, cacheKey)

	if err == nil {
		utils.MakeHeaders(w, cachedResponse.Headers, cachedResponse.Status)
		w.Write([]byte(cachedResponse.Body))
		return
	}

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

	// Define the time-to-live (TTL) for caching the response.
	keyTtl := 30 * time.Minute

	// Cache the response only if the status code is not in the 5xx range.
	if resp.StatusCode < 500 {
		err := services.CacheResponse(
			redisClient,
			cacheKey,
			resp,
			string(body),
			keyTtl,
		)

		if err != nil {
			fmt.Println("Failed to cache response in Redis:", err)
		}
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
	port := 8000

	// Start the proxy server and log any errors.
	fmt.Printf("Proxy server listening on :%d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Error starting proxy server:", err)
	}
}
