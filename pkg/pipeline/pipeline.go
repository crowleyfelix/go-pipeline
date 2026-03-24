package pipeline

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/crowleyfelix/go-pipeline/pkg/log"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

// Pipelines represents a collection of pipelines that can be executed.
type Pipelines struct {
	pipelines map[string]Pipeline
}

// Execute runs the specified pipelines by their IDs in the given context.
// It creates a Datadog span for each pipeline execution and returns the updated context or an error if any pipeline fails.
func (p Pipelines) Execute(ctx context.Context, scope Scope, ids ...string) (Scope, error) {
	for _, id := range ids {
		pipe, ok := p.pipelines[id]
		if !ok {
			return scope, fmt.Errorf("Pipeline %s not found: available %+v", id, lo.Keys(p.pipelines))
		}

		var err error
		scope, err = pipe.Execute(ctx, scope)

		if err != nil {
			return scope, err
		}
	}

	return scope, nil
}

// Pipeline represents a single pipeline with an ID and a sequence of steps to execute.
type Pipeline struct {
	Uses        string `yaml:"uses"`
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
}

// Load creates a new Pipelines instance by loading pipeline definitions from the provided file system.
// It reads all YAML files, unmarshals them into Pipeline objects, and maps them by their IDs.
func Load(fileSystem fs.FS) (Pipelines, error) {
	pipelines := make(map[string]Pipeline)

	err := fs.WalkDir(fileSystem, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || (!strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml")) {
			return nil
		}

		blob, err := fs.ReadFile(fileSystem, name)
		if err != nil {
			return err
		}

		var pipe Pipeline

		if err := yaml.Unmarshal(blob, &pipe); err != nil {
			return err
		}

		pipelines[pipe.ID] = pipe

		return nil
	})
	if err != nil {
		return Pipelines{}, err
	}

	return Pipelines{
		pipelines: pipelines,
	}, nil
}

// Execute runs all the steps in the pipeline in the given context.
// It logs the execution progress and returns the updated context or an error if any step fails.
func (p Pipeline) Execute(ctx context.Context, scope Scope) (Scope, error) {
	return interceptor(ctx, scope, p, func(ctx context.Context, scope Scope) (Scope, error) {
		log.Log().Info(ctx, "Executing pipeline %s", p)

		var err error

		if p.Uses != "" {
			scope, err = scope.Pipelines.Execute(ctx, scope, p.Uses)
			if err != nil {
				return scope, err
			}
		}

		for _, step := range p.Steps {
			if scope.Finished {
				return scope, nil
			}

			scope, err = executors.Execute(ctx, scope, step)

			if err != nil {
				log.Log().Error(ctx, "Error executing step %s: %s", step, err)

				return scope, err
			}
		}

		log.Log().Info(ctx, "Executed pipeline %s", p)

		return scope, nil
	})
}

func (p Pipeline) String() string {
	if p.ID == "" {
		return "anonymous"
	}

	return p.ID
}
