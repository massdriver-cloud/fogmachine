package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/fogmachine/pkg/version"
	"github.com/spf13/cobra"
)

func VersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Version of the fogmachine CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("fogmachine version %s, sha %s\n", version.Version(), version.GitSHA())
		},
	}
	return versionCmd
}
