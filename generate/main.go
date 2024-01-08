//go:build ignore

package main

import (
	"fmt"
	"log"
	"path"
	"time"

	"github.com/dramich/aws-mocker/pkg/mock"
	"github.com/dramich/aws-mocker/pkg/writer"
)

func main() {
	fmt.Println("Generating mocks")
	t := time.Now()
	packageName := "mock"
	mockOpts := mock.Options{
		BaseDir:        ".",
		SearchPackages: "github.com/massdriver-cloud/fogmachine/pkg/client",
		PackageName:    packageName,
		ClientDefault:  true,
		Writer:         writer.New(path.Join("../testing/mock", packageName+".go")),
	}

	if err := mock.Run(&mockOpts); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Completed in", time.Since(t))
}
