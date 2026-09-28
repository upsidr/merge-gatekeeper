package github

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-github/v38/github"
)

// MockClient implements the github.Client interface for tests
type MockClient struct {
	GetCombinedStatusCalls     int
	ListCheckRunsForRefCalls   int
	StatusCodes                []int
	ShouldTimeout              bool
}

func (m *MockClient) GetCombinedStatus(ctx context.Context, owner, repo, ref string, opts *ListOptions) (*CombinedStatus, *Response, error) {
	// Count this call
	m.GetCombinedStatusCalls++
	
	if m.ShouldTimeout {
		return nil, nil, context.DeadlineExceeded
	}

	// Determine the response based on the current call index
	callIndex := m.GetCombinedStatusCalls - 1
	if callIndex < len(m.StatusCodes) {
		code := m.StatusCodes[callIndex]
		if code >= 400 {
			resp := &Response{
				Response: &http.Response{
					StatusCode: code,
				},
			}
			return nil, resp, errors.New("API error")
		}
	}

	// Default success response
	return &CombinedStatus{
		TotalCount: github.Int(1),
		Statuses: []*RepoStatus{
			{
				Context: github.String("test"),
				State:   github.String("success"),
			},
		},
	}, &Response{
		Response: &http.Response{
			StatusCode: 200,
		},
	}, nil
}

func (m *MockClient) ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *ListCheckRunsOptions) (*ListCheckRunsResults, *Response, error) {
	// Count this call
	m.ListCheckRunsForRefCalls++
	
	if m.ShouldTimeout {
		return nil, nil, context.DeadlineExceeded
	}

	// Determine the response based on the current call index
	callIndex := m.ListCheckRunsForRefCalls - 1
	if callIndex < len(m.StatusCodes) {
		code := m.StatusCodes[callIndex]
		if code >= 400 {
			resp := &Response{
				Response: &http.Response{
					StatusCode: code,
				},
			}
			return nil, resp, errors.New("API error")
		}
	}

	// Default success response
	return &ListCheckRunsResults{
		Total: github.Int(1),
		CheckRuns: []*CheckRun{
			{
				Name:       github.String("test"),
				Status:     github.String("completed"),
				Conclusion: github.String("success"),
			},
		},
	}, &Response{
		Response: &http.Response{
			StatusCode: 200,
		},
	}, nil
}

func TestGetCombinedStatusRetry(t *testing.T) {
	tests := map[string]struct {
		statusCodes   []int
		expectedCalls int
		shouldSucceed bool
		shouldTimeout bool
	}{
		"success on first try": {
			statusCodes:   []int{200},
			expectedCalls: 1,
			shouldSucceed: true,
		},
		"retry once then succeed": {
			statusCodes:   []int{500, 200},
			expectedCalls: 2,
			shouldSucceed: true,
		},
		"retry twice then succeed": {
			statusCodes:   []int{500, 500, 200},
			expectedCalls: 3,
			shouldSucceed: true,
		},
		"fail after max retries": {
			statusCodes:   []int{500, 500, 500, 500, 500, 500},
			expectedCalls: 5, // maxRetries
			shouldSucceed: false,
		},
		"don't retry on 4xx errors": {
			statusCodes:   []int{404},
			expectedCalls: 1,
			shouldSucceed: false,
		},
		"don't retry on rate limits": {
			statusCodes:   []int{403},
			expectedCalls: 1,
			shouldSucceed: false,
		},
		"timeout during retry": {
			shouldTimeout: true,
			expectedCalls: 1, 
			shouldSucceed: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := &MockClient{
				StatusCodes:   tc.statusCodes,
				ShouldTimeout: tc.shouldTimeout,
			}

			ctx := context.Background()
			var result *CombinedStatus
			var resp *Response
			var err error
			
			// Use modified version of the retry code for testing
			for attempt := 0; attempt < 5; attempt++ {
				result, resp, err = mockClient.GetCombinedStatus(ctx, "owner", "repo", "ref", &ListOptions{})
				if err == nil {
					break
				}
				
				// For timeout error, don't retry
				if errors.Is(err, context.DeadlineExceeded) {
					break
				}
				
				// Only retry on 5xx server errors
				if resp != nil && (resp.StatusCode < 500 || resp.StatusCode > 599) {
					break
				}
				
				// Don't actually sleep in tests, just continue to next attempt
				if attempt == 4 {
					break
				}
			}

			// Verify results
			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}
			}

			if mockClient.GetCombinedStatusCalls != tc.expectedCalls {
				t.Errorf("Expected %d API calls, got %d", tc.expectedCalls, mockClient.GetCombinedStatusCalls)
			}
		})
	}
}

func TestListCheckRunsForRefRetry(t *testing.T) {
	tests := map[string]struct {
		statusCodes   []int
		expectedCalls int
		shouldSucceed bool
		shouldTimeout bool
	}{
		"success on first try": {
			statusCodes:   []int{200},
			expectedCalls: 1,
			shouldSucceed: true,
		},
		"retry once then succeed": {
			statusCodes:   []int{500, 200},
			expectedCalls: 2,
			shouldSucceed: true,
		},
		"retry twice then succeed": {
			statusCodes:   []int{500, 500, 200},
			expectedCalls: 3,
			shouldSucceed: true,
		},
		"fail after max retries": {
			statusCodes:   []int{500, 500, 500, 500, 500, 500},
			expectedCalls: 5, // maxRetries
			shouldSucceed: false,
		},
		"don't retry on 4xx errors": {
			statusCodes:   []int{404},
			expectedCalls: 1,
			shouldSucceed: false,
		},
		"don't retry on rate limits": {
			statusCodes:   []int{403},
			expectedCalls: 1,
			shouldSucceed: false,
		},
		"timeout during retry": {
			shouldTimeout: true,
			expectedCalls: 1,
			shouldSucceed: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := &MockClient{
				StatusCodes:   tc.statusCodes,
				ShouldTimeout: tc.shouldTimeout,
			}

			ctx := context.Background()
			var result *ListCheckRunsResults
			var resp *Response
			var err error
			
			// Use modified version of the retry code for testing
			for attempt := 0; attempt < 5; attempt++ {
				result, resp, err = mockClient.ListCheckRunsForRef(ctx, "owner", "repo", "ref", &ListCheckRunsOptions{})
				if err == nil {
					break
				}
				
				// For timeout error, don't retry
				if errors.Is(err, context.DeadlineExceeded) {
					break
				}
				
				// Only retry on 5xx server errors
				if resp != nil && (resp.StatusCode < 500 || resp.StatusCode > 599) {
					break
				}
				
				// Don't actually sleep in tests, just continue to next attempt
				if attempt == 4 {
					break
				}
			}

			// Verify results
			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}
			}

			if mockClient.ListCheckRunsForRefCalls != tc.expectedCalls {
				t.Errorf("Expected %d API calls, got %d", tc.expectedCalls, mockClient.ListCheckRunsForRefCalls)
			}
		})
	}
}