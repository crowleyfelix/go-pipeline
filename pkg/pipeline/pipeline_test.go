package pipeline

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestLoadLoadsPipelinesFromNestedFolders(t *testing.T) {
	t.Parallel()

	fileSystem := fstest.MapFS{
		"root.yaml":                  {Data: []byte("name: root-pipeline\nsteps: []\n")},
		"nested/child.yaml":          {Data: []byte("name: child-pipeline\ndescription: child\nsteps: []\n")},
		"nested/deep/grandchild.yml": {Data: []byte("name: grandchild-pipeline\nsteps: []\n")},
		"nested/ignored.txt":         {Data: []byte("this should be ignored")},
	}

	pipelines, err := Load(fileSystem)
	if !assert.NoError(t, err) {
		return
	}

	assert.Len(t, pipelines.pipelines, 3)

	scope := NewScope(pipelines)
	_, err = pipelines.Execute(context.Background(), scope, "root-pipeline", "child-pipeline", "grandchild-pipeline")
	assert.NoError(t, err)
}

func TestLoadReturnsErrorForInvalidNestedYAML(t *testing.T) {
	t.Parallel()

	fileSystem := fstest.MapFS{
		"nested/invalid.yaml": {Data: []byte("name: invalid-pipeline\nsteps: [\n")},
	}

	_, err := Load(fileSystem)
	assert.Error(t, err)
}

func TestLoadReturnsErrorWhenPipelineNameIsMissing(t *testing.T) {
	t.Parallel()

	fileSystem := fstest.MapFS{
		"legacy-id.yaml": {Data: []byte("id: legacy-pipeline\nsteps: []\n")},
	}

	_, err := Load(fileSystem)
	assert.Error(t, err)
}

func TestPipelineNamespaces(t *testing.T) {
	t.Parallel()

	t.Run("isolates when pipeline id is set", func(t *testing.T) {
		t.Parallel()

		pipelines := Pipelines{
			pipelines: map[string]Pipeline{
				"main": {
					Name: "main",
					Steps: []Step{
						{
							Type: "pipeline",
							Params: map[string]any{
								"id":   "alpha-run",
								"uses": "alpha",
							},
						},
						{
							Type: "pipeline",
							Params: map[string]any{
								"id":   "beta-run",
								"uses": "beta",
							},
						},
					},
				},
				"alpha": {
					Name: "alpha",
					Steps: []Step{
						{
							ID:   "setup",
							Type: "set",
							Params: map[string]any{
								"value": 1,
							},
						},
						{
							ID:   "copy",
							Type: "set",
							Params: map[string]any{
								"value": `{{ variableGet . "setup" "value" }}`,
							},
						},
					},
				},
				"beta": {
					Name: "beta",
					Steps: []Step{
						{
							ID:   "setup",
							Type: "set",
							Params: map[string]any{
								"value": 2,
							},
						},
					},
				},
			},
		}

		scope := NewScope(pipelines)

		result, err := pipelines.Execute(context.Background(), scope, "main")
		if !assert.NoError(t, err) {
			return
		}

		alphaSetup, err := result.Variable("alpha-run.setup")
		if !assert.NoError(t, err) {
			return
		}

		alphaSetupMap, ok := alphaSetup.(map[string]any)
		if !assert.True(t, ok) {
			return
		}

		assert.Equal(t, 1, alphaSetupMap["value"])

		alphaCopy, err := result.Variable("alpha-run.copy")
		if !assert.NoError(t, err) {
			return
		}

		alphaCopyMap, ok := alphaCopy.(map[string]any)
		if !assert.True(t, ok) {
			return
		}

		assert.Equal(t, "1", alphaCopyMap["value"])

		betaSetup, err := result.Variable("beta-run.setup")
		if !assert.NoError(t, err) {
			return
		}

		betaSetupMap, ok := betaSetup.(map[string]any)
		if !assert.True(t, ok) {
			return
		}

		assert.Equal(t, 2, betaSetupMap["value"])
	})

	t.Run("does not isolate when pipeline id is omitted", func(t *testing.T) {
		t.Parallel()

		pipelines := Pipelines{
			pipelines: map[string]Pipeline{
				"main": {
					Name: "main",
					Steps: []Step{
						{
							Type: "pipeline",
							Params: map[string]any{
								"uses": "alpha",
							},
						},
						{
							Type: "pipeline",
							Params: map[string]any{
								"uses": "beta",
							},
						},
					},
				},
				"alpha": {
					Name: "alpha",
					Steps: []Step{{
						ID:   "setup",
						Type: "set",
						Params: map[string]any{
							"value": 1,
						},
					}},
				},
				"beta": {
					Name: "beta",
					Steps: []Step{{
						ID:   "setup",
						Type: "set",
						Params: map[string]any{
							"value": 2,
						},
					}},
				},
			},
		}

		scope := NewScope(pipelines)

		result, err := pipelines.Execute(context.Background(), scope, "main")
		if !assert.NoError(t, err) {
			return
		}

		setup, err := result.Variable("setup")
		if !assert.NoError(t, err) {
			return
		}

		setupMap, ok := setup.(map[string]any)
		if !assert.True(t, ok) {
			return
		}

		assert.Equal(t, 2, setupMap["value"])
	})

	t.Run("uses resolves by name not id", func(t *testing.T) {
		t.Parallel()

		pipelines := Pipelines{
			pipelines: map[string]Pipeline{
				"main": {
					Name: "main",
					Steps: []Step{
						{
							Type: "pipeline",
							Params: map[string]any{
								"id":   "alpha-run",
								"uses": "alpha",
							},
						},
					},
				},
				"alpha": {
					Name: "alpha",
					ID:   "alpha-instance-id",
					Steps: []Step{{
						ID:   "setup",
						Type: "set",
						Params: map[string]any{
							"value": 42,
						},
					}},
				},
			},
		}

		scope := NewScope(pipelines)

		result, err := pipelines.Execute(context.Background(), scope, "main")
		if !assert.NoError(t, err) {
			return
		}

		_, err = result.Variable("alpha-run.alpha-instance-id.setup")
		if !assert.NoError(t, err) {
			return
		}

		_, err = result.Variable("alpha-run.alpha.setup")
		assert.Error(t, err)
	})
}
