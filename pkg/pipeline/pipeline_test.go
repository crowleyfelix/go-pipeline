package pipeline

import (
	"context"
	"testing"
	"testing/fstest"
)

func TestLoadLoadsPipelinesFromNestedFolders(t *testing.T) {
	t.Parallel()

	fileSystem := fstest.MapFS{
		"root.yaml":                  {Data: []byte("id: root-pipeline\nsteps: []\n")},
		"nested/child.yaml":          {Data: []byte("id: child-pipeline\ndescription: child\nsteps: []\n")},
		"nested/deep/grandchild.yml": {Data: []byte("id: grandchild-pipeline\nsteps: []\n")},
		"nested/ignored.txt":         {Data: []byte("this should be ignored")},
	}

	pipelines, err := Load(fileSystem)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(pipelines.pipelines) != 3 {
		t.Fatalf("expected 3 pipelines, got %d", len(pipelines.pipelines))
	}

	scope := NewScope(pipelines)
	_, err = pipelines.Execute(context.Background(), scope, "root-pipeline", "child-pipeline", "grandchild-pipeline")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestLoadReturnsErrorForInvalidNestedYAML(t *testing.T) {
	t.Parallel()

	fileSystem := fstest.MapFS{
		"nested/invalid.yaml": {Data: []byte("id: invalid-pipeline\nsteps: [\n")},
	}

	_, err := Load(fileSystem)
	if err == nil {
		t.Fatal("expected Load() to return an error for invalid YAML")
	}
}
