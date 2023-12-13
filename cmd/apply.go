package cmd

import (
	"github.com/massdriver-cloud/fogmachine/pkg/apply"
	"github.com/spf13/cobra"
)

func ApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Create or update a Cloudformation stack",
		Long:  "Create or update a Cloudformation stack",
		Run:   apply.CfApply,
	}

	cmd.Flags().StringP("package-name", "p", "", "Package name")
	_ = cmd.MarkFlagRequired("package-name")
	cmd.Flags().StringP("region", "r", "", "AWS region")
	_ = cmd.MarkFlagRequired("region")
	cmd.Flags().StringP("template-path", "", "", "Path to CloudFormation template")
	_ = cmd.MarkFlagRequired("template-path")
	cmd.Flags().StringP("parameter-path", "", "", "Path to CloudFormation input vars")
	_ = cmd.MarkFlagRequired("parameter-path")
	cmd.Flags().Int("timeout", 600, "time in seconds to wait for resources to finish, this does not cancel the cloud formation run")
	cmd.Flags().Int("poll-interval", 3, "time in seconds between each poll of the AWS api for updates")

	return cmd
}
