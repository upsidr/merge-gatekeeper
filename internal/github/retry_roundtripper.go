package github

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

type RetryTransport struct {
	Transport http.RoundTripper
	Retries   int
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	for i := 0; i <= t.Retries; i++ {
		resp, err = t.Transport.RoundTrip(req)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			fmt.Printf("Retrying request due to status code: %d\n", resp.StatusCode)
			delay := time.Duration(math.Pow(2, float64(i))) * time.Second
			time.Sleep(delay)

			resp.Body.Close()
			continue
		}

		break
	}

	return resp, err
}
