package pipeline

import (
	"errors"
	"time"
)

type VariablePathNode string

const (
	PathNodeRangeItem  VariablePathNode = "$rangeItem"
	PathNodeRangeIndex VariablePathNode = "$rangeIndex"
)

type VariablePath string

var ErrVariableNotFound = errors.New("variable not found")

type Scope struct {
	Finished  bool
	CreatedAt time.Time
	Pipelines Pipelines
	variables map[VariablePath]any
}

func NewScope(pipelines Pipelines) Scope {
	return Scope{
		CreatedAt: time.Now(),
		variables: map[VariablePath]any{},
		Pipelines: pipelines,
	}
}

func (c Scope) WithVariable(path VariablePath, item any) Scope {
	if path == "" {
		return c
	}

	variable := map[VariablePath]any{}
	for k, v := range c.variables {
		variable[k] = v
	}

	variable[path] = item
	c.variables = variable

	return c
}

func (c Scope) WithVariables(items map[VariablePath]any) Scope {
	for path, item := range items {
		c = c.WithVariable(path, item)
	}

	return c
}

func (c Scope) Clone() Scope {
	clone := c
	clone.variables = make(map[VariablePath]any)

	for k, v := range c.variables {
		clone.variables[k] = v
	}

	return clone
}

func (c Scope) Merge(ctx Scope) Scope {
	return c.WithVariables(ctx.variables)
}

func (c Scope) Variable(path VariablePath) (any, error) {
	item, found := c.variables[path]
	if !found {
		return nil, ErrVariableNotFound
	}

	return item, nil
}
