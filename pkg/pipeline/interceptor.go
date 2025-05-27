package pipeline

import (
	"time"

	"github.com/crowleyfelix/go-pipeline/pkg/log"
)

// Interceptor defines a function that intercepts a pipeline execution.
type Interceptor func(ctx Context, pipeline Pipeline, execute func(ctx Context) (Context, error)) (Context, error)

// StepInterceptor defines a function that intercepts the execution of a step within a pipeline.
type StepInterceptor func(ctx Context, step Step, processor StepProcessor) (Context, error)

func SetInterceptor(itc Interceptor) {
	interceptor = itc
}

func SetStepInterceptor(itc StepInterceptor) {
	stepInterceptor = itc
}

func defaultInterceptor(ctx Context, pipeline Pipeline, executor func(ctx Context) (Context, error)) (Context, error) {
	start := time.Now()
	ctx, err := executor(ctx)
	end := time.Now()
	log.Log().Info(ctx, "Pipeline %s executed in %s", pipeline, end.Sub(start))

	return ctx, err
}

func defaultStepInterceptorfunc(ctx Context, step Step, processor StepProcessor) (Context, error) {
	start := time.Now()
	ctx, err := processor(ctx, step)
	end := time.Now()
	log.Log().Info(ctx, "Step %s executed in %s", step, end.Sub(start))

	return ctx, err
}
