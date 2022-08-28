/*
Package config handles configuration of the app.

A config is handled one of three ways:
  - If a file exists at the path, it is attempted to be parsed as a valid config file.
  - If a file does not exists at the path, the built-in defaults are used after a
    warning is shown about the missing file.
  - If the path is blank, the default config is used.

The config file is in yaml format for easy readability.

This package must not import any other packages from within this repo to prevent
import loops (besides minor utility packages) since the config package is most likely
read by nearly all other packages in this repo.

When adding a new field to the config file:
  - Add the field to the File type below.
  - Determine any default value(s) for the field and set it in newDefaultConfig().
  - Set validation in validate().
  - Document the field as needed (README, other documentation).
*/
package config

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c9845/fresher/version"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"
)

// DefaultConfigFileName is the typical name of the config file.
const DefaultConfigFileName = "fresher.conf"

// File defines the list of configuration fields. The value for each field will be
// set by a default or read from a config file. The config file is typically stored
// in the same directory as the executable.
//
// Don't use uints! If user provided a negative number an ugly error message is
// kicked out. We would rather check for the negative number here and provide a
// nicer error message.
//
// Struct tags are needed for working with yaml.v2 package, otherwise the package
// expects fields to start with lower case characters. However, if we lower cased all
// the struct field names, then we wouldn't be able to access those fields in other
// packages.
type File struct {
	WorkingDir string `yaml:"WorkingDir"` //path to working directory.
	TempDir    string `yaml:"TempDir"`    //directory off of working directory to store temporary files

	ExtensionsToWatch   []string `yaml:"ExtensionsToWatch"`   //files to watch for changes; .go, .html are most common.
	NoRebuildExtensions []string `yaml:"NoRebuildExtensions"` //files to we restart the binary on, but don't rebuild the binary.
	DirectoriesToIgnore []string `yaml:"DirectoriesToIgnore"` //directories to ignore files in, files won't be watches for changes; .git, node_modules, temp, etc.

	BuildDelayMilliseconds int64    //milliseconds to wait until triggering rebuild, to prevent rebuilding on "save" being trigger multiple times very quickly.
	BuildName              string   `yaml:"BuildName"`        //name of binary when built
	BuildLogFilename       string   `yaml:"BuildLogFilename"` //name of file in TempDir where build errors will be logged to
	GoBuildTags            []string `yaml:"GoBuildTags"`      //anything provided in go build tags (go build -tags asdf).

	VerboseLogging bool `yaml:"VerboseLogging"` //log more diagnostic info from fresher
}

// parsedConfig is the data parsed from the config file. This data is stored so that
// we don't need to reparse the config file each time we need a piece of data from it.
// This is not exported so that changes cannot be made to the parsed data as easily.
// Use the Data() func to get the data for use elsewhere.
var parsedConfig File

// newDefaultConfig returns a File with default values set for each field.
func newDefaultConfig() (f File) {
	//Base working directory is relative to where fresher has been called from. This
	//is done, instead of using absolute path, so that in case the fresher.conf config
	//file is saved to version control, no identifying information from an absolute
	//path is saved. I.e.: if using absolute paths, the path to the working directory
	//may be something like /users/johnsmith/.../fresher.conf, leaking the user's name.
	workingDir := "."

	f = File{
		WorkingDir:             workingDir,
		TempDir:                filepath.Join(workingDir, "tmp"),
		ExtensionsToWatch:      []string{".go", ".html"},
		NoRebuildExtensions:    []string{".html"},
		DirectoriesToIgnore:    []string{"tmp", "node_modules", ".git", ".vscode"},
		BuildDelayMilliseconds: 300,
		BuildName:              "fresher-build",
		BuildLogFilename:       "fresher-build-errors.log",
		VerboseLogging:         false,
	}
	return
}

// CreateDefaultConfig creates a config file in the current directory with the default
// configuration. This is mostly used to create a template config file to modify.
// Used/called by the -init flag.
func CreateDefaultConfig() (err error) {
	//Get default config.
	cfg := newDefaultConfig()

	//Get path to save config to.
	path := filepath.Join(".", DefaultConfigFileName)

	//Check if a config file already exists at this path to prevent overwriting it.
	_, err = os.Stat(path)
	if err == nil {
		log.Printf("WARNING! (config) Config file already exists at %s, skipping creation. Remove the -init flag.", path)
		return nil
	}

	//Save the config to a file.
	err = cfg.write(path)
	if err != nil {
		return
	}

	log.Printf("WARNING! (config) Config file created with defaults at %s, remove the -init flag in the future.", path)
	return
}

// Read handles reading and parsing the config file at the provided path. The parsed
// data is sanitized and validated. The print argument is used to print the config
// as it was read/parsed and as it was understood after sanitizing, validating, and
// handling default values.
//
// If a config file is not found at the given path, a warning is shown and the
// built-in default config is used instead. Use -init to create a default config file.
//
// The parsed configuration is stored in a local variable for access with the
// Data() func. This is done so that the config file doesn't need to be reparsed
// each time we want to get data from it.
func Read(path string, print bool) (err error) {
	// log.Println("Provided config file path:", path, print)

	//Handle path to config file.
	// - If the path is blank, we just use the default config. An empty path
	//   should not ever happen since the flag that provides the path has a
	//   default set.
	// - If a path is provided, check that a file exists at it. If a file does
	//   not exist, show a warning and use the default build-in config.
	// - If a file at the path does exist, parse it as a config file.
	if strings.TrimSpace(path) == "" {
		//Get default config.
		cfg := newDefaultConfig()

		//Save the config to this package for use elsewhere in the app.
		parsedConfig = cfg

	} else if _, err = os.Stat(path); os.IsNotExist(err) {
		log.Printf("WARNING! (config) Config file not found at %s, use -init flag to create it, using built-in defaults.", path)

		//Get default config.
		cfg := newDefaultConfig()

		//Save the config to this package for use elsewhere in the app.
		parsedConfig = cfg

		//Unset the file not found error.
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
	file.WriteString("#Generated config file for Fresher.\n")
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
	defaults := newDefaultConfig()

	//Make sure working directory is set. This should just be "." in most cases since
	//the working directory is the directory where "fresher" is being run.
	conf.WorkingDir = filepath.FromSlash(strings.TrimSpace(conf.WorkingDir))
	if conf.WorkingDir == "" {
		return errors.New("config: WorkingDir not set. Typically this should be set to \".\"")
	}

	//Make sure temp directory is somewhat valid looking.
	conf.TempDir = filepath.FromSlash(strings.TrimSpace(conf.TempDir))
	if conf.TempDir == "" {
		conf.TempDir = defaults.TempDir
		log.Println("WARNING! (config) TempDir not provided, defaulting to " + conf.TempDir + ".")
	}

	//Sanitize each provided extension. This catches blanks and missing leading
	//periods. This also catches duplicates.
	validExtensionsToWatch := []string{}
	for _, extention := range conf.ExtensionsToWatch {
		extention = strings.TrimSpace(extention)

		if !strings.HasPrefix(extention, ".") {
			log.Println("WARNING! (config) ExtensionsToWatch " + extention + " missing leading period, added.")
		}

		if slices.Contains(validExtensionsToWatch, extention) {
			log.Println("WARNING! (config) ExtensionsToWatch duplicate " + extention + ", ignored.")
			continue
		}

		validExtensionsToWatch = append(validExtensionsToWatch, extention)
	}
	conf.ExtensionsToWatch = validExtensionsToWatch

	validNoRebuildExtensionss := []string{}
	for _, extention := range conf.NoRebuildExtensions {
		extention = strings.TrimSpace(extention)

		if !strings.HasPrefix(extention, ".") {
			log.Println("WARNING! (config) NoRebuildExtensions " + extention + " missing leading period, added.")
		}

		if slices.Contains(validNoRebuildExtensionss, extention) {
			log.Println("WARNING! (config) NoRebuildExtensions duplicate " + extention + ", ignored.")
			continue
		}

		validNoRebuildExtensionss = append(validNoRebuildExtensionss, extention)
	}
	conf.NoRebuildExtensions = validNoRebuildExtensionss

	//Make sure at least one extension to watch was given. If no extensions were given,
	//then we don't know what files to watch for changes!
	if len(conf.ExtensionsToWatch) == 0 {
		conf.ExtensionsToWatch = defaults.ExtensionsToWatch
		log.Printf("WARNING! (config) ExtensionsToWatch not provided, defaulting to %s .", conf.ExtensionsToWatch)
	}

	//Make sure any directories to ignore actually exist off the working dir.
	validDirectoriesToIgnore := []string{}
	for _, dir := range conf.DirectoriesToIgnore {
		//Sanitize.
		dir = strings.TrimSpace(dir)
		dir = filepath.Clean(dir)

		//We don't check if a directory actually exists. Who cares if a directory
		//listed in the config file doesn't actually exists in the repo.

		if slices.Contains(validDirectoriesToIgnore, dir) {
			log.Println("WARNING! (config) Duplicate directory " + dir + " in DirectoriesToIgnore.")
			continue
		}

		validDirectoriesToIgnore = append(validDirectoriesToIgnore, dir)
	}
	conf.DirectoriesToIgnore = validDirectoriesToIgnore

	//Validate some other stuff.
	if conf.BuildDelayMilliseconds < 0 {
		conf.BuildDelayMilliseconds = defaults.BuildDelayMilliseconds
		log.Printf("WARNING! (config) BuildDelayMilliseconds must be greater then 0, defaulting to %d.", conf.BuildDelayMilliseconds)
	}

	if strings.TrimSpace(conf.BuildName) == "" {
		conf.BuildName = defaults.BuildName
		log.Println("WARNING! (config) BuildName was not given, defaulting to " + conf.BuildName + ".")
	}

	if strings.TrimSpace(conf.BuildLogFilename) == "" {
		conf.BuildLogFilename = defaults.BuildLogFilename
		log.Println("WARNING! (config) BuildLogFilename was not given, defaulting to " + conf.BuildLogFilename + ".")
	}

	return
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

// IsTempDir returns true if the given path represents the same directory as TempDir.
// We use absolute paths here since we want to be certain if the path given matches
// the same underlying directory as given in TempDir.
func IsTempDir(path string) (yes bool, err error) {
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	fullTempDirPath, err := filepath.Abs(parsedConfig.TempDir)
	if err != nil {
		return
	}

	if pathAbs == fullTempDirPath {
		return true, nil
	}

	return false, nil
}

// IsDirectoryToIgnore returns true if the given path is in the DirectoriesToIgnore.
func IsDirectoryToIgnore(path string) bool {
	for _, d := range parsedConfig.DirectoriesToIgnore {
		if strings.HasPrefix(path, d) {
			return true
		}
	}

	return false
}

// HasExtensionToWatch returns true if the given path contains an extension we should
// watch for changes.
func HasExtensionToWatch(path string) bool {
	//Make sure path isn't to a file in our temporary directory. This shouldn't really
	//be needed since thte temp dir shouldn't be watched for file changes anyway.
	//
	//Errors are ignored just for ease of using this func in select.
	absolutePath, _ := filepath.Abs(path)
	absoluteTempPath, _ := filepath.Abs(parsedConfig.TempDir)
	if strings.HasPrefix(absolutePath, absoluteTempPath) {
		return false
	}

	//Check if this file has a valid extension we want to watch for.
	extension := filepath.Ext(path)
	return slices.Contains(parsedConfig.ExtensionsToWatch, extension)
}

// UseDefaults populates the config with the default settings. This should ONLY be
// used for setting up the environment for running tests.
func UseDefaults() {
	cfg := newDefaultConfig()
	parsedConfig = cfg
}
