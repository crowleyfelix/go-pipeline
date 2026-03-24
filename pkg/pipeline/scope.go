package pipeline

import (
	"errors"
	"strings"
	"time"
)

type VariablePathNode string

const (
	PathNodeIndex VariablePathNode = "$index"
)

type VariablePath string

var ErrVariableNotFound = errors.New("variable not found")

type Scope struct {
	Finished  bool
	CreatedAt time.Time
	Pipelines Pipelines
	variables map[VariablePath]any
	namespace []VariablePathNode
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

	path = c.qualifyPath(path)

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

	clone.namespace = append([]VariablePathNode{}, c.namespace...)

	for k, v := range c.variables {
		clone.variables[k] = v
	}

	return clone
}

func (c Scope) Merge(ctx Scope) Scope {
	merged := c.Clone()

	for path, item := range ctx.variables {
		merged.variables[path] = item
	}

	return merged
}

func (c Scope) Variable(path VariablePath) (any, error) {
	for _, candidate := range c.candidates(path) {
		item, found := c.variables[candidate]
		if found {
			return item, nil
		}
	}

	return nil, ErrVariableNotFound
}

func (c Scope) WithNamespace(node VariablePathNode) Scope {
	if node == "" {
		return c
	}

	next := c.Clone()
	next.namespace = append(next.namespace, node)

	return next
}

func (c Scope) qualifyPath(path VariablePath) VariablePath {
	prefix := c.namespacePrefix()
	if prefix == "" {
		return path
	}

	pathStr := string(path)
	if strings.HasPrefix(pathStr, prefix+".") || pathStr == prefix {
		return path
	}

	return VariablePath(prefix + "." + pathStr)
}

func (c Scope) candidates(path VariablePath) []VariablePath {
	if path == "" {
		return nil
	}

	pathStr := string(path)
	all := make([]VariablePath, 0, len(c.namespace)+1)
	seen := map[VariablePath]bool{}

	for i := len(c.namespace); i > 0; i-- {
		prefix := joinNodes(c.namespace[:i])

		candidate := VariablePath(prefix + "." + pathStr)
		if !seen[candidate] {
			seen[candidate] = true

			all = append(all, candidate)
		}
	}

	raw := VariablePath(pathStr)
	if !seen[raw] {
		all = append(all, raw)
	}

	return all
}

func (c Scope) namespacePrefix() string {
	if len(c.namespace) == 0 {
		return ""
	}

	return joinNodes(c.namespace)
}

func joinNodes(nodes []VariablePathNode) string {
	parts := make([]string, len(nodes))
	for i, node := range nodes {
		parts[i] = string(node)
	}

	return strings.Join(parts, ".")
}
