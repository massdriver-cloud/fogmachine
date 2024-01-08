/*
Copyright © 2023 Massdriver
*/
package main

import (
	_ "github.com/dramich/aws-mocker/pkg/mock"
	"github.com/massdriver-cloud/fogmachine/cmd"
)

func main() {
	cmd.Execute()
}
