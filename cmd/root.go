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

	rootCmd.AddCommand(
		ApplyCmd(),
	)

	cobra.OnInitialize(initLogging)
	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cmd := NewCmdFogMachine()
	err := cmd.Execute()
	if err != nil {
		log.Fatal().Err(err)
	}
}

func initLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
