package destroy

import (
	"context"

	"github.com/massdriver-cloud/fogmachine/pkg/client"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func Destroy(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	packageName, err := cmd.Flags().GetString("package-name")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	region, err := cmd.Flags().GetString("region")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	timeout, err := cmd.Flags().GetInt("timeout")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	pollInterval, err := cmd.Flags().GetInt("poll-interval")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	client, err := client.NewCloudformationClient(ctx, packageName, region, timeout, pollInterval)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if err := client.ExecuteDestroyStack(ctx); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
