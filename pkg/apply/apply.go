package apply

import (
	"context"

	"github.com/massdriver-cloud/fogmachine/pkg/client"
	"github.com/massdriver-cloud/fogmachine/pkg/template"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func CfApply(cmd *cobra.Command, _ []string) {
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

	templatePath, err := cmd.Flags().GetString("template-path")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	parameterPath, err := cmd.Flags().GetString("parameter-path")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	template, err := template.Read(template.Input{
		TemplatePath:  templatePath,
		ParameterPath: parameterPath,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if err = client.CreateChangeset(ctx, template.Template, template.Parameters); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if err = client.ExecuteChangeSet(ctx); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
