package cmd

import (
	"fmt"
	libutils "github.com/EscanBE/go-lib/utils"
	"github.com/bcdevtools/validator-health-check/config"
	"github.com/bcdevtools/validator-health-check/constants"
	"github.com/spf13/cobra"
)

// checkCmd do print and validate the configuration file config.yaml
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: fmt.Sprintf("Print and validate %s's configuration, combine with `--home` flag to specify folder", constants.APP_NAME),
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.LoadConfig(homeDir)
		libutils.ExitIfErr(err, "unable to load configuration")

		// Output some options to console
		conf.PrintOptions()

		// Perform validation
		err = conf.Validate()
		libutils.ExitIfErr(err, "failed to validate configuration")

		fmt.Println("Validation completed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
