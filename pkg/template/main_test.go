package template_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/fogmachine/pkg/template"
)

func TestParseConfig(t *testing.T) {
	input := template.Input{
		TemplatePath:  "testdata/s3.yaml",
		ParameterPath: "testdata/s3-values.json",
	}

	got, err := template.Read(input)

	if err != nil {
		t.Fatal(err)
	}

	wantTemplate, err := os.ReadFile("testdata/s3.yaml")

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(wantTemplate, got.Template) {
		t.Fatalf("Got %s but expected %s", wantTemplate, got.Template)
	}

	wantParameters := map[string]string{
		"BucketName":       "md-test-cf-1234",
		"DevBucketName":    "md-test-dev-cf-1234",
		"connections.auth": "test",
	}

	for _, parameter := range got.Parameters {
		got := *parameter.ParameterValue
		want := wantParameters[*parameter.ParameterKey]
		if got != want {
			t.Fatalf("Got %s but expected %s", got, want)
		}
	}
}
