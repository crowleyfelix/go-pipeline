package pipeline

import (
	"encoding/json"
	"errors"
	"text/template"

	"github.com/PaesslerAG/jsonpath"
)

var templateFuncs = template.FuncMap{
	"baggage": func(ctx Context, path BaggagePath) (any, error) {
		result, err := ctx.BaggageItem(path)
		if err != nil {
			return nil, err
		}

		return result, nil
	},
	"baggageDict": func(ctx Context, path BaggagePath, key string) (any, error) {
		result, err := ctx.BaggageItem(path)
		if err != nil {
			return nil, err
		}

		if result == nil {
			return nil, err
		}

		mp, ok := result.(map[string]any)
		if !ok {
			return nil, errors.New("expected a map[string]any")
		}

		return mp[key], nil
	},
	"jsonPath": func(path string, data string) (any, error) {
		var src any
		err := json.Unmarshal([]byte(data), &src)

		if err != nil {
			return nil, err
		}

		return jsonpath.Get(path, src)
	},
}
