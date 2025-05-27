package main

import (
	"context"
	httplib "net/http"
	"os"
	"strings"

	"github.com/crowleyfelix/go-pipeline/pkg/http"
	"github.com/crowleyfelix/go-pipeline/pkg/log"
	"github.com/crowleyfelix/go-pipeline/pkg/pipeline"
	"github.com/samber/lo"
)

var (
	pipelineDir = os.Getenv("PIPELINE_DIR")
	pipelineIDs = strings.Split(os.Getenv("PIPELINE_IDS"), ",")
)

func main() {
	log.SetUp(log.Standard{})
	http.RegisterProcessor(httplib.DefaultClient)

	pipelines := lo.Must(pipeline.Load(os.DirFS(pipelineDir)))

	ctx := pipeline.NewContext(context.Background(), pipelines)

	_, err := pipelines.Execute(ctx, pipelineIDs...)
	if err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
