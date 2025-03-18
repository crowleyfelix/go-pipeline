package expression

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/crowleyfelix/go-pipeline/pkg/log"
	"gopkg.in/yaml.v3"
)

// Field to be evaluated with context.
type Field[T any] struct {
	any
}

func (f *Field[T]) UnmarshalYAML(node *yaml.Node) error {
	var t T

	err := node.Decode(&t)
	f.any = t

	return err
}

func (f *Field[T]) MarshalYAML() (any, error) {
	return f.any, nil
}

// Eval the values with the context.
func (f Field[T]) Eval(ctx context.Context) (T, error) {
	var value T

	var nodeBuff bytes.Buffer

	err := yaml.NewEncoder(&nodeBuff).Encode(f.any)
	if err != nil {
		return value, err
	}

	log.Log().Debug(ctx, "field template: %s", nodeBuff.String())

	parsed, err := templ.Parse(nodeBuff.String())
	if err != nil {
		return value, err
	}

	nodeBuff.Reset()

	if err = parsed.Execute(&nodeBuff, ctx); err != nil {
		return value, err
	}

	log.Log().Debug(ctx, "field evaluated: %s", nodeBuff.String())

	err = yaml.Unmarshal(nodeBuff.Bytes(), &value)
	if err != nil {
		return value, err
	}

	return value, nil
}

type Bool struct {
	Field[string]
}

func (f Bool) Eval(ctx context.Context) (bool, error) {
	value, err := f.Field.Eval(ctx)

	if err != nil {
		return false, err
	}

	if value == "" {
		return false, nil
	}

	return strconv.ParseBool(value)
}

type Int struct {
	Field[string]
}

func (f Int) Eval(ctx context.Context) (int, error) {
	value, err := f.Field.Eval(ctx)

	if err != nil {
		return 0, err
	}

	if value == "" {
		return 0, nil
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return intValue, nil
}

type Duration struct {
	Field[string]
}

func (f Duration) Eval(ctx context.Context) (time.Duration, error) {
	value, err := f.Field.Eval(ctx)

	if err != nil {
		return 0, err
	}

	if value == "" {
		return 0, nil
	}

	return time.ParseDuration(value)
}

type JSON[T any] struct {
	Field[string]
}

func (f JSON[T]) Eval(ctx context.Context) (T, error) {
	var t T

	value, err := f.Field.Eval(ctx)

	if err != nil {
		return t, err
	}

	err = json.Unmarshal([]byte(value), &t)
	if err != nil {
		return t, err
	}

	return t, nil
}
