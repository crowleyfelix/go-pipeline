# go-pipeline

A toolkit to build processing pipelines using yaml.

## Table of contents

- [How does it work](#how-does-it-work)
- [How to](#how-to)
- [Available steps](#available-steps)
- [Customize](#customize)
- [Contributing](./docs/CONTRIBUTING)

## How does it work?

A **pipeline** is a set of **steps** to be executed in sequence. Every step has a **type** that defines what kind of processing should be done under a given **context**, and returns the context modified.

```mermaid
flowchart LR
    subgraph pipeline 
        direction LR
        A["Step 1"] -- context --> B
        B["Step 2"] -- context --> C["Step 3"]
    end
```

The step can modify the context by adding items in the **baggage**, and this baggage is carried over the whole pipeline. The item is only added in the baggage if the step has and id or has a reserved key (started with $).

```mermaid
stateDiagram
    direction LR
    state "baggage{ id1: somevalue }" as baggage1
    state "baggage{ id1: somevalue, id2: foo }" as baggage2
    state "baggage{ id1: somevalue, id2: modified, id3.$reserved: something }" as baggage3

    [*] --> baggage1
    baggage1 --> baggage2
    baggage2 --> baggage3
    baggage3 --> [*]
```

During the step execution, their params can be dynamically evaluated along with the context throught **expressions** following the [go template](https://pkg.go.dev/text/template). To use a processing result from a previous step, use the "baggage" or "baggageDict" function passing the item path (an id or an id + reserved key, separated by dots).

```mermaid
stateDiagram
    direction LR
    state "param.field: '{{ gt 2 1 }}'" as field1
    state "param.field: 'true'" as field2
    [*] --> field1
    field1 --> field2
    field2 --> [*]
```

## How to

Create an YAML with the pipeline definition.

```yaml
id: range-example
steps:   
- id: some-step
  type: set
  params:
    list: '{{ list 1 2 3 4 5 6 7 8 9 10 | toJson }}'
- type: range-json
  params:
    source: '{{ baggageDict . "some-step" "list" }}'
    concurrency: '{{ env "RANGE_CONCURRENCY" | default "2" }}'
    steps:
    - type: log
      params:
        message: '{{ printf "Processing %d item: %v" ( baggage . "$rangeIndex" ) ( baggage . "$rangeItem" )}}'
```

Load the pipeline passing the folder path, and execute.

```go
import (
  "context"
  "os"
  "github.com/crowleyfelix/go-pipeline/pkg/pipeline"
)

func main() {
  pipelines, err := pipeline.Load(os.DirFS(os.Getenv("PIPELINE_DIR")))
  if err != nil {
    panic(err)
  }

  ctx := pipeline.NewContext(context.Background(), pipelines)
  ctx, err := pipelines.Execute(ctx, "range-example")
  if err != nil && err != context.Canceled {
    panic(err)
  }
}
```

or execute the cli

```bash
PIPELINE_FOLDER=./example PIPELINE_IDS=range-example go run cmd/*.go
```

You can see more examples [here](./example/).

## Available steps

### Basic

| **Step Type**       | **Parameter**       | **Type**               | **Description**                                                                                     |
|----------------------|---------------------|------------------------|-----------------------------------------------------------------------------------------------------|
| **set**              | `params`           | `map[string]any`      | Key-value pairs to set in the pipeline context.                                                   |
| **stop**             | `condition`        | `bool`                | Condition to stop the pipeline.                                                                   |
|                      | `message`          | `string`              | Message to log when stopping the pipeline.                                                        |
|                      | `is_error`         | `bool`                | Whether stopping the pipeline should be treated as an error.                                       |
| **range-json**       | `source`           | `json`                | JSON array to iterate over.                                                                       |
|                      | `concurrency`      | `int`                 | Number of concurrent executions.                                                                  |
|                      | `steps`            | `[]step`              | Steps to execute for each item in the JSON array.                                                 |
| **log**              | `message`          | `string`              | Message to log. Can use Go templates for dynamic content.                                          |
| **until**            | `condition`        | `bool`                | Condition to evaluate for repeating the pipeline.                                                 |
|                      | `steps`            | `[]step`              | Steps to execute repeatedly until the condition is false.                                         |
| **wait**             | `duration`         | `duration`            | Duration to wait before proceeding to the next step.

### Plugins

The following steps should be registered before it's used.

eg.:

```go
import (
  httplib "net/http"
  "github.com/crowleyfelix/go-pipeline/pkg/http"
)

func main() {
  http.RegisterProcessor(httplib.DefaultClient)
}

```

| **Step Type**       | **Parameter**       | **Type**               | **Description**                                                                                     |
|----------------------|---------------------|------------------------|-----------------------------------------------------------------------------------------------------|
| **http**            | `url`              | `string`              | The URL to send the HTTP request to. Supports Go templates and pipeline contexts.                  |
|                      | `method`           | `string`              | The HTTP method (e.g., GET, POST).                                                                |
|                      | `body`             | `string`              | The body of the HTTP request. Can use Go templates for dynamic content.                            |
|                      | `header`           | `map[string]string`   | HTTP headers as key-value pairs.                                                                  |

## Go Template Functions


| **Function**         | **Description**                                                                                     | **Example**                                                                                     |
|-----------------------|-----------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------|
| `baggage`            | Retrieves an item from the pipeline context baggage using its path.                                  | `{{ baggage . "step-id" }}`                                                                   |
| `baggageDict`        | Retrieves a specific key from a dictionary stored in the pipeline context baggage.                   | `{{ baggageDict . "step-id" "key" }}`                                                         |
| `jsonPath`           | Extracts data from a JSON string using a JSONPath expression.                                        | `{{ jsonPath "$.items[0].name" "{\"items\": [{\"name\": \"example\"}]}" }}`                   |

Besides the standard library functions, all functions from the [sprig](https://masterminds.github.io/sprig/) library are availble.

## Customize

It's possible to extend the go-pipeline by registering step processors and go template functions.

```go
package main

import (
  "fmt"
  "html/template"

  "github.com/crowleyfelix/go-pipeline/pkg/expression"
  "github.com/crowleyfelix/go-pipeline/pkg/pipeline"
)

type CustomParams struct {
  Name expression.Field[string] `yaml:"name"`
}

func main() {
  pipeline.RegisterProcessor("custom", func(ctx pipeline.Context, step pipeline.Step) (pipeline.Context, error) {
    params, err := pipeline.StepParams[CustomParams](step.Params)
    if err != nil {
      return ctx, err
    }

    name, err := params.Name.Eval(ctx)
    if err != nil {
      return ctx, err
    }

    value := fmt.Sprintf("officer %s", name)

    return ctx.WithBaggage(step.BaggagePath(), value), nil
  })

  expression.RegisterFuncs(template.FuncMap{
    "greeting": func(ctx pipeline.Context, path pipeline.BaggagePath) (string, error) {
      item, err := ctx.BaggageItem(path)

      return fmt.Sprintf("hello, %s", item), err
    },
  })
}
```

And the registered plugins can be used like this

```yaml
id: my-pipeline
steps:
- id: previous-step
  type: custom
  params:
    name: 'Bob'
- type: log
  params:
    message: '{{ greeting . "previous-step" }}'
```
