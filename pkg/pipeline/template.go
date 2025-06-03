package pipeline

import (
	"encoding/json"
	"errors"
	"io"
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

		switch mapResult := result.(type) {
		case map[string]any:
			return mapResult[key], nil
		case map[string]string:
			return mapResult[key], nil
		}

		return nil, errors.New("expected a map[string]any or map[string]string")
	},
	"jsonPath": func(path string, data string) (any, error) {
		var src any
		err := json.Unmarshal([]byte(data), &src)

		if err != nil {
			return nil, err
		}

		return jsonpath.Get(path, src)
	},
	"isJson": func(data string) (bool, error) {
		var js json.RawMessage
		err := json.Unmarshal([]byte(data), &js)
		if err != nil {
			return false, nil
		}

		return true, nil
	},
	"read": func(reader io.Reader) (string, error) {
		defer func() {
			if r, ok := reader.(io.Closer); ok {
				_ = r.Close()
			}
		}()

		data, err := io.ReadAll(reader)
		if err != nil {
			return "", err
		}

		return string(data), nil
	},
}
