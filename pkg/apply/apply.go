package apply

import (
	"context"

	"github.com/massdriver-cloud/fogmachine/pkg/client"
	"github.com/massdriver-cloud/fogmachine/pkg/template"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func CfApply(cmd *cobra.Command, args []string) {

	ctx := context.Background()
	packageName, err := cmd.Flags().GetString("package-name")
	if err != nil {
		log.Fatal().Err(err)
	}

	region, err := cmd.Flags().GetString("region")
	if err != nil {
		log.Fatal().Err(err)
	}

	client, err := client.NewCloudformationClient(packageName, region, ctx)
	if err != nil {
		log.Fatal().Err(err)
	}

	templatePath, err := cmd.Flags().GetString("template-path")
	if err != nil {
		log.Fatal().Err(err)
	}

	parameterPath, err := cmd.Flags().GetString("parameter-path")
	if err != nil {
		log.Fatal().Err(err)
	}

	template, err := template.Read(template.Input{
		TemplatePath:  templatePath,
		ParameterPath: parameterPath,
	})
	if err != nil {
		log.Fatal().Err(err)
		return
	}

	err = client.CreateChangeset(template.Template, template.Parameters, ctx)

	if err != nil {
		log.Fatal().Err(err)
		return
	}

	err = client.ExecuteChangeSet(ctx)

	if err != nil {
		log.Fatal().Err(err)
		return
	}
}
