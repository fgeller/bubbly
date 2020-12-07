package bubbly

import (
	"net/http"
	"testing"

	"github.com/rs/zerolog"

	"github.com/stretchr/testify/assert"

	"github.com/verifa/bubbly/env"
	"gopkg.in/h2non/gock.v1"
)

// TestApply simply validates that a given directory containing bubbly
// configuration including a pipeline_run will result in a POST of data to
// the bubbly server.
// See client/load_test.go for actual evaluation of the loading using
// the gofight package.
func TestApply(t *testing.T) {

	defer gock.Off()

	// Subtest
	t.Run("sonarqube", func(t *testing.T) {
		// Create a new server route for mocking a Bubbly server response
		bCtx := env.NewBubblyContext()
		gock.New(bCtx.ServerConfig.HostURL()).
			Post("/alpha1/upload").
			Reply(http.StatusOK).
			JSON(map[string]interface{}{"status": "uploaded"})

		bCtx.UpdateLogLevel(zerolog.DebugLevel)

		err := Apply(bCtx, "./testdata/sonarqube")

		assert.NoError(t, err, "Failed to apply resource")
	})
}

func TestApplyTaskRun(t *testing.T) {
	// Subtest
	t.Run("task_run_sonarqube_extract", func(t *testing.T) {
		bCtx := env.NewBubblyContext()
		bCtx.UpdateLogLevel(zerolog.DebugLevel)

		err := Apply(bCtx, "./testdata/resources/v1/taskrun/extract_sonarqube.bubbly")

		assert.NoError(t, err, "Failed to apply resource")
	})
}

func TestApplyQuery(t *testing.T) {
	// Subtest
	t.Run("apply basic query", func(t *testing.T) {
		bCtx := env.NewBubblyContext()
		bCtx.UpdateLogLevel(zerolog.DebugLevel)

		gock.New(bCtx.ServerConfig.HostURL()).
			Post("/api/graphql").
			Reply(http.StatusOK).
			JSON(`{"data":{"test_run":{"name":"run 1","repo_version_id":0,"test_set":[{"name":"set 1","test_case":[{"ID":1,"name":"case 1.1","status":"PASS","test_set_id":1},{"ID":2,"name":"case 1.2","status":"PASS","test_set_id":1},{"ID":3,"name":"case 1.3","status":"FAIL","test_set_id":1}]},{"name":"set 2","test_case":[{"ID":4,"name":"case 2.1","status":"FAIL","test_set_id":2},{"ID":5,"name":"case 2.2","status":"FAIL","test_set_id":2}]}]}}}`)

		err := ApplyQueries(bCtx, "./testdata/resources/v1/query/query.bubbly")

		assert.NoError(t, err, "Failed to apply resource")
	})
}
