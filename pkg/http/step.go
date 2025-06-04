package http

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/crowleyfelix/go-pipeline/pkg/expression"
	"github.com/crowleyfelix/go-pipeline/pkg/log"
	"github.com/crowleyfelix/go-pipeline/pkg/pipeline"
)

const (
	VariablePathNodeBody pipeline.VariablePathNode = "$body"
)

type Client interface {
	Do(*http.Request) (*http.Response, error)
}

func RegisterStepExecutor(client Client) {
	pipeline.RegisterStepExecutor("http", StepExecutor(client))
}

type ExecutorParams struct {
	expression.YAML[struct {
		URL    string      `yaml:"url"`
		Method string      `yaml:"method"`
		Body   string      `yaml:"body"`
		Header http.Header `yaml:"header"`
		Read   bool        `yaml:"read"`
	}] `yaml:",inline"`
}

// StepExecutor executes an HTTP request based on the provided parameters.
// It supports setting the HTTP method, URL, headers, and body.
// If the `read` parameter is true, the response body is read and stored in the pipeline scope.
//
// Example YAML:
//
//	id: http-example
//	steps:
//	- id: http-step
//	  type: http
//	  params:
//	  	url: 'https://api.example.com/data'
//	  	method: 'GET'
//	  	header:
//	  	  Authorization: ['Bearer some-token']
//	  	  Content-Type: ['application/json']
//	  	read: true
func StepExecutor(client Client) pipeline.StepExecutor {
	return pipeline.TypedStepExecutor[ExecutorParams](
		func(ctx context.Context, scope pipeline.Scope, step pipeline.Step, p ExecutorParams) (pipeline.Scope, error) {
			params, err := p.Eval(ctx, scope)
			if err != nil {
				return scope, err
			}

			var reader io.Reader
			if params.Body != "" {
				reader = strings.NewReader(params.Body)
			}

			req, err := http.NewRequestWithContext(ctx, params.Method, params.URL, reader)
			if err != nil {
				return scope, err
			}

			req.Header = params.Header

			resp, err := client.Do(req)
			if err != nil {
				return scope, err
			}

			variables := map[pipeline.VariablePath]any{
				step.VariablePath():                     resp,
				step.VariablePath(VariablePathNodeBody): resp.Body,
			}

			if params.Read {
				defer func() {
					err = resp.Body.Close()
					if err != nil {
						log.Log().Error(ctx, "failed to close response body %v", err)
					}
				}()

				blob, err := io.ReadAll(resp.Body)
				if err != nil {
					return scope, err
				}

				variables[step.VariablePath(VariablePathNodeBody)] = string(blob)
			}

			return scope.WithVariables(variables), nil
		},
	)
}
