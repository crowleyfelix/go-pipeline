package pipeline

import (
	"context"
	"time"

	"github.com/crowleyfelix/go-pipeline/pkg/log"
)

type Executor func(ctx context.Context, scope Scope) (Scope, error)

// Interceptor defines a function that intercepts a pipeline execution.
type Interceptor func(ctx context.Context, scope Scope, pipeline Pipeline, execute Executor) (Scope, error)

// StepInterceptor defines a function that intercepts the execution of a step within a pipeline.
type StepInterceptor func(ctx context.Context, scope Scope, step Step, executor StepExecutor) (Scope, error)

func SetInterceptor(itc Interceptor) {
	interceptor = itc
}

func SetStepInterceptor(itc StepInterceptor) {
	stepInterceptor = itc
}

func defaultInterceptor(ctx context.Context, scope Scope, pipeline Pipeline, executor Executor) (Scope, error) {
	start := time.Now()
	scope, err := executor(ctx, scope)
	end := time.Now()
	log.Log().Info(ctx, "Pipeline %s executed in %s", pipeline, end.Sub(start))

	return scope, err
}

func defaultStepInterceptorfunc(ctx context.Context, scope Scope, step Step, executor StepExecutor) (Scope, error) {
	start := time.Now()
	scope, err := executor(ctx, scope, step)
	end := time.Now()
	log.Log().Info(ctx, "Step %s executed in %s", step, end.Sub(start))

	return scope, err
}
