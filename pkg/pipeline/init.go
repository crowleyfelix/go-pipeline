package pipeline

import (
	"github.com/crowleyfelix/go-pipeline/pkg/expression"
)

var (
	processors      StepProcessors
	interceptor     Interceptor
	stepInterceptor StepInterceptor
)

func init() {
	expression.RegisterFuncs(templateFuncs)

	processors = StepProcessors{}

	RegisterProcessors()
	SetInterceptor(defaultInterceptor)
	SetStepInterceptor(defaultStepInterceptorfunc)
}
