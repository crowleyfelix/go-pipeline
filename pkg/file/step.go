package file

import (
	"context"
	"os"

	"github.com/crowleyfelix/go-pipeline/pkg/expression"
	"github.com/crowleyfelix/go-pipeline/pkg/pipeline"
)

const fileMode = 0644

func RegisterStepExecutors() {
	pipeline.RegisterStepExecutor("file-write", pipeline.TypedStepExecutor[WriteParams](WriteExecutor))
}

type WriteParams struct {
	Path   expression.String `yaml:"path"`
	Text   expression.String `yaml:"text"`
	Append expression.Bool   `yaml:"append"`
}

// WriteExecutor writes the provided text to a file at the specified path.
// It supports appending to the file if the `append` parameter is set to true.
//
// Example YAML:
//
//	id: write-example
//	steps:
//	- id: write-step
//	  type: file-write
//	  params:
//	    path: './output.txt'
//	    text: 'Hello, World!'
//	    append: true
func WriteExecutor(ctx context.Context, scope pipeline.Scope, step pipeline.Step, params WriteParams) (pipeline.Scope, error) {
	path, err := params.Path.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	text, err := params.Text.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	append, err := params.Append.Eval(ctx, scope)
	if err != nil {
		return scope, err
	}

	var (
		file *os.File
	)

	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC

	if append {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}

	//nolint:gosec // ignore G304: Use of the os package is safe here.
	file, err = os.OpenFile(path, flag, fileMode)

	if err != nil {
		return scope, err
	}

	defer func() {
		_ = file.Close()
	}()

	n, err := file.WriteString(text)

	return scope.WithVariable(step.VariablePath(), n), err
}
