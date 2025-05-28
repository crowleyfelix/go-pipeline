package pipeline

import (
	"github.com/crowleyfelix/go-pipeline/pkg/expression"
)

var (
	executors       StepExecutors
	interceptor     Interceptor
	stepInterceptor StepInterceptor
)

func init() {
	expression.RegisterFuncs(templateFuncs)

	executors = StepExecutors{}

	RegisterStepExecutors()
	SetInterceptor(defaultInterceptor)
	SetStepInterceptor(defaultStepInterceptorfunc)
}
