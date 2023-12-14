package cmd

import (
	"github.com/massdriver-cloud/fogmachine/pkg/destroy"
	"github.com/spf13/cobra"
)

func DestroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy a Cloudformation stack",
		Long:  "Destroy a Cloudformation stack",
		Run:   destroy.Destroy,
	}

	cmd.Flags().StringP("package-name", "p", "", "Package name")
	_ = cmd.MarkFlagRequired("package-name")
	cmd.Flags().StringP("region", "r", "", "AWS region")
	_ = cmd.MarkFlagRequired("region")
	cmd.Flags().Int("timeout", 600, "time in seconds to wait for resources to finish, this does not cancel the cloud formation run")
	cmd.Flags().Int("poll-interval", 3, "time in seconds between each poll of the AWS api for updates")

	return cmd
}
