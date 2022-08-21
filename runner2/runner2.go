/*
Package runner2 handles building and running the binary when a file has been modified.
*/
package runner2

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/c9845/fresher/config"
	"github.com/howeyc/fsnotify"
	"golang.org/x/exp/slices"
)

// Define loggers. Multiple loggers because of prefixes on log lines for easier
// reading of logs and diagnosing of errors.
var (
	mainLog   log.Logger
	watchLog  log.Logger
	runnerLog log.Logger
	buildLog  log.Logger
	appLog    log.Logger
)

// Define channels.
var (
	startChan chan string //fsnotify String formats the event e in the form "filename: DELETE|MODIFY|..."
	stopChan  chan bool
)

// Configure handles some system configuration before watching for file changes and
// handling building and running the binary.
func Configure() (err error) {
	//Set the number of maximum file descriptors that can be opened by this process.
	//This is needed for watching a HUGE amount of files.
	//
	//Windows is not applicable.
	if runtime.GOOS != "windows" {
		var rLimit syscall.Rlimit
		rLimit.Max = 10000
		rLimit.Cur = 10000
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			err = fmt.Errorf("runner.Configure: error setting rlimit %w", err)
			return
		}
	}

	//Set up the logging. Multiple loggers based on what is writing to logs. This was
	//done because `fresh` used multiple loggers, each with different colors. We remove
	//the colors but keep the prefixed logs for easier identifying of log lines.
	mainLog = *log.New(os.Stderr, "main ", 0)
	watchLog = *log.New(os.Stderr, "watch ", 0)
	runnerLog = *log.New(os.Stderr, "runner ", 0)
	buildLog = *log.New(os.Stderr, "build ", 0)
	appLog = *log.New(os.Stderr, "app ", 0)

	return
}

var ErrSkipDir = errors.New("runner: skipping directory")

// Watch handles watching files. This skips ignored directories and only watches for
// changes on files with correct extensions per the config file field ExtensionsToWatch.
func Watch() {
	//Get root directory to start watching from. This is the working directory per the
	//config file.
	workingDir := config.Data().WorkingDir

	//Walk the directory tree, checking if each file found should be watched.
	filepath.Walk(workingDir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			//Ignore directory if it is the temp dir where we store built binaries.
			yes, err := isTempDir(path)
			if err != nil {
				return err
			}
			if yes {
				//Returning error since WalkFunc requires an error to be returned.
				return ErrSkipDir
			}

			//Ignore directory if it is in list of ignored directories.
			//
			//Ignored directories listed in config file are based off of the root of
			//the repository which is also the WorkingDir in the config file. The path
			//in the WalkFunc here is also based off of the WorkingDir, so therefore
			//we can compare easily without having to handle absolute paths.
			if isPathInIgnoredDirectory(path) {
				//Returning error since WalkFunc requires an error to be returned.
				mainLog.Println("Skipping dir", path)
				return ErrSkipDir
			} else {
				//diagnostics
				mainLog.Println("NOT SKIP DIR", path)
			}

			//Watch for file changes in this directory.
			watchDirectory(path)
		}

		return err
	})
}

// isTempDir checks if a given path is the path the the temporary directory where we
// store built binaries. Ignore directory if it is the temp dir where we store built
// binaries.
func isTempDir(path string) (yes bool, err error) {
	fullWalkPath, err := filepath.Abs(path)
	if err != nil {
		return
	}
	fullTempDirPath, err := filepath.Abs(config.Data().TempDir)
	if err != nil {
		return
	}

	if fullWalkPath == fullTempDirPath {
		return true, nil
	}

	return false, nil
}

// isPathInIgnoredDirectory checks if path is within DirectoriesToIgnore.
func isPathInIgnoredDirectory(path string) bool {
	return slices.Contains(config.Data().DirectoriesToIgnore, path)
}

// watchDirectory sets up fsnotify watchers for each file within a directory. This is
// what powers the "watch for file changes" functionality. Only files with valid
// extensions per the config file field ExtensionsToWatch are watched for changes.
func watchDirectory(path string) {
	//Set up filesystem notifier.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln(err)
		return
	}

	//Set up events to watch for.
	//watcher.Event.Name returns the path and name of a file.
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if isWatchedFile(ev.Name) {
					watchLog.Printf("Sending event %s", ev)
					startChan <- ev.String()
				}
			case err := <-watcher.Error:
				watchLog.Println("Error", err)
			}
		}
	}()

	//Watch the directory.
	watchLog.Println("Watching", path)
	err = watcher.Watch(path)

	if err != nil {
		log.Fatalln(err)
		return
	}
}

// isWatchedFile checks if the file at path is a file we should be watching for changes.
// This returns true on files with valid file extensions per the config file field
// ExtensionsToWatch.
func isWatchedFile(path string) bool {
	//Make sure path isn't to a file in our temporary directory.
	//Errors are ignored just for ease of using this func in select.
	absolutePath, _ := filepath.Abs(path)
	absoluteTempPath, _ := filepath.Abs(config.Data().TempDir)
	if strings.HasPrefix(absolutePath, absoluteTempPath) {
		return false
	}

	//Check if this file has a valid extension we want to watch for.
	extension := filepath.Ext(path)
	return slices.Contains(config.Data().ExtensionsToWatch, extension)
}

// start runs the building and running.
func start() {
	//Loop counter to see how many times fresher has noticed a file change. Just for
	//info purposes.
	loopIndex := 0

	//Get build delay so that we don't rebuild too fast. This helps when "save" happens
	//multiple times in quick succession.
	buildDelay := config.Data().BuildDelayMilliseconds

	//Is fresher running the binary.
	started := false

	go func() {
		for {
			//Increment for diagnostics.
			loopIndex++
			mainLog.Printf("Waiting (loop %d)...", loopIndex)

			//Read from channel to start. EventName will contain the name of the file
			//that has been modifed.
			eventName := <-startChan
			mainLog.Printf("Receiving first event %s", eventName)
			mainLog.Printf("Sleeping for %d milliseconds", buildDelay)
			time.Sleep(time.Duration(buildDelay) * time.Millisecond)

			mainLog.Printf("Flushing events")
			flushStartChan()

			//Should NumGoroutines should closely match number of files being watched?
			//TODO: explain this better.
			mainLog.Printf("Started! (%d Goroutines)", runtime.NumGoroutine())
			err := deleteBuildErrorsLog()
			if err != nil {
				mainLog.Println("Error deleting build log.", err)
				//not exiting on error since this isn't an end-of-the-world event.
			}

			//Track if build is successful.
			buildFailed := false

			//Determine if we need to rebuild the binary. We really only need to
			//rebuild if a .go file changes (unless the binary is using embedded
			//files, but that is a totally different deal). This simply speeds up
			//rebuilds by not requiring recompiling if, say, an HTML file is changed.
			//
			//eventName contains the name of the file that has been modified.
			//See the fsnotify *FileEvent.String method for more info.
			if shouldRebuild(eventName) {
				//Build the binary. Same as running `go build`.
				errMsg, buildSuccessful := build()
				if !buildSuccessful {
					buildFailed = true
					mainLog.Printf("Build Failed: \n %s", errMsg)
					if !started {
						os.Exit(1)
					}

					//Save the build error output to a log file for more easily
					//anaylizing errors. There could be stacktraces and a whole bunch
					//of other info that is easier to look at in a file versus in a
					//terminal.
					saveBuildErrorsLog(errMsg)
				}
			}

			//Run the built binary.
			if !buildFailed {
				if started {
					stopChan <- true
				}

				run()
			}

			//Note that binary is started and add log line that separates fresher's
			//logging from app's logging.
			started = true
			mainLog.Println(strings.Repeat("-", 20))
		}
	}()
}

// flushStartChan empties the startChan.
func flushStartChan() {
	for {
		select {
		case eventName := <-startChan:
			mainLog.Printf("Receiving event %s", eventName)
		default:
			return
		}
	}
}

// deleteBuildErrorsLog deletes the build errors log file located at the path noted in
// the config file field BuildLogFilename within the TempDir. Each time a new binary
// is built the error log is deleted and recreated if another error occurs.
func deleteBuildErrorsLog() (err error) {
	pathToFile := filepath.Join(config.Data().TempDir, config.Data().BuildLogFilename)
	err = os.Remove(pathToFile)
	return
}

// shouldRebuild checks if the file noted in an event for a modified file has a valid
// extension for a file we should rebuild the binary on. This compares the file noted
// in the eventName against the config file field NoRebuildExtensions.
func shouldRebuild(eventName string) bool {
	//Get the filename from the event.
	fileName := strings.Replace(strings.Split(eventName, ":")[0], `"`, "", -1)

	//Check if filename has an extension that rebuilding should be skipped.
	extension := filepath.Ext(fileName)
	return slices.Contains(config.Data().NoRebuildExtensions, extension)
}

// build builds the binary. This runs `go build` and outputs a binary to the temp
// directory noted in the config file.
//
// A string is returned only upon an stderr output when an stderr occurs in `go build`.
// True is returned when build is successful.
func build() (string, bool) {
	buildLog.Println("Building...")

	//Create name of output binary. The temp path is added to this since that is
	//where we store the built binary.
	pathToBuiltBinary := getPathToBuiltBinary()
	mainLog.Println("Saving binary to:", pathToBuiltBinary)

	//Get path to entry point of app. This is typically just the repository root.
	//Not naming a single "main.go" file allows for using any .go filename as the
	//entry point, and using multiple .go files at once.
	entryPoint := config.Data().WorkingDir

	//Build arguments passed to "go".
	args := []string{
		"build",
		"-o", pathToBuiltBinary,
	}

	if len(config.Data().Tags) > 0 {
		tags := strings.Join(config.Data().Tags, " ")
		args = append(args, "-tags", tags)
	}
	args = append(args, entryPoint)

	//Set up logging for when the command runs. We want to capture the output logging
	//and output it to the user running fresher. This is so the user can see any output
	//from running `go build` to diagnose issues.
	cmd := exec.Command("go", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	//Run the command.
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	//Copy output for stdout.
	_, err = io.Copy(os.Stdout, stdout)
	if err != nil {
		log.Fatal(err)
	}

	//Wait for command to finish before dumping error logging.
	errBuf, err := io.ReadAll(stderr)
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		return string(errBuf), false
	}

	//Build was successful. Binary is now located in temp dir.
	return "", true
}

// getPathToBuiltBinary returns the path to where the build binary will be saved.
// Basically, append BuildName to TempDir and add .exe if needed.
func getPathToBuiltBinary() string {
	path := filepath.Join(config.Data().TempDir, config.Data().BuildName)
	if runtime.GOOS == "windows" && filepath.Ext(path) != ".exe" {
		path += ".exe"
	}

	return path
}

// saveBuildErrorsLog saves the stderr output from `go build` when build() is called
// to a file. This file is deleted each time a build is attempted via
// deleteBuildErrorsLog which is called in start().
func saveBuildErrorsLog(message string) {
	//Get path to log file.
	pathToFile := filepath.Join(config.Data().TempDir, config.Data().BuildLogFilename)

	//Create the file.
	f, err := os.Create(pathToFile)
	if err != nil {
		log.Println("Could not create log file", err)
		//not exiting on error since we don't do anything with error anyway.
	}
	defer f.Close()

	//Write to file
	_, err = f.WriteString(message)
	if err != nil {
		log.Println("Could not write log file", err)
		//not exiting on error since we don't do anything with error anyway.
	}
}

// run runs the binary build in build().
//
// True is returned upon successfully running the binary.
func run() bool {
	runnerLog.Println("Running...")

	//Get path to built binary.
	pathToBuiltBinary := getPathToBuiltBinary()
	mainLog.Println("Running binary at:", pathToBuiltBinary)

	//Set up logging for when the command runs. We want to capture the output logging
	//and output it to the user running fresher. This is so the user can see any output
	//from running the binary to diagnose issues.
	cmd := exec.Command(pathToBuiltBinary)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalln(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalln(err)
	}

	//Run the binary.
	err = cmd.Start()
	if err != nil {
		log.Fatalln(err)
	}

	//Copy output from the command to output from fresher. This way the output from
	//the binary is displayed to the user.
	go io.Copy(appLog.Writer(), stderr)
	go io.Copy(appLog.Writer(), stdout)

	//Handle a user killing fresher. The binary should also be killed.
	go func() {
		<-stopChan
		pid := cmd.Process.Pid
		runnerLog.Printf("Killing PID %d", pid)
		cmd.Process.Kill()
	}()

	return true
}

// Start calls start() to handle building the running the binary.
func Start() {
	start()

	//Start handling directories to watch for changes.
	startChan <- "/"

	<-make(chan int)
}
