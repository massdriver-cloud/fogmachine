/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/spf13/cobra"
)

func NewCmdFogMachine() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "fogmachine",
		Short: "CLI for running AWS Cloudformation in CI",
		Long:  `Get detailed status data about running Cloudformation actions with FogMachine.`,
	}

	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Set the log level [debug, info, warn, error]")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logLevel, err := cmd.Flags().GetString("log-level")
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		initLogging(logLevel)
	}

	rootCmd.AddCommand(
		ApplyCmd(),
		DestroyCmd(),
	)

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cmd := NewCmdFogMachine()
	err := cmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

func initLogging(level string) {
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		log.Warn().Msg("Unable to parse log level, defaulting to info")
		l = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(l)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
