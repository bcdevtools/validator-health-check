package utils

import (
	libutils "github.com/EscanBE/go-lib/utils"
	"github.com/bcdevtools/validator-health-check/constants"
	"os"
	"path"
)

// GetDefaultHomeDirectory returns default home directory, typically `~/.binaryName`
func GetDefaultHomeDirectory() string {
	home, err := os.UserHomeDir()
	libutils.ExitIfErr(err, "failed to use home directory")
	return path.Join(home, constants.DEFAULT_HOME)
}
