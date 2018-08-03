package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	//"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocGeneration(t *testing.T) {
	// create a temporary directory to hold generated doc files
	outputDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	defer os.RemoveAll(outputDir)

	// tests run from the same working directory as the test file, so two directories up is our goflow root
	err = generateDocs("../../", outputDir)
	require.NoError(t, err)

	// check each rendered template for changes
	for _, template := range templates {
		existing, err := ioutil.ReadFile("../../docs/md/" + template.path)
		require.NoError(t, err)

		generated, err := ioutil.ReadFile(path.Join(outputDir, "md", template.path))
		require.NoError(t, err)

		// if the docs we just generated don't match the existing ones, someone needs to run docgen
		require.Equal(t, string(existing), string(generated), "changes have been made that require re-running docgen (go install github.com/nyaruka/goflow/cmd/docgen; docgen)")
	}
}
