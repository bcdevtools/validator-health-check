package constants

// Define constants in this file

//goland:noinspection GoSnakeCaseUsage
const (
	APP_NAME    = "Validator Health-check daemon"
	APP_DESC    = "Health-check validators"
	BINARY_NAME = "hcvald"

	// Do not change bellow

	DEFAULT_HOME     = "." + BINARY_NAME
	CONFIG_FILE_NAME = "config." + CONFIG_TYPE
	USERS_FILE_NAME  = "users." + CONFIG_TYPE
	CONFIG_TYPE      = "yaml"
)

//goland:noinspection GoSnakeCaseUsage
const (
	FLAG_HOME = "home"
)

//goland:noinspection GoSnakeCaseUsage
const (
	FILE_PERMISSION     = 0o600
	FILE_PERMISSION_STR = "600"
)
