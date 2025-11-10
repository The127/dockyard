package args

import "flag"

var configFilePath string
var environment string

func ConfigFilePath() string {
	return configFilePath
}

func IsProduction() bool {
	return environment == "PRODUCTION"
}

func Init() {
	flag.StringVar(&configFilePath, "config", "", "The path for the config file.")
	flag.StringVar(&environment, "environment", "PRODUCTION", "The environment that this application is running in (can be PRODUCTION or DEVELOPMENT).")
	flag.Parse()
}
