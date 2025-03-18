package http

import (
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

func RegisterProcessor(client Client) {
	pipeline.RegisterProcessor("http", StepProcessor(client))
}

type ProcessorParams struct {
	URL    string            `yaml:"url"`
	Method string            `yaml:"method"`
	Body   string            `yaml:"body"`
	Header map[string]string `yaml:"header"`
}

type Response struct {
	*http.Response
	Body string
}

func StepProcessor(client Client) pipeline.StepProcessor {
	return func(ctx pipeline.Context, step pipeline.Step) (pipeline.Context, error) {
		raw, err := pipeline.StepParams[expression.Field[ProcessorParams]](step.Params)
		if err != nil {
			return ctx, err
		}

		params, err := raw.Eval(ctx)
		if err != nil {
			return ctx, err
		}

		header := http.Header{}
		for k, v := range params.Header {
			header.Add(k, v)
		}

		var body io.Reader
		if params.Body != "" {
			body = strings.NewReader(params.Body)
		}

		req, err := http.NewRequestWithContext(ctx.Context, params.Method, params.URL, body)
		if err != nil {
			return ctx, err
		}

		req.Header = header

		resp, err := client.Do(req)
		if err != nil {
			return ctx, err
		}

		defer func() {
			err = resp.Body.Close()
			if err != nil {
				log.Log().Error(ctx, "failed to close response body %v", err)
			}
		}()

		blob, err := io.ReadAll(resp.Body)
		if err != nil {
			return ctx, err
		}

		return ctx.WithBaggage(step.BaggagePath(), Response{resp, string(blob)}), nil
	}
}
