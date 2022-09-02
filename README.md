# fresher
Automatic rebuilding and running of Go apps on source file changes.


### Purpose:
fresher is designed for app-development use and alleviates the need to run stop and `go run` your binary each time a source code file is changed. You can simply run `fresher` in place of `go run` and your binary will be rebuild upon any file changes.


### How fresher Works:
1. The directory tree, starting where fresher is run is traversed, recusively.
2. Each directory that contains at least one file with an applicable extension (i.e.: .go) is watched.
3. When a file is changed, `go build` is run. The built binary is then run. This is repeated upon each file change.


### Rewrite of fresh:
This is a rewrite of `github.com/gravityblast/fresh` (previousl known as `github.com/pilu/fresh`) to add better configuration, better code documenation, and improved performance.

fresh's configuration was basically a one-off, roll-your-own, custom method. It was not very clear. fresher's configuration aims to be simpler to understand, use, and configure.

fresh's codebase had near-zero code documentation. fresher has a lot. This should help with future maintenance.

fresh had a build delay after a file change was recognized. This was, to my understanding, a way to get around building too fast upon file changes.


## Motivation:
`fresh` works well, but:
  - The configuration file is in a self-defined format, it does not use YAML, JSON, or something easily understood and commonly understood. 
  - The configuration options also lack the ability to support build tags or other such features.
  - The code base is nearly totally undocumented so adding documentation should help future maintance. 
  - Does not use go modules.
  - Performance is quite good, but hopefully we can make it a bit faster.

## FAQs: 

### Why not just use `air` (https://github.com/cosmtrek/air)?
Simply put, in my testing and usage `fresh` is much faster than `air`. This is odd since `air` is a fork of `fresh` more recent maintenance. However, it seems `air` is much "heavier" and there is more of a focus on tooling (testing, dev. tools, etc.) rather than the underlying functionality.

