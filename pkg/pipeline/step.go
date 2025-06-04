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
	RegisterStepExecutor("pipeline", TypedStepExecutor[Pipeline](PipelineExecutor))
	RegisterStepExecutor("set", TypedStepExecutor[SetParams](SetExecutor))
	RegisterStepExecutor("range", TypedStepExecutor[RangeParams](RangeExecutor))
	RegisterStepExecutor("wait", TypedStepExecutor[WaitParams](WaitExecutor))
	RegisterStepExecutor("stop", TypedStepExecutor[StopParams](StopExecutor))
	RegisterStepExecutor("until", TypedStepExecutor[UntilParams](UntilExecutor))
	RegisterStepExecutor("log", TypedStepExecutor[LogParams](LogExecutor))
	RegisterStepExecutor("fanout", TypedStepExecutor[FanoutParams](FanoutExecutor))
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

	scope, err := stepInterceptor(ctx, scope, step, executor)
	if err != nil {
		err = fmt.Errorf("error executing step %s: %w", step, err)
	}

	return scope, err
}

// RegisterStepExecutor registers a step executor function with a given name.
func RegisterStepExecutor(name string, executor StepExecutor) {
	executors[name] = executor
}

type TypedStepExecutor[Params any] func(ctx context.Context, scope Scope, step Step, params Params) (Scope, error)

func (f TypedStepExecutor[Params]) Execute(ctx context.Context, scope Scope, step Step) (Scope, error) {
	var params Params

	blob, err := yaml.Marshal(step.Params)
	if err != nil {
		return scope, err
	}

	if err := yaml.Unmarshal(blob, &params); err != nil {
		return scope, err
	}

	return f(ctx, scope, step, params)
}

// StepExecutor defines the interface for executing a step in the pipeline.
type StepExecutor interface {
	Execute(ctx context.Context, scope Scope, step Step) (Scope, error)
}

func PipelineExecutor(ctx context.Context, scope Scope, step Step, params Pipeline) (Scope, error) {
	return params.Execute(ctx, scope)
}

type SetParams struct {
	expression.YAML[map[string]any] `yaml:",inline"`
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
func SetExecutor(ctx context.Context, scope Scope, step Step, params SetParams) (Scope, error) {
	value, err := params.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	return scope.WithVariable(step.VariablePath(), value), nil
}

// StopParams defines the parameters for the StopExecutor.
type StopParams struct {
	Condition expression.Bool   `yaml:"condition"`
	Message   expression.String `yaml:"message"`
	IsError   expression.Bool   `yaml:"is_error"`
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
func StopExecutor(ctx context.Context, scope Scope, step Step, params StopParams) (Scope, error) {
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
	Items       []any                  `yaml:"items"`
	Variable    VariablePath           `yaml:"variable"`
	JSON        expression.JSON[[]any] `yaml:"json"`
	Concurrency expression.Int         `yaml:"concurrency"`
	Pipeline    `yaml:",inline"`
}

// RangeExecutor executes a pipeline for each item in the source with optional concurrency.
// Example YAML:
//
//	id: range-example
//	steps:
//	- id: range
//	  type: range
//	  params:
//		items: [1, 2, 3]
//	  	variable: 'step-id'
//	  	json: '{{ list 4 5 6 | toJson }}'
//	  	concurrency: '{{ env "RANGE_CONCURRENCY" | default "2" }}'
//	  	steps:
//		- type: log
//	  	  params:
//	  		message: '{{ printf "Processing item %v: %v" ( variable . "range.$index") ( variable . "range" )}}'
func RangeExecutor(ctx context.Context, scope Scope, step Step, params RangeParams) (Scope, error) {
	items := params.Items

	if params.Variable != "" {
		variable, err := scope.Variable(params.Variable)
		if err != nil {
			return scope, err
		}

		v, ok := variable.([]any)
		if !ok {
			return scope, fmt.Errorf("variable %s is not a slice", params.Variable)
		}

		items = append(items, v)
	}

	if params.JSON != "" {
		json, err := params.JSON.Eval(ctx, scope)
		if err != nil {
			return scope, err
		}

		items = append(items, json...)
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
	}, items...)
}

// LogParams defines the parameters for the LogExecutor.
type LogParams struct {
	Message expression.String `yaml:"message"`
}

// LogExecutor logs a message to the context logger.
// Example YAML:
//
//	id: log-example
//	steps:
//	- type: log
//	  params:
//	  	message: '{{ printf "Step %s completed at %s" (variableGet . "some_step" "id") (now | date "2006-01-02 15:04:05") }}'
func LogExecutor(ctx context.Context, scope Scope, step Step, params LogParams) (Scope, error) {
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
func UntilExecutor(ctx context.Context, scope Scope, step Step, params UntilParams) (Scope, error) {
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
func WaitExecutor(ctx context.Context, scope Scope, step Step, params WaitParams) (Scope, error) {
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
func FanoutExecutor(ctx context.Context, scope Scope, step Step, params FanoutParams) (Scope, error) {
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
