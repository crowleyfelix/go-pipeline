# Project Guidelines

## Code Style
- Follow existing Go style and keep package boundaries small and explicit (see `pkg/pipeline`, `pkg/expression`).
- Prefer adding or extending typed step params via `expression.*` wrappers (for example `expression.String`, `expression.YAML[...]`) so runtime template evaluation remains consistent.
- Keep YAML tags explicit on pipeline/step param structs.
- Avoid introducing global mutable state outside existing registries (`pipeline.RegisterStepExecutor`, `expression.RegisterFuncs`).

## Architecture
- CLI entrypoint is `cmd/pipeline/main.go`.
- Core execution engine lives in `pkg/pipeline`:
  - `pipeline.Load` reads `*.yaml` files from the configured FS and indexes pipelines by `id`.
  - `Pipelines.Execute` orchestrates selected pipeline IDs in order.
  - `Pipeline.Execute` runs `uses` first (if set), then executes steps sequentially unless `scope.Finished`.
  - Step execution is registry-based (`RegisterStepExecutor`) with typed adapters (`TypedStepExecutor`).
  - Interceptor hooks exist for pipeline and step timing/logging (`pkg/pipeline/interceptor.go`).
- Built-in step types are registered in `pkg/pipeline/step.go`; plugin step packages (for example `pkg/http`, `pkg/file`) must be registered by callers before use.
- Scope variables are the data bus between steps (`pkg/pipeline/scope.go`).

## Build and Test
- Use `make` targets from repo root:
  - `make run` to run CLI (`go run cmd/pipeline/*.go`).
  - `make build` to build binary to `bin/app`.
  - `make test` for unit tests + coverage profile generation.
  - `make test-cover` for HTML coverage.
  - `make lint` / `make lint-fix` for linting.
  - `make deps` to refresh `vendor/`.
- Environment/development bootstrap: see `docs/CONTRIBUTING.md` (`make init`, `make docker-up run`).

## Conventions
- Pipeline definitions are YAML files with top-level `id` and `steps`.
- Step IDs become variable path prefixes in scope. Reusing step IDs can overwrite values.
- Metadata nodes use `$` prefixes (for example `step_id.$body`, `range.$index`).
- Step params are template expressions (Go `text/template` + Sprig + custom functions). Prefer existing functions:
  - `variable`
  - `variableGet`
  - `jsonPath`
  - `read`
- When adding a new step type:
  - Define typed params.
  - Register executor in the appropriate init/registration path.
  - Add or update an example under `example/`.

## Known Pitfalls
- CLI env vars used by code are `PIPELINE_DIR` and `PIPELINE_IDS` (comma-separated).
- `pipeline.Load` recursively loads `*.yaml` and `*.yml` from nested folders under the provided FS root.
- HTTP and file plugin steps are unavailable unless `http.RegisterStepExecutor(...)` and `file.RegisterStepExecutors()` are called before execution.
