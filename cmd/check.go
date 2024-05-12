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
		appConf, err := config.LoadAppConfig(homeDir)
		libutils.ExitIfErr(err, "unable to load app config")

		usersConf, err := config.LoadUsersConfig(homeDir)
		libutils.ExitIfErr(err, "unable to load users config")

		chainsConf, err := config.LoadChainsConfig(homeDir)
		libutils.ExitIfErr(err, "unable to load chains config")

		// Output some options to console
		appConf.PrintOptions()

		// Perform validation
		err = appConf.Validate()
		libutils.ExitIfErr(err, "bad app config")

		err = usersConf.ToUserRecords().Validate()
		libutils.ExitIfErr(err, "bad users config")

		err = chainsConf.Validate(usersConf)
		libutils.ExitIfErr(err, "bad chains config")

		fmt.Println("Validation completed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
