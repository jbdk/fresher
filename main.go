package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/c9845/fresh/config"
	"github.com/c9845/fresh/runner"
	"github.com/c9845/fresh/version"
)

func main() {
	//Handle flags.
	configFilePath := flag.String("config", "./"+config.DefaultConfigFileName, "Full path to the configuration file.")
	showVersion := flag.Bool("version", false, "Shows the version of the app.")
	flag.Parse()

	//If user just wants to see app version, print it and exit.
	//Not using log.Println() so that a timestamp isn't printed.
	if *showVersion {
		fmt.Printf("Version: %s (Released: %s)\n", version.V, version.ReleaseDate)
		os.Exit(0)
		return
	}

	//Starting messages.
	//Always show version number when starting for diagnostics.
	log.Println("Starting Fresh...")
	log.Println("Version:", version.V)

	//Read and parse the config file at the provided path. The config file provides
	//runtime configuration of the app and contains settings that are rarely modified.
	// - If the --config flag is blank, the default value, a default config is used.
	// - If the --config flag has a path set, look for a file at the provided path.
	//    - If a file is found, parse it as config file and handle any errors.
	//    - If a file cannot be found, create a default config and save it to the path provided.
	err := config.Read(*configFilePath, *printConfig)
	if err != nil {
		log.Fatalln("Could not parse config file.", errors.Unwrap(err))
		return
	}

	if *configPath != "" {
		if _, err := os.Stat(*configPath); err != nil {
			fmt.Printf("Can't find config file `%s`\n", *configPath)
			os.Exit(1)
		} else {
			os.Setenv("RUNNER_CONFIG_PATH", *configPath)
		}
	}

	runner.Start()
}
