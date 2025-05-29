package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/crowleyfelix/go-pipeline/pkg/expression"
	"github.com/crowleyfelix/go-pipeline/pkg/log"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

// RegisterStepExecutors registers all available step executors.
func RegisterStepExecutors() {
	RegisterStepExecutor("pipeline", PipelineExecutor)
	RegisterStepExecutor("set", SetExecutor)
	RegisterStepExecutor("range-json", RangeJSONExecutor)
	RegisterStepExecutor("wait", WaitExecutor)
	RegisterStepExecutor("stop", StopExecutor)
	RegisterStepExecutor("until", UntilExecutor)
	RegisterStepExecutor("log", LogExecutor)
	RegisterStepExecutor("fanout", FanoutExecutor)
}

// Step represents a single step in the pipeline with its ID, type, and parameters.
type Step struct {
	ID     VariablePathNode `yaml:"id"`
	Type   string           `yaml:"type"`
	Params map[string]any   `yaml:"params"`
}

// String returns a string representation of the step, including its type and ID.
func (s Step) String() string {
	str := fmt.Sprintf("step-%s", s.Type)

	if s.ID != "" {
		str = fmt.Sprintf("%s-%s", str, s.ID)
	}

	return str
}

func (s Step) VariablePath(pathNodes ...VariablePathNode) VariablePath {
	var allNodes = []VariablePathNode{}
	if s.ID != "" {
		allNodes = append(allNodes, s.ID)
	}

	allNodes = append(allNodes, pathNodes...)

	bpn := lo.Map(allNodes, func(key VariablePathNode, _ int) string { return string(key) })
	path := strings.Join(bpn, ".")

	return VariablePath(path)
}

// StepParams unmarshals a raw map into a typed struct of the specified type.
func StepParams[T any](raw map[string]any) (T, error) {
	var value T

	blob, err := yaml.Marshal(raw)
	if err != nil {
		return value, err
	}

	return value, yaml.Unmarshal(blob, &value)
}

// StepExecutors is a map of step executor functions keyed by their step type.
type StepExecutors map[string]StepExecutor

// Execute executes the executor for the given step type with the provided context.
func (p StepExecutors) Execute(ctx context.Context, scope Scope, step Step) (Scope, error) {
	log.Log().Debug(ctx, "Executing %s", step)

	executor, found := p[step.Type]

	if !found {
		return scope, fmt.Errorf("unknown step type: %s", step.Type)
	}

	return stepInterceptor(ctx, scope, step, executor)
}

// RegisterStepExecutor registers a step executor function with a given name.
func RegisterStepExecutor(name string, executor StepExecutor) {
	executors[name] = executor
}

// StepExecutor defines the function signature for a step executor.
type StepExecutor func(ctx context.Context, scope Scope, step Step) (Scope, error)

func PipelineExecutor(ctx context.Context, scope Scope, step Step) (Scope, error) {
	params, err := StepParams[Pipeline](step.Params)
	if err != nil {
		return scope, err
	}

	return params.Execute(ctx, scope)
}

// # SetExecutor sets a map[string]any in the context.
// Example YAML:
//
//	id: set-example
//	steps:
//	- id: 'setup'
//	  type: set
//	  params:
//	    counter: '1'
//	- id: 'plus'
//	  type: set
//	  params:
//	    counter: '{{ add (variableGet . "setup" "counter") 10 }}'
func SetExecutor(ctx context.Context, scope Scope, step Step) (Scope, error) {
	params, err := StepParams[expression.Field[map[string]any]](step.Params)
	if err != nil {
		return scope, err
	}

	value, err := params.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	return scope.WithVariable(step.VariablePath(), value), nil
}

// StopParams defines the parameters for the StopExecutor.
type StopParams struct {
	Condition expression.Bool          `yaml:"condition"`
	Message   expression.Field[string] `yaml:"message"`
	IsError   expression.Bool          `yaml:"is_error"`
}

// StopExecutor stops the pipeline execution if the condition evaluates to true.
// Example YAML:
//
//	id: 'stop-example'
//	steps:
//	- type: stop
//	  params:
//	  	condition: '{{ gt 2 1 | and (eq "true" "true") }}'
//	  	message: 'Stopping pipeline'
//	  	is_error: 'true'
func StopExecutor(ctx context.Context, scope Scope, step Step) (Scope, error) {
	params, err := StepParams[StopParams](step.Params)
	if err != nil {
		return scope, err
	}

	stop, err := params.Condition.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	msg, err := params.Message.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	isError, err := params.IsError.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	if isError {
		err = fmt.Errorf("stop error: %s", msg)
	}

	if stop {
		log.Log().Info(ctx, msg)

		scope.Finished = true

		return scope, err
	}

	return scope, nil
}

// RangeParams defines the parameters for the RangeExecutor.
type RangeParams struct {
	Source      expression.JSON[[]any] `yaml:"source"`
	Concurrency expression.Int         `yaml:"concurrency"`
	Pipeline    `yaml:",inline"`
}

// RangeJSONExecutor executes a pipeline for each item in the json source with optional concurrency.
// Example YAML:
//
//	id: range-example
//	steps:
//	- id: range
//	  type: range-json
//	  params:
//	  	source: '{{ list 1 2 3 | toJson }}'
//	  	concurrency: '{{ env "RANGE_CONCURRENCY" | default "2" }}'
//	  	steps:
//		- type: log
//	  	  params:
//	  		message: '{{ printf "Processing item %v: %v" ( variable . "range.$index") ( variable . "range" )}}'
func RangeJSONExecutor(ctx context.Context, scope Scope, step Step) (Scope, error) {
	params, err := StepParams[RangeParams](step.Params)
	if err != nil {
		return scope, err
	}

	source, err := params.Source.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	concurrency, err := params.Concurrency.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	if concurrency == 0 {
		concurrency = 1
	}

	return fanout(ctx, scope, concurrency, func(item any, i int) workerParams {
		return workerParams{
			Pipeline: params.Pipeline,
			Variables: map[VariablePath]any{
				step.VariablePath():              item,
				step.VariablePath(PathNodeIndex): i,
			},
		}
	}, source...)
}

// LogParams defines the parameters for the LogExecutor.
type LogParams struct {
	Message expression.Field[string] `yaml:"message"`
}

// LogExecutor logs a message to the context logger.
// Example YAML:
//
//	id: log-example
//	steps:
//	- type: log
//	  params:
//	  	message: '{{ printf "Step %s completed at %s" (variableGet . "some_step" "id") (now | date "2006-01-02 15:04:05") }}'
func LogExecutor(ctx context.Context, scope Scope, step Step) (Scope, error) {
	params, err := StepParams[LogParams](step.Params)
	if err != nil {
		return scope, err
	}

	message, err := params.Message.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	log.Log().Info(ctx, message)

	return scope, nil
}

// UntilParams defines the parameters for the UntilExecutor.
type UntilParams struct {
	Condition expression.Bool `yaml:"condition"`
	Pipeline  `yaml:",inline"`
}

// UntilExecutor executes a pipeline repeatedly until the condition evaluates to false.
// Example YAML:
//
//	id: until-example
//	steps:
//	- type: until
//	  params:
//	  	condition: '{{ lt (variableGet . "setup" "counter" | int) 5 }}'
//	  	steps:
//	  	- type: log
//	  	  params:
//	  	 	message: '{{ printf "Counter is %d" (variableGet . "setup" "counter") }}'
func UntilExecutor(ctx context.Context, scope Scope, step Step) (Scope, error) {
	params, err := StepParams[UntilParams](step.Params)
	if err != nil {
		return scope, err
	}

	proceed, err := params.Condition.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	for proceed && !scope.Finished {
		scope, err = params.Execute(ctx, scope)
		if err != nil {
			return scope, err
		}

		proceed, err = params.Condition.Eval(ctx, scope)
		if err != nil {
			return scope, err
		}
	}

	return scope, err
}

// WaitParams defines the parameters for the WaitExecutor.
type WaitParams struct {
	Duration expression.Duration `yaml:"duration"`
}

// WaitExecutor pauses the pipeline execution for the specified duration.
// Example YAML:
//
//	id: wait-example
//	steps:
//	- type: wait
//	  params:
//	    duration: '5s'
func WaitExecutor(ctx context.Context, scope Scope, step Step) (Scope, error) {
	params, err := StepParams[WaitParams](step.Params)
	if err != nil {
		return scope, err
	}

	duration, err := params.Duration.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	time.Sleep(duration)

	return scope, nil
}

type FanoutParams struct {
	Concurrency expression.Int `yaml:"concurrency"`
	Pipelines   []Pipeline     `yaml:"pipelines"`
}

// FanoutExecutor executes multiple pipelines concurrently.
// Example YAML:
//
//	id: fanout-example
//	steps:
//	- type: fanout
//	  params:
//	 	pipelines:
//		- id: 'pipe1'
//		  steps:
//		  - type: log
//		    params:
//		      message: 'Running pipeline 1'
//		- id: 'pipe2'
//		  steps:
//		  - type: log
//		    params:
//		      message: 'Running pipeline 2'
func FanoutExecutor(ctx context.Context, scope Scope, step Step) (Scope, error) {
	params, err := StepParams[FanoutParams](step.Params)
	if err != nil {
		return scope, err
	}

	concurrency, err := params.Concurrency.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	if concurrency == 0 {
		concurrency = len(params.Pipelines)
	}

	pipelines := params.Pipelines

	return fanout(ctx, scope, concurrency, func(item Pipeline, i int) workerParams {
		return workerParams{Pipeline: item}
	}, pipelines...)
}

func fanout[T any](ctx context.Context, scope Scope, concurrency int, mapper func(item T, i int) workerParams, items ...T) (Scope, error) {
	in := make(chan workerParams, concurrency)
	out := make(chan workerResult, concurrency)

	defer func() {
		close(in)
	}()

	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	for range concurrency {
		go worker(ctx, scope, in, out)
	}

	go func() {
		for i, item := range items {
			in <- mapper(item, i)
		}
	}()

	for range len(items) {
		if scope.Finished {
			return scope, nil
		}

		result := <-out
		if result.error != nil {
			return scope, result.error
		}

		scope = scope.Merge(result.Scope)
	}

	return scope, nil
}

type workerParams struct {
	Pipeline
	Variables map[VariablePath]any
}

type workerResult struct {
	Scope
	error
}

func worker(ctx context.Context, scope Scope, in chan workerParams, out chan workerResult) {
	defer func() {
		if r := recover(); r != nil {
			out <- workerResult{scope, fmt.Errorf("panic: %v", r)}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case input, closed := <-in:
			if !closed {
				return
			}

			scope = scope.Clone().WithVariables(input.Variables)

			var err error
			scope, err = input.Execute(ctx, scope)
			out <- workerResult{scope, err}
		}
	}
}
