/*
Package config handles configuration of the app. The configuration can simply be
a default, or a config file can be provided. The config data is used for low-level
settings of the app and can be used elsewhere in this app.

The config file is in yaml format for easy readability. However, the file does not
need to end in the yaml extension.

This package must not import any other packages from within this app to prevent
import loops (besides minor utility packages).

---

When adding a new field to the config file:
  - Add the field to the File type below.
  - Determine any default value(s) for the field and set it in newDefaultConfig().
  - Document the field as needed (README, other documentation).
  - Set validation in validate().
*/
package config

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c9845/fresh/version"
	"gopkg.in/yaml.v2"
)

// DefaultConfigFileName is the typical name of the config file.
const DefaultConfigFileName = "fresh.conf"

// File defines the list of configuration fields. The value for each field will be
// set by a default or read from a config file. The config file is typically stored
// in the same directory as the executable.
//
// Don't use uints! If user provided a negative number an ugly error message is
// kicked out. We would rather check for the negative number here and provide a
// nicer error message.
type File struct {
	WorkingDir string `yaml:"WorkingDir"` //path to working directory.
	TempDir    string `yaml:"TempDir"`    //directory off of working directory to store temporary files

	ExtensionsToWatch   []string `yaml:"ExtensionsToWatch"`   //files to watch for changes; .go, .html are most common.
	NoRebuildExtensions []string `yaml:"NoRebuildExtensions"` //files to we watch to restart the binary on, but don't rebuild on.
	IgnoredDirs         []string `yaml:"IgnoredDirs"`         //directories to ignore files in, files won't be watches for changes; .git, node_modules, temp, etc.

	BuildDelayMilliseconds int64 //milliseconds to wait until triggering rebuild, to prevent rebuilding on "save" being trigger multiple times very quickly.

}

// parsedConfig is the data parsed from the config file. This data is stored so that
// we don't need to reparse the config file each time we need a piece of data from it.
// This is not exported so that changes cannot be made to the parsed data as easily.
// Use the Data() func to get the data for use elsewhere.
var parsedConfig File

// Stuff used for validation.
const ()

// Errors.
var (
	//ErrNoFilePathGiven is returned when trying to parse the config file but no
	//file path was given.
	ErrNoFilePathGiven = errors.New("config: no file path given")
)

// newDefaultConfig returns a File with default values set for each field.
func newDefaultConfig() (f File, err error) {
	//Get path the command is being run in to use as default base path
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return
	}

	f = File{}
	return
}

// Read handles reading and parsing the config file at the provided path. The parsed
// data is sanitized and validated. The print argument is used to print the config
// as it was read/parsed and as it was understood after sanitizing, validating, and
// handling default values.
//
// The parsed configuration is stored in a local variable for access with the
// Data() func. This is done so that the config file doesn't need to be reparsed
// each time we want to get data from it.
func Read(path string, print bool) (err error) {
	// log.Println("Provided config file path:", path, print)

	//Get absolute path to config file. Absolute path is nicer for logging, diagnostics,
	//and can prevent future issues.
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return
	}
	path = absolutePath

	//Handle path to config file.
	// - If the path is blank, we just use the default config. An empty path
	//   should not ever happen since the flag that provides the path has a
	//   default set. However, we still need this since if the path is empty
	//   we cannot save the default config to a file (we don't know where to
	//   save it).
	// - If a path is provided, check that a file exists at it. If a file does
	//   not exist, create a default config at the given path.
	// - If a file at the path does exist, parse it as a config file.
	if strings.TrimSpace(path) == "" {
		log.Println("Using default config; path to config file not provided.")

		//Get default config.
		cfg, innerErr := newDefaultConfig()
		if innerErr != nil {
			return innerErr
		}

		//We don't get a random private key encryption key since if the app is
		//started over and over without a config file path, the encryption key
		//will be different each time and thus the private keys won't be usable.

		//Save the config to this package for use elsewhere in the app.
		parsedConfig = cfg

	} else if _, err = os.Stat(path); os.IsNotExist(err) {
		log.Println("WARNING! (config) Creating default config at:", path)

		//Get default config.
		cfg, innerErr := newDefaultConfig()
		if innerErr != nil {
			return innerErr
		}

		//Save the config to this package for use elsewhere in the app.
		parsedConfig = cfg

		//Save the config to a file.
		innerErr = cfg.write(path)
		if innerErr != nil {
			return
		}

		//Unset the os.IsNotExist error since we created the file.
		err = nil
	} else {
		log.Println("Using config from file:", path)

		//Read the file at the path.
		f, innerErr := os.ReadFile(path)
		if innerErr != nil {
			return innerErr
		}

		//Parse the file as yaml.
		var cfg File
		innerErr = yaml.Unmarshal(f, &cfg)
		if innerErr != nil {
			return innerErr
		}

		//Print the config, if needed, as it was parsed from the file. This logs
		//out the config fields with the user provided data before any validation.
		if print {
			log.Println("***PRINTING CONFIG AS PARSED FROM FILE***")
			cfg.print(path)
		}

		//Validate & sanitize the data since it could have been edited by a human.
		innerErr = cfg.validate()
		if innerErr != nil {
			return innerErr
		}

		//Save the config to this package for use elsewhere in the app.
		parsedConfig = cfg
	}

	//Print the config, if needed, as it was sanitized and validated. This logs out
	//the config as it was understood by the app and some changes may have been made
	//(for example, user provided an invalid value for a field and a default value
	//was used instead). This also prints out the config if it was created or if the
	//config path was blank and a default config was used instead.
	//Always exit at this point since printing config is just for diagnostics.
	if print {
		log.Println("***PRINTING CONFIG AS UNDERSTOOD BY APP***")
		parsedConfig.print(path)
		os.Exit(0)
		return
	}

	return
}

// write writes a config to a file at the provided path.
func (conf *File) write(path string) (err error) {
	//Marshal to yaml.
	y, err := yaml.Marshal(conf)
	if err != nil {
		return
	}

	//Create the file.
	file, err := os.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	//Add some comments to config file so a human knows it was generated, not
	//written by a human.
	file.WriteString("#Generated config file for Fresh.\n")
	file.WriteString("#Generated at: " + time.Now().UTC().Format(time.RFC3339) + "\n")
	file.WriteString("#Version: " + version.V + "\n")
	file.WriteString("#This file is in YAML format.\n")
	file.WriteString("\n")
	file.WriteString("#***Do not delete this file!***\n")
	file.WriteString("\n")

	//Write config to file.
	_, err = file.Write(y)
	return
}

// validate handles sanitizing and validation of a config file's data.
func (conf *File) validate() (err error) {
	//Get defaults to use for cases when user provided invalid input.
	defaults, err := newDefaultConfig()
	if err != nil {
		return
	}

	return
}

// returnErrRequired is a helper func to create a standardized error message for when
// a required field doesn't have a value.  This helps clean up validate() and keep
// error messages for required fields the same.
func returnErrRequired(field string) error {
	return errors.New("config: A value for the field " + field + " was not provided but is required")
}

// print logs out the configuration file. This is used for diagnostic purposes.
// This will show all fields from the File struct, even fields that the provided
// config file omitted (except nonPublishedFields).
func (conf File) print(path string) {
	//Full path to the config file, so if file is in same directory as the
	//executable and -config flag was not provided we still get the complete path.
	pathAbs, _ := filepath.Abs(path)

	log.Println("Path to config file (flag):", path)
	log.Println("Path to config file (absolute):", pathAbs)
}

// Data returns the full parsed config file data
// This is used in other packages to use config file setting data.
func Data() File {
	return parsedConfig
}
