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

// String to be evaluated with context.
type String string

// Eval the values with the context.
func (f String) Eval(ctx context.Context, scope any) (string, error) {
	log.Log().Debug(ctx, "field template: %s", f)

	templ, err := templ.Clone()
	if err != nil {
		return "", err
	}

	parsed, err := templ.Parse(string(f))
	if err != nil {
		return "", err
	}

	var nodeBuff bytes.Buffer
	if err = parsed.Execute(&nodeBuff, scope); err != nil {
		return "", err
	}

	log.Log().Debug(ctx, "field evaluated: %s", nodeBuff.String())

	return nodeBuff.String(), nil
}

type Bool String

func (b Bool) Eval(ctx context.Context, scope any) (bool, error) {
	value, err := String(b).Eval(ctx, scope)

	if err != nil {
		return false, err
	}

	if value == "" {
		return false, nil
	}

	return strconv.ParseBool(value)
}

type Int String

func (i Int) Eval(ctx context.Context, scope any) (int, error) {
	value, err := String(i).Eval(ctx, scope)

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

type Duration String

func (d Duration) Eval(ctx context.Context, scope any) (time.Duration, error) {
	value, err := String(d).Eval(ctx, scope)

	if err != nil {
		return 0, err
	}

	if value == "" {
		return 0, nil
	}

	return time.ParseDuration(value)
}

type JSON[T any] String

func (j JSON[T]) Eval(ctx context.Context, scope any) (T, error) {
	var t T

	value, err := String(j).Eval(ctx, scope)

	if err != nil {
		return t, err
	}

	err = json.Unmarshal([]byte(value), &t)
	if err != nil {
		return t, err
	}

	return t, nil
}

type YAML[T any] String

func (f *YAML[T]) UnmarshalYAML(node *yaml.Node) error {
	blob, err := yaml.Marshal(node)
	*f = YAML[T](blob)

	return err
}

func (y YAML[T]) Eval(ctx context.Context, scope any) (T, error) {
	var t T

	value, err := String(y).Eval(ctx, scope)

	if err != nil {
		return t, err
	}

	err = yaml.Unmarshal([]byte(value), &t)
	if err != nil {
		return t, err
	}

	return t, nil
}

type Map map[string]String

func (m Map) Eval(ctx context.Context, scope any) (map[string]string, error) {
	mapped := make(map[string]string)

	for k, v := range m {
		value, err := v.Eval(ctx, scope)
		if err != nil {
			return nil, err
		}

		mapped[k] = value
	}

	return mapped, nil
}
