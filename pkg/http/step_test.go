package http

import (
	"context"
	"io"
	nethttp "net/http"
	"strings"
	"testing"

	"github.com/crowleyfelix/go-pipeline/pkg/pipeline"
)

type mockClient struct {
	response *nethttp.Response
	err      error
}

func (m mockClient) Do(*nethttp.Request) (*nethttp.Response, error) {
	return m.response, m.err
}

func TestStepExecutor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseCode   int
		responseBody   string
		stopMessage    string
		stopIsError    bool
		expectError    bool
		expectFinished bool
		expectBody     string
	}{
		{
			name:           "stops with error when stop is truthy and is_error is true",
			responseCode:   500,
			responseBody:   `{"error":"boom"}`,
			stopMessage:    "unexpected status",
			stopIsError:    true,
			expectError:    true,
			expectFinished: true,
		},
		{
			name:           "stops without error when stop is truthy and is_error is false",
			responseCode:   500,
			responseBody:   `{"error":"boom"}`,
			stopMessage:    "stop without error",
			stopIsError:    false,
			expectError:    false,
			expectFinished: true,
		},
		{
			name:           "keeps running when stop is falsy",
			responseCode:   200,
			responseBody:   `{"ok":true}`,
			stopMessage:    "should not stop",
			stopIsError:    true,
			expectError:    false,
			expectFinished: false,
			expectBody:     `{"ok":true}`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			executor := StepExecutor(mockClient{
				response: &nethttp.Response{
					StatusCode: tc.responseCode,
					Body:       io.NopCloser(strings.NewReader(tc.responseBody)),
					Header:     nethttp.Header{},
				},
			})

			step := pipeline.Step{
				ID:   "http",
				Type: "http",
				Params: map[string]any{
					"url":    "https://example.com",
					"method": "GET",
					"read":   true,
					"stop": map[string]any{
						"condition": `{{ eq (variable . "http").StatusCode 500 }}`,
						"message":   tc.stopMessage,
						"is_error":  tc.stopIsError,
					},
				},
			}

			scope := pipeline.NewScope(pipeline.Pipelines{})
			result, err := executor.Execute(context.Background(), scope, step)

			if tc.expectError && err == nil {
				t.Fatal("expected stop to return an error")
			}

			if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Finished != tc.expectFinished {
				t.Fatalf("unexpected finished flag: got %v want %v", result.Finished, tc.expectFinished)
			}

			if tc.expectBody == "" {
				return
			}

			body, err := result.Variable("http.$body")
			if err != nil {
				t.Fatalf("expected response body in scope: %v", err)
			}

			bodyText, ok := body.(string)
			if !ok || bodyText != tc.expectBody {
				t.Fatalf("unexpected body value: %#v", body)
			}
		})
	}
}
