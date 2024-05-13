package cmd

import (
	"fmt"
	libutils "github.com/EscanBE/go-lib/utils"
	cmdutils "github.com/bcdevtools/validator-health-check/cmd/utils"
	"github.com/bcdevtools/validator-health-check/constants"
	"github.com/spf13/cobra"
	"os"
	"path"
)

// initCmd represents the init command, to be used to generate home directory with configuration file named `config.yaml`
var initCmd = &cobra.Command{
	Use:   "init",
	Short: fmt.Sprintf("Init home directory & configuration files for %s at %s", constants.APP_NAME, cmdutils.GetDefaultHomeDirectory()),
	Run: func(cmd *cobra.Command, args []string) {
		_, err := os.Stat(homeDir)

		if err != nil && os.IsNotExist(err) {
			fmt.Printf("Require home dir '%s' does not exists, going to create new home dir\n", homeDir)
			err := os.Mkdir(homeDir, 0o750)
			libutils.ExitIfErr(err, fmt.Sprintf("Unable to create home dir %s", homeDir))
		} else if err != nil {
			cobra.CheckErr(err)
		}

		cfgFile := path.Join(homeDir, constants.CONFIG_FILE_NAME)
		usersFile := path.Join(homeDir, constants.USERS_FILE_NAME)

		writeYamlFile("Config", cfgFile, // trailing style: 2 spaces
			fmt.Sprintf(`# %s's configuration file
general:
  hot-reload: 5m
  health-check: 10m
worker:
  health-check-count: 5
logging:
  level: info # debug || info || error
  format: json
`, constants.APP_NAME))

		writeYamlFile("User", usersFile, // trailing style: 2 spaces
			fmt.Sprintf(`# %s's users configuration file
users:
  username1:
    root: false
    telegram:
      username: "UserName1"
      id: -1
      token: "token"
`, constants.APP_NAME))

		writeYamlFile("Chain", path.Join(homeDir, fmt.Sprintf("%stest.%s", constants.CHAIN_FILE_NAME_PREFIX, constants.CONFIG_TYPE)), // trailing style: 2 spaces
			`
chain-name: "test"
chain-id: "test"
disable: true
priority: false
rpc: []
validators:
  valoper1:
    watchers: []
    health-check-rpc: ""
`)

		fmt.Println("Initialized successfully!")
	},
}

func writeYamlFile(identity, filePath, content string) {
	_, err := os.Stat(filePath)

	if err == nil {
		fmt.Printf("%s file already exists, skip writing %s\n", identity, filePath)
		return
	}

	if !os.IsNotExist(err) {
		panic(err)
	}
	fmt.Printf("%s file does not exists, going to create new file with permission %s\n", filePath, constants.FILE_PERMISSION_STR)
	file, err := os.Create(filePath)
	libutils.ExitIfErr(err, fmt.Sprintf("Unable to create %s file %s", identity, filePath))
	err = file.Chmod(constants.FILE_PERMISSION)
	libutils.ExitIfErr(err, fmt.Sprintf("Unable to set permission for new %s file %s to %s", identity, filePath, constants.FILE_PERMISSION_STR))
	_, err = file.WriteString(fmt.Sprintf(content))
	if err != nil {
		fmt.Printf("ERR: unable to write content for new %s file %s\n", identity, filePath)
		panic(err)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
