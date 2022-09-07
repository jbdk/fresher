# fresher
Automatic rebuilding and running of Go apps on source file changes.


# Purpose:
fresher is designed for use during development and alleviates the need to execute `go run` each time a source code file is changed. You can simply run `fresher` in place of `go run` and your binary will be rebuilt upon any file changes.


# Installing:
Run `go install github.com/c9845/fresher@latest`.

# Usage:
Run `fresher` in the same directory as you would run `go run`.

For more advanced usage, and customizing how `fresher` works, run `fresher -init` to create a config file in the current directory. The config file is pretty self-explainatory, however, see the [config file description](#config-file-details) for more details.


# How `fresher` Works:
1. The directory tree, starting where fresher is run is traversed, recusively.
2. Each directory that contains at least one file with an applicable extension (i.e.: .go) is watched.
3. When a file is changed, `go build` is run. The built binary is then run. This is repeated upon each file change.


# Rewrite of fresh:
`fresher` is a rewrite of `github.com/gravityblast/fresh` (previously known as `github.com/pilu/fresh`) to add better configuration, improved code base, and improved performance. You can use fresher in the same manner as fresh.

#### Configuration:
- `fresh`'s configuration file was custom. `fresher` uses a YAML file. `fresher`'s configuration file is much easier to understand.
- `fresh` did not allow for using go build tags. `fresher` allows for this via the config file for the `-tags` flag being passed through.
- `fresh`'s configuration *will not* work with `fresher`, although it is somewhat similar and can be translated.

#### Improved Code:
- `fresher` modernizes the codebase using the latest third-party libraries, latest Golang features and standard library, and implements go modules.
- Code documentation is immensely better; `fresh` had near-zero code documentation`. This should help with future development and maintance. 

#### Improved Performance:
- `fresher` is a bit faster due to reduced build delays and faster rebuilds when a file change occurs during an ongoing build (the ongoing build is killed in `fresher`; `fresh` waited for the build to complete before rebuilding).
- File changes on Windows are handled better; duplicate file change events are caught preventing needless rebuilds.
- `fresher` uses one watcher goroutine instead of one goroutine per watched directory.

# Config File Details:

| Field | Description | Default|
|-------|-------------|--------|
| WorkingDir | The directory `fresher` should operate on. | . |
| TempDir | The name of the directory of of WorkingDir that `fresher` uses for storing the built binary and error logs. | "tmp" |
| ExtensionsToWatch | The types of files `fresher` will watch for changes. Typically just files used in a binary. | [".go", ".html"] |
| NoRebuildExtensions | The types of files `fresher` will just rerun, not rebuild, the binary on upon a file change occuring. Typically this includes files that are read and cached by a running binary (for example, HTML templates via the `html/template` package), but not included in the binary. Caution if you use embedded files! | [".html"] |
| DirectoriesToIgnore | Directories that will not be watched for changes, recursively. | ["tmp", "node_modules", ".git", ".vscode"]
| BuildDelayMilliseconds | The amount of time to wait after a file change event occurs before rebuilding the binary. A delay is useful for catching multiple saves happening in rapid succession. You should not need to set this higher than 300. | 300 | 
| BuildName | The name of the binary as built by `fresher`. This file is stored in TempDir. | fresher-build |
| BuildLogFilename | The name of the log file where errors from `go build` will be saved to. This file is stored in TempDir. | fresher-build-errors.log |
| GoTags | Anything you would provide to `go run -tags` or `go build -tags`. | "" |
| GoLdflags | Anything you would provide to `go build -ldflags`. | "-s -w" |
| GoTrimpath | If the `-trimpath` flag is provided to `go build`. | true |
| VerboseLogging | If extra logging is provided while `fresher` is running. | false |

Create a config file with the `fresher -init` command.

Some config file fields can be overridden by flags to `fresher`.
- -tags.
- -verbose.


# FAQs: 

### Why not just use `air` (https://github.com/cosmtrek/air)?
In my testing and usage `fresh` is much faster than `air`. Although `air` is a more-or-less a fork of `fresh`, it does not improve performance. It seems like `air` is much "heavier" and there is more of a focus on tooling (for building `air`, i.e.: testing, CI, tools, etc.) rather than the underlying functionality.

