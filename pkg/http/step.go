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
	URL    string            `yaml:"url"`
	Method string            `yaml:"method"`
	Body   string            `yaml:"body"`
	Header map[string]string `yaml:"header"`
}

type Response struct {
	*http.Response
	Body string
}

func StepExecutor(client Client) pipeline.StepExecutor {
	return func(ctx context.Context, scope pipeline.Scope, step pipeline.Step) (pipeline.Scope, error) {
		raw, err := pipeline.StepParams[expression.Field[ExecutorParams]](step.Params)
		if err != nil {
			return scope, err
		}

		params, err := raw.Eval(ctx, scope)
		if err != nil {
			return scope, err
		}

		header := http.Header{}
		for k, v := range params.Header {
			header.Add(k, v)
		}

		var body io.Reader
		if params.Body != "" {
			body = strings.NewReader(params.Body)
		}

		req, err := http.NewRequestWithContext(ctx, params.Method, params.URL, body)
		if err != nil {
			return scope, err
		}

		req.Header = header

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
	}
}
