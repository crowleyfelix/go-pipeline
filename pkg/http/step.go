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
	URL    expression.String   `yaml:"url"`
	Method expression.String   `yaml:"method"`
	Body   expression.String   `yaml:"body"`
	Header http.Header         `yaml:"header"`
	Read   bool                `yaml:"read"`
	Set    pipeline.SetParams  `yaml:"set"`
	Stop   pipeline.StopParams `yaml:"stop"`
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
//	  	set:
//	  	  status: '{{ (variable . "http-step").StatusCode }}'
//	  stop:
//	    condition: '{{ ne (variable . "http-step").StatusCode 200 }}'
//	    message: 'unexpected response status'
//	    is_error: true
func StepExecutor(client Client) pipeline.StepExecutor {
	return pipeline.TypedStepExecutor[ExecutorParams](
		func(ctx context.Context, scope pipeline.Scope, step pipeline.Step, p ExecutorParams) (pipeline.Scope, error) {
			url, err := p.URL.Eval(ctx, scope)
			if err != nil {
				return scope, err
			}

			method, err := p.Method.Eval(ctx, scope)
			if err != nil {
				return scope, err
			}

			body, err := p.Body.Eval(ctx, scope)
			if err != nil {
				return scope, err
			}

			var reader io.Reader
			if body != "" {
				reader = strings.NewReader(body)
			}

			req, err := http.NewRequestWithContext(ctx, method, url, reader)
			if err != nil {
				return scope, err
			}

			req.Header = p.Header

			resp, err := client.Do(req)
			if err != nil {
				return scope, err
			}

			variables := map[pipeline.VariablePath]any{
				step.VariablePath():                     resp,
				step.VariablePath(VariablePathNodeBody): resp.Body,
			}

			if p.Read {
				defer func() {
					err = resp.Body.Close()
					if err != nil {
						log.Log().Error(ctx, "failed to close response body %v", err)
					}
				}()

				blob, readErr := io.ReadAll(resp.Body)
				if readErr != nil {
					return scope, readErr
				}

				variables[step.VariablePath(VariablePathNodeBody)] = string(blob)
			}

			scope = scope.WithVariables(variables)

			stop, err := p.Stop.Condition.Eval(ctx, scope)
			if err != nil {
				return scope, err
			}

			if stop {
				return pipeline.StopExecutor(ctx, scope, step, p.Stop)
			}

			if string(p.Set.YAML) != "" {
				scope, err = pipeline.SetExecutor(ctx, scope, step, p.Set)
				if err != nil {
					return scope, err
				}
			}

			return scope, nil
		},
	)
}
