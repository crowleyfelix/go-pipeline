package pipeline

import (
	"context"
	"errors"
	"time"
)

type BaggagePathNode string

const (
	BaggageKeyRangeItem  BaggagePathNode = "$rangeItem"
	BaggageKeyRangeIndex BaggagePathNode = "$rangeIndex"
)

type BaggagePath string

var ErrBaggageItemNotFound = errors.New("baggage item not found")

type Context struct {
	context.Context
	Stop      context.CancelCauseFunc
	CreatedAt time.Time
	Pipelines Pipelines
	baggage   map[BaggagePath]any
}

func NewContext(ctx context.Context, pipelines Pipelines) Context {
	ctx, cancel := context.WithCancelCause(ctx)

	return Context{
		Context:   ctx,
		Stop:      cancel,
		CreatedAt: time.Now(),
		baggage:   map[BaggagePath]any{},
		Pipelines: pipelines,
	}
}

func (c Context) WithBaggage(path BaggagePath, item any) Context {
	if path == "" {
		return c
	}

	baggage := map[BaggagePath]any{}
	for k, v := range c.baggage {
		baggage[k] = v
	}

	baggage[path] = item
	c.baggage = baggage

	return c
}

func (c Context) WithBaggageItems(items map[BaggagePath]any) Context {
	for path, item := range items {
		c = c.WithBaggage(path, item)
	}

	return c
}

func (c Context) Clone() Context {
	clone := c
	clone.baggage = make(map[BaggagePath]any)

	for k, v := range c.baggage {
		clone.baggage[k] = v
	}

	return clone
}

func (c Context) BaggageItem(key BaggagePath) (any, error) {
	item, found := c.baggage[key]
	if !found {
		return nil, ErrBaggageItemNotFound
	}

	return item, nil
}
