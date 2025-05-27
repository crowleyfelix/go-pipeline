package pipeline

import (
	"fmt"
	"io"
	"io/fs"

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
func (p Pipelines) Execute(ctx Context, ids ...string) (Context, error) {
	for _, id := range ids {
		pipe, ok := p.pipelines[id]
		if !ok {
			return ctx, fmt.Errorf("Pipeline %s not found: available %+v", id, lo.Keys(p.pipelines))
		}

		var err error
		ctx, err = pipe.Execute(ctx)

		if err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

// Pipeline represents a single pipeline with an ID and a sequence of steps to execute.
type Pipeline struct {
	ID    string `yaml:"id"`
	Steps []Step `yaml:"steps"`
}

// Load creates a new Pipelines instance by loading pipeline definitions from the provided file system.
// It reads all YAML files, unmarshals them into Pipeline objects, and maps them by their IDs.
func Load(fileSystem fs.FS) (Pipelines, error) {
	pipelines := make(map[string]Pipeline)

	names, err := fs.Glob(fileSystem, "*.yaml")
	if err != nil {
		return Pipelines{}, err
	}

	for _, name := range names {
		fl, err := fileSystem.Open(name)
		if err != nil {
			return Pipelines{}, err
		}

		blob, err := io.ReadAll(fl)
		if err != nil {
			return Pipelines{}, err
		}

		var pipe Pipeline

		err = yaml.Unmarshal(blob, &pipe)
		if err != nil {
			return Pipelines{}, err
		}

		pipelines[pipe.ID] = pipe
	}

	return Pipelines{
		pipelines: pipelines,
	}, nil
}

// Execute runs all the steps in the pipeline in the given context.
// It logs the execution progress and returns the updated context or an error if any step fails.
func (p Pipeline) Execute(ctx Context) (Context, error) {
	return interceptor(ctx, p, func(ctx Context) (Context, error) {
		log.Log().Info(ctx, "Executing pipeline %s", p)

		var err error

		for _, step := range p.Steps {
			select {
			case <-ctx.Done():
				return ctx, ctx.Err()
			default:
				ctx, err = processors.Execute(ctx, step)

				if err != nil {
					log.Log().Error(ctx, "Error executing step %s: %s", step, err)

					return ctx, err
				}
			}
		}

		log.Log().Info(ctx, "Executed pipeline %s", p)

		return ctx, nil
	})
}

func (p Pipeline) String() string {
	if p.ID == "" {
		return "unidentified"
	}

	return p.ID
}
