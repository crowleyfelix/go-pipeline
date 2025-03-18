package pipeline

import (
	"github.com/crowleyfelix/go-pipeline/pkg/expression"
)

var processors StepProcessors

func init() {
	expression.RegisterFuncs(templateFuncs)

	processors = StepProcessors{}

	RegisterProcessors()
}
