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
	}] `yaml:",inline"`
}

func (p ExecutorParams) toRequest(ctx context.Context, scope pipeline.Scope) (*http.Request, error) {
	params, err := p.Eval(ctx, scope)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if params.Body != "" {
		reader = strings.NewReader(params.Body)
	}

	req, err := http.NewRequestWithContext(ctx, params.Method, params.URL, reader)
	if err != nil {
		return nil, err
	}

	req.Header = params.Header

	return req, err
}

type Response struct {
	*http.Response
	Body string
}

func StepExecutor(client Client) pipeline.StepExecutor {
	return pipeline.TypedStepExecutor[ExecutorParams](
		func(ctx context.Context, scope pipeline.Scope, step pipeline.Step, params ExecutorParams) (pipeline.Scope, error) {
			req, err := params.toRequest(ctx, scope)
			if err != nil {
				return scope, err
			}

			resp, err := client.Do(req)
			if err != nil {
				return scope, err
			}

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

			return scope.WithVariable(step.VariablePath(), Response{resp, string(blob)}), nil
		},
	)
}
