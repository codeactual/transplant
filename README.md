# transplant [![GoDoc](https://godoc.org/github.com/codeactual/transplant?status.svg)](https://godoc.org/github.com/codeactual/transplant) [![Go Report Card](https://goreportcard.com/badge/github.com/codeactual/transplant)](https://goreportcard.com/report/github.com/codeactual/transplant) [![Build Status](https://travis-ci.org/codeactual/transplant.png)](https://travis-ci.org/codeactual/transplant)

transplant exports a Go project out of a multi-project module into a standalone module.

The target use-case is the sharing of the project's tags/releases from a private monorepo, ensuring that the standalone's go.mod uses the exact same third-party dependencies as the origin and only bundles the minimum [shared first-party dependencies](doc/README.md#shared-first-party-dependencies).

Its CLI consists of two main commands:

1. `transplant export run`: create/update the standalone copy
1. `transplant import run`: propagate changes to the standalone copy's local files back into the origin module

This repository itself was exported using [transplant.yml](transplant.yml) as the config file. Below is the full command which relies on the support for [variables in the config values](doc/config.md#variables).

```
origin_module_filepath="/path/to/origin/module" \
  origin_local_filepath="tools/transplant" \
  copy_module_importpath="github.com/codeactual/transplant" \
  copy_module_filepath="/path/to/clone/of/this/repo" \
  transplant export run --op github
```

The above degree of parameterization is optional and all values can be self-contained in the config file.

## Supports

- Automatic copying of the target project and [shared first-party dependencies](doc/README.md#shared-first-party-dependencies) from the origin.
- Automatic [pruning](doc/features.md#pruning) of unused globals/files from shared first-party dependencies.
- Automatic `go.mod/go.sum/vendor` creation with versions synced with the origin.
- Automatic rewriting of import paths and names.
- Granular configuration of file include/exclude patterns, import paths, file renaming, source tree topologies, etc.

# Documentation

- [Topics](doc/README.md)
- [GoDoc](https://godoc.org/github.com/codeactual/transplant/internal/transplant)

# Installation

- Latest tag: `go get github.com/codeactual/transplant/cmd/transplant`
- Latest commit: `go get github.com/codeactual/transplant/cmd/transplant@master`

# License

[Mozilla Public License Version 2.0](https://www.mozilla.org/en-US/MPL/2.0/) ([About](https://www.mozilla.org/en-US/MPL/), [FAQ](https://www.mozilla.org/en-US/MPL/2.0/FAQ/))
