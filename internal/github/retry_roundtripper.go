package github

import (
	"log"
	"math"
	"net/http"
	"time"
)

type RetryTransport struct {
	next    http.RoundTripper
	Retries int
	log     log.Logger
}

func NewRetryTransport(retry int) *RetryTransport {
	return &RetryTransport{
		next:    http.DefaultTransport,
		Retries: retry,
	}
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	for i := 0; i <= t.Retries; i++ {
		resp, err = t.next.RoundTrip(req)
		if err != nil {
			return resp, err
		}

		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			log.Printf("Retrying request due to status code: %d\n", resp.StatusCode)
			delay := time.Duration(math.Pow(2, float64(i))) * time.Second
			time.Sleep(delay)

			resp.Body.Close()
			continue
		}

		break
	}

	return resp, err
}
