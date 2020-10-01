package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chonlatee/simplehttproundtripper/cache"
)

// - logging
// - retrying
// - auth
// - caching
// - headers manipulation -> not implement it yet
// - testing -> not implement it yet

func main() {

	memCache := cache.NewMemoryCache()

	// c := &http.Client{
	// 	Transport: &authRoundTripper{
	// 		next: &retryRoundTripper{
	// 			next: &loggingRoundTripper{
	// 				next: &cacheRoundTripper{
	// 					cache: memCache,
	// 					next:  http.DefaultTransport,
	// 				},
	// 				logger: os.Stdout,
	// 			},
	// 			maxRetries: 3,
	// 			delay:      time.Second,
	// 		},
	// 		user: "bob",
	// 		pwd:  "pwd",
	// 	},
	// }

	c := &http.Client{
		Transport: &cacheRoundTripper{
			cache: memCache,
			next:  http.DefaultTransport,
		},
	}

	req, err := http.NewRequest(http.MethodGet, "http://httpbin.org/get", nil)
	if err != nil {
		panic(err)
	}

	reqTicker := time.NewTicker(time.Second * 1)

	terminateChannel := make(chan os.Signal, 1)

	signal.Notify(terminateChannel, syscall.SIGTERM, syscall.SIGHUP)
	for {
		select {
		case <-reqTicker.C:
			before := time.Now()
			res, err := c.Do(req)

			if err != nil {
				panic(err)
			}

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				panic(err)
			}

			defer res.Body.Close()

			fmt.Println("--- Response ---")
			fmt.Println("STATUS CODE: ", res.StatusCode)
			fmt.Println("STATUS: ", res.Status)
			fmt.Println("BODY: ", string(body))
			diff := time.Now().Sub(before)

			fmt.Println(diff)
		case <-terminateChannel:
			reqTicker.Stop()
			return
		}
	}

}

type cacheRoundTripper struct {
	next  http.RoundTripper
	cache cache.Cache
}

// simple cache not use in production
func (c cacheRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	fmt.Println("cache roundtrip")

	cacheValue, err := c.cache.Get("resp")
	if err != nil {
		fmt.Println("Read from server")
		resp, err := c.next.RoundTrip(r)

		respBody, err := httputil.DumpResponse(resp, true)

		c.cache.Set("resp", string(respBody))
		return resp, err
	}

	if cacheValue != "" {
		fmt.Println("Read from cache")
		buf := bytes.NewBuffer([]byte(cacheValue))
		return http.ReadResponse(bufio.NewReader(buf), r)
	}

	return c.next.RoundTrip(r)
}

type authRoundTripper struct {
	next http.RoundTripper
	user string
	pwd  string
}

func (a authRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	fmt.Println("basic auth roundtrip")
	r.SetBasicAuth(a.user, a.pwd)
	return a.next.RoundTrip(r)
}

type retryRoundTripper struct {
	next       http.RoundTripper
	maxRetries int
	delay      time.Duration // delay between each retry
}

func (rr retryRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	fmt.Println("retry roundtrip")
	var attempts int
	for {
		resp, err := rr.next.RoundTrip(r)

		// max retries exceeded
		if attempts == rr.maxRetries {
			return resp, err
		}

		// good outcome
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			return resp, err
		}
		attempts++
		fmt.Println("Waiting..")

		select {
		case <-r.Context().Done():
			return resp, r.Context().Err()
		case <-time.After(rr.delay):
		}
	}
}

type loggingRoundTripper struct {
	next   http.RoundTripper
	logger io.Writer
}

// RoundTrip is a decorator on top of the default roundtripper
func (l loggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	fmt.Println("logging roundtrip")
	fmt.Fprintf(l.logger, "[%s] - %s %s\n", time.Now().Format(time.RFC3339), r.Method, r.URL.String())
	return l.next.RoundTrip(r)
}
