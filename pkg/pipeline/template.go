package pipeline

import (
	"encoding/json"
	"errors"
	"text/template"

	"github.com/PaesslerAG/jsonpath"
)

var templateFuncs = template.FuncMap{
	"variable": func(ctx Scope, path VariablePath) (any, error) {
		result, err := ctx.Variable(path)
		if err != nil {
			return nil, err
		}

		return result, nil
	},
	"variableGet": func(ctx Scope, path VariablePath, key string) (any, error) {
		result, err := ctx.Variable(path)
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
