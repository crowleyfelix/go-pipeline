package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/crowleyfelix/go-pipeline/pkg/expression"
	"github.com/crowleyfelix/go-pipeline/pkg/log"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

// RegisterProcessors registers all available step processors.
func RegisterProcessors() {
	RegisterProcessor("set", SetProcessor)
	RegisterProcessor("range-json", RangeJSONProcessor)
	RegisterProcessor("wait", WaitProcessor)
	RegisterProcessor("stop", StopProcessor)
	RegisterProcessor("until", UntilProcessor)
	RegisterProcessor("log", LogProcessor)
}

// Step represents a single step in the pipeline with its ID, type, and parameters.
type Step struct {
	ID     BaggagePathNode `yaml:"id"`
	Type   string          `yaml:"type"`
	Params map[string]any  `yaml:"params"`
}

// String returns a string representation of the step, including its type and ID.
func (s Step) String() string {
	str := fmt.Sprintf("step-%s", s.Type)

	if s.ID != "" {
		str = fmt.Sprintf("%s-%s", str, s.ID)
	}

	return str
}

func (s Step) BaggagePath(pathNodes ...BaggagePathNode) BaggagePath {
	var allNodes = []BaggagePathNode{}
	if s.ID != "" {
		allNodes = append(allNodes, s.ID)
	}

	allNodes = append(allNodes, pathNodes...)

	bpn := lo.Map(allNodes, func(key BaggagePathNode, _ int) string { return string(key) })
	path := strings.Join(bpn, ".")

	return BaggagePath(path)
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

// StepProcessors is a map of step processor functions keyed by their step type.
type StepProcessors map[string]StepProcessor

// Execute executes the processor for the given step type with the provided context.
func (p StepProcessors) Execute(ctx Context, step Step) (Context, error) {
	log.Log().Debug(ctx, "Executing %s", step)

	processor, found := p[step.Type]

	if !found {
		return ctx, fmt.Errorf("unknown step type: %s", step.Type)
	}

	return processor(ctx, step)
}

// RegisterProcessor registers a step processor function with a given name.
func RegisterProcessor(name string, processor StepProcessor) {
	processors[name] = processor
}

// StepProcessor defines the function signature for a step processor.
type StepProcessor func(ctx Context, step Step) (Context, error)

// # SetProcessor sets a map[string]any in the context.
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
//	    counter: '{{ add (baggageDict . "setup" "counter") 10 }}'
func SetProcessor(ctx Context, step Step) (Context, error) {
	params, err := StepParams[expression.Field[map[string]any]](step.Params)
	if err != nil {
		return ctx, err
	}

	value, err := params.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	return ctx.WithBaggage(step.BaggagePath(), value), nil
}

// StopParams defines the parameters for the StopProcessor.
type StopParams struct {
	Condition expression.Bool          `yaml:"condition"`
	Message   expression.Field[string] `yaml:"message"`
	IsError   expression.Bool          `yaml:"is_error"`
}

// StopProcessor stops the pipeline execution if the condition evaluates to true.
// Example YAML:
//
//	id: 'stop-example'
//	steps:
//	- type: stop
//	  params:
//	  	condition: '{{ gt 2 1 | and (eq "true" "true") }}'
//	  	message: 'Stopping pipeline'
//	  	is_error: 'true'
func StopProcessor(ctx Context, step Step) (Context, error) {
	params, err := StepParams[StopParams](step.Params)
	if err != nil {
		return ctx, err
	}

	stop, err := params.Condition.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	msg, err := params.Message.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	isError, err := params.IsError.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	if isError {
		err = fmt.Errorf("stop error: %s", msg)
	}

	if stop {
		log.Log().Info(ctx, msg)
		ctx.Stop(err)

		return ctx, err
	}

	return ctx, nil
}

// RangeParams defines the parameters for the RangeProcessor.
type RangeParams struct {
	Source      expression.JSON[[]any] `yaml:"source"`
	Concurrency expression.Int         `yaml:"concurrency"`
	Pipeline    `yaml:",inline"`
}

// RangeJSONProcessor executes a pipeline for each item in the json source with optional concurrency.
// Example YAML:
//
//	id: range-example
//	steps:
//	- type: range-json
//	  params:
//	  	source: '{{ list 1 2 3 | toJson }}'
//	  	concurrency: '{{ env "RANGE_CONCURRENCY" | default "2" }}'
//	  	steps:
//		- type: log
//	  	  params:
//	  		message: '{{ printf "Processing item: %v" ( baggage . "$rangeItem" )}}'
func RangeJSONProcessor(ctx Context, step Step) (Context, error) {
	params, err := StepParams[RangeParams](step.Params)
	if err != nil {
		return ctx, err
	}

	source, err := params.Source.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	concurrency, err := params.Concurrency.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	if concurrency == 0 {
		concurrency = 1
	}

	in := make(chan map[BaggagePath]any, concurrency)
	out := make(chan contextError, concurrency)

	defer func() {
		close(in)
	}()

	for range concurrency {
		go execAsync(ctx, params.Pipeline, in, out)
	}

	go func() {
		for i, item := range source {
			i := i
			item := item

			in <- map[BaggagePath]any{
				step.BaggagePath(BaggageKeyRangeItem):  item,
				step.BaggagePath(BaggageKeyRangeIndex): i,
			}
		}
	}()

	for range len(source) {
		select {
		case <-ctx.Done():
			return ctx, ctx.Err()

		case result := <-out:
			if result.error != nil {
				return ctx, err
			}

			// If greather than 1 the context should be lost, due to no
			// garanties of the step execution order
			if concurrency <= 1 {
				ctx = result.Context
			}
		}
	}

	return ctx, nil
}

// LogParams defines the parameters for the LogProcessor.
type LogParams struct {
	Message expression.Field[string] `yaml:"message"`
}

// LogProcessor logs a message to the context logger.
// Example YAML:
//
//	id: log-example
//	steps:
//	- type: log
//	  params:
//	  	message: '{{ printf "Step %s completed at %s" (baggageDict . "some_step" "id") (now | date "2006-01-02 15:04:05") }}'
func LogProcessor(ctx Context, step Step) (Context, error) {
	params, err := StepParams[LogParams](step.Params)
	if err != nil {
		return ctx, err
	}

	message, err := params.Message.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	log.Log().Info(ctx, message)

	return ctx, nil
}

type contextError struct {
	Context
	error
}

// execAsync executes a pipeline asynchronously for each input item.
func execAsync(ctx Context, pipe Pipeline, in chan map[BaggagePath]any, out chan contextError) {
	defer func() {
		if r := recover(); r != nil {
			out <- contextError{ctx, fmt.Errorf("panic: %v", r)}
		}
	}()

	parentCtx := ctx

	for {
		select {
		case <-ctx.Done():
			return
		case items, closed := <-in:
			if !closed {
				return
			}

			ctx = parentCtx.Clone().WithBaggageItems(items)

			var err error
			ctx, err = pipe.Execute(ctx)
			out <- contextError{ctx, err}
		}
	}
}

// UntilParams defines the parameters for the UntilProcessor.
type UntilParams struct {
	Condition expression.Bool `yaml:"condition"`
	Pipeline  `yaml:",inline"`
}

// UntilProcessor executes a pipeline repeatedly until the condition evaluates to false.
// Example YAML:
//
//	id: until-example
//	steps:
//	- type: until
//	  params:
//	  	condition: '{{ lt (baggageDict . "setup" "counter" | int) 5 }}'
//	  	steps:
//	  	- type: log
//	  	  params:
//	  	 	message: '{{ printf "Counter is %d" (baggageDict . "setup" "counter") }}'
func UntilProcessor(ctx Context, step Step) (Context, error) {
	params, err := StepParams[UntilParams](step.Params)
	if err != nil {
		return ctx, err
	}

	proceed, err := params.Condition.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	for proceed {
		select {
		case <-ctx.Done():
			return ctx, ctx.Err()
		default:
			ctx, err = params.Execute(ctx)
			if err != nil {
				return ctx, err
			}

			proceed, err = params.Condition.Eval(ctx)
			if err != nil {
				return ctx, err
			}
		}
	}

	return ctx, err
}

// WaitParams defines the parameters for the WaitProcessor.
type WaitParams struct {
	Duration expression.Duration `yaml:"duration"`
}

// WaitProcessor pauses the pipeline execution for the specified duration.
// Example YAML:
//
//	id: wait-example
//	steps:
//	- type: wait
//	  params:
//	    duration: '5s'
func WaitProcessor(ctx Context, step Step) (Context, error) {
	params, err := StepParams[WaitParams](step.Params)
	if err != nil {
		return ctx, err
	}

	duration, err := params.Duration.Eval(ctx)
	if err != nil {
		return ctx, err
	}

	time.Sleep(duration)

	return ctx, nil
}
