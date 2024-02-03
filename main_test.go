package main

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	redisServer   *miniredis.Miniredis
	fakeServerURL string
)

// CustomTransport is a custom HTTP transport that embeds http.DefaultTransport and overrides the RoundTrip method.
type CustomTransport struct{}

// RoundTrip implements the RoundTrip method of the http.RoundTripper interface.
func (c *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Creating a fake response
	fakeResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("Fake Backend Response")),
	}
	return fakeResponse, nil
}

func ockRedis() *miniredis.Miniredis {
	s, err := miniredis.Run()

	if err != nil {
		panic(err)
	}

	return s
}

func setup() {
	redisServer = ockRedis()
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisServer.Addr(),
	})

	// Create a fake HTTP server for testing
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Fake Response Body"))
	}))
	fakeServerURL = fakeServer.URL
}

func teardown() {
	// Close the mock Redis server
	redisServer.Close()
}

func TestProxyHandler(t *testing.T) {
	setup()
	defer teardown()

	oldProxyURL := os.Getenv("PROXY_URL")
	defer func() { os.Setenv("PROXY_URL", oldProxyURL) }()
	os.Setenv("PROXY_URL", fakeServerURL)

	originalTransport := http.DefaultTransport
	http.DefaultTransport = &CustomTransport{}
	defer func() { http.DefaultTransport = originalTransport }()

	req, err := http.NewRequest("GET", "/v1/bet_insights/partner-sports", nil)
	req.Header.Add("Partner-Id", "123")
	req.Header.Add("Authorization", "123")
	assert.NoError(t, err)

	recorder := httptest.NewRecorder()

	proxyHandler(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Fake Backend Response")
}

func TestProxyHandlerInvalidHeaders(t *testing.T) {
	setup()
	defer teardown()

	oldProxyURL := os.Getenv("PROXY_URL")
	defer func() { os.Setenv("PROXY_URL", oldProxyURL) }()
	os.Setenv("PROXY_URL", fakeServerURL)

	originalTransport := http.DefaultTransport
	http.DefaultTransport = &CustomTransport{}
	defer func() { http.DefaultTransport = originalTransport }()

	req, err := http.NewRequest("GET", "/v1/bet_insights/partner-sports", nil)
	assert.NoError(t, err)

	recorder := httptest.NewRecorder()

	proxyHandler(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestProxyHandlerNotMapUrl(t *testing.T) {
	setup()
	defer teardown()

	oldProxyURL := os.Getenv("PROXY_URL")
	defer func() { os.Setenv("PROXY_URL", oldProxyURL) }()
	os.Setenv("PROXY_URL", fakeServerURL)

	originalTransport := http.DefaultTransport
	http.DefaultTransport = &CustomTransport{}
	defer func() { http.DefaultTransport = originalTransport }()

	req, err := http.NewRequest("GET", "/some_url", nil)
	req.Header.Add("Partner-Id", "123")
	req.Header.Add("Authorization", "123")
	assert.NoError(t, err)

	recorder := httptest.NewRecorder()

	proxyHandler(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
