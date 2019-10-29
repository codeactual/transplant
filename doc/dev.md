[← Documentation Index](README.md)

### Development

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Contributing](#contributing)
- [Project layout](#project-layout)
- [Tests](#tests)
- [Documentation](#documentation)
- [FAQ](#faq)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

* * *

# Contributing

Questions, suggestions, issues, and PRs are welcome.

# Project layout

Main components:

- [CLI](cli.md)
  - Location: `[cmd](https://godoc.org/github.com/codeactual/transplant/cmd)`
- API (internal-only)
  - `[internal/transplant](https://godoc.org/github.com/codeactual/transplant/internal/transplant)`
    - `Audit` and `Copier` types
- [Shared first-party dependencies](README.md#shared-first-party-dependencies)
  - `internal/cage/go/packages`
    - `Inspector` type
- Third-party dependencies
  - [x/tools/go/packages](https://godoc.org/golang.org/x/tools/go/packages)

> `x/tools/go/packages` provides the package and AST analysis. `internal/cage/go/packages` wraps a lot of `x/tools/go/packages` functionality in its `Inspector`. `Audit` relies on `Inspector` to produce analysis specific to transplant's business logic. And finally `Copier` relies on the `Audit` results to peform the file processing and copying logic.

# Tests

- To reduce the per-test boilerplate, the [base suite](../internal/transplant/suite_test.go) includes some ["magic"](https://en.wikipedia.org/wiki/Magic_(programming)) that may be useful before adding a new test.

# Documentation

- Regenerate the table-of-contents in the `doc/` directory: `make toc`
  - Requires: `doctoc` ([npm](https://www.npmjs.com/package/doctoc), [github](https://github.com/thlorenz/doctoc))

# FAQ

- Why does the [CLI](cli.md) name its primary commands "export/import" while most of the code calls those modes "egress/ingress"?
  - To avoid the ambiguity and search-pollution related to import mode versus an `import` statement, name, package, path, etc. Also, "import" cannot be used as an import name.
- Why is a [directed acyclic graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph) generated but not really used?
  - An earlier version used them for pruning decisions. They have not been refactored out due to plans to enhance the `[why](cli.md)` commands with some degree of support for globals.
- Why are there references to "auto-detect", especially in tests and fixtures?
  - Originally the `[GoFilePath](config.md#structure)` fields was named `AutoDetectFilePath` and those files are not yet refactored.
- Why do all `go.mod` fixtures use an old `go 1.12`? And how do the tests still pass?
  - Tests which compare directory contents use the `DirsMatchExceptGomod` method of their suite. The method ensures that the dynamic `go` directive is effectively ignored as long as the expectation-fixture contains `go 1.12` (picked arbitrarily because this workaround was added during a migration to `1.13`).
