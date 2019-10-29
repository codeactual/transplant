[← Documentation Index](README.md)

### Features

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Modules](#modules)
  - [`go.mod/go.sum`](#gomodgosum)
  - [Vendoring](#vendoring)
- [Topologies](#topologies)
- [Refactoring](#refactoring)
  - [Pruning](#pruning)
    - [Intent](#intent)
    - [Scope](#scope)
    - [Implementation packages](#implementation-packages)
    - [Test packages](#test-packages)
  - [Filenames](#filenames)
- [Import mode](#import-mode)
  - [Propagating project-local modifications back to the origin](#propagating-project-local-modifications-back-to-the-origin)
  - [Propagating dependency modifications back to the origin](#propagating-dependency-modifications-back-to-the-origin)
    - [`go.mod/go.sum` modifications](#gomodgosum-modifications)
    - [`Ops.Dep` file tree modifications](#opsdep-file-tree-modifications)
- [Traits](#traits)
  - [Unsupported](#unsupported)
    - [Redundant import statements](#redundant-import-statements)
    - [Shadowed import and global identifier names](#shadowed-import-and-global-identifier-names)
  - [Untested](#untested)
    - [Implementation and test files in the same package](#implementation-and-test-files-in-the-same-package)
    - [Packages which are not named after their directories](#packages-which-are-not-named-after-their-directories)
    - [Files containing multiple init functions](#files-containing-multiple-init-functions)
  - [Partial support](#partial-support)
    - [Dot/unqualified imports](#dotunqualified-imports)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

* * *

# Modules

## `go.mod/go.sum`

The relevant `require` and `replace` directives of the origin module's `go.mod` are propagated to the copy's `go.mod`. A `go.sum` will be generated by `go mod tidy`.

## Vendoring

transplant populates a `vendor` directory in the copy if all of these are true:

1. `GOFLAGS` contains `-mod=vendor`
1. [`<Ops.From.ModuleFilePath>`](config.md#structure)`/go.sum` exists
1. `<Ops.From.ModuleFilePath>/vendor/modules.txt` exists

The latter two requirements are used as an indication that `go mod vendor` has already been used in the origin.

# Topologies

For more information about the supported origin/copy topologies, see the [topologies section of the configuration docs](config.md#topologies).

# Refactoring

## Pruning

### Intent

The main intent of pruning [shared first-party dependencies](README.md#shared-first-party-dependencies) identified by [Ops.Dep sections](config.md#structure) in the config file is based on several assumptions:

1. The origin module may contain many of this type of package.
1. A given package may contain many globals which are not used by the project targeted for extraction.
1. All stakeholders desire the copy to be as small as possible.

### Scope

### Implementation packages

If a global defined in an [`Ops.Dep`](config.md#structure) implementation package is not used, directly or transitively, by an [`Ops.From.LocalFilePath`](config.md#structure) package, it is omitted from the copy.

### Test packages

If test support is enabled for an [`Ops.Dep`](config.md#structure), all tests will be included in full and their dependencies will be satisfied.

:warning: Since tests are not pruned, and may cover many globals which are not actually used by the extracted project, the copy will include those globals in order to satisfy the dependencies of the tests. Tests may be pruned in future versions if feasible.

## Filenames

If a package is copied from the top of the [`Ops.From.LocalFilePath/Ops.Dep.From.FilePath`](config.md#structure) file tree, and contains implementation or test Go files which are named after that top-level directory, the naming convention is maintained in an copy using the [`Ops.To.FilePath/Ops.Dep.To.FilePath`](config.md#structure) directory name.

Example:

- [`Ops.From.LocalFilePath/Ops.Dep.From.FilePath`](config.md#structure) is named `clock`
- `clock/` contains two files: `clock.go` and `clock_test.go`
- [`Ops.To.LocalFilePath/Ops.Dep.To.FilePath`](config.md#structure) is named `time`
- In the copy, the files will be renamed to: `time.go` and `time_test.go`

# Import mode

## Propagating [project-local](README.md#target-project) modifications back to the origin

Changes to files in the copy, which originated in [`Ops.From.LocalFilePath`](config.md#structure), are supported by an import-mode copy operation.

## Propagating dependency modifications back to the origin

Currently this is not supported, but PRs/suggestions are welcome. Below are the rationales for this lack of support.

### `go.mod/go.sum` modifications

The primary rationale is that it's unclear how to reconcile conflicts between changes made to those trees in the origin's codebase and those made to the copy.

### [`Ops.Dep`](config.md#structure) file tree modifications

The primary rationale is the same as for `go.mod/go.sum` modifications.

# Traits

> This section highlights project/code traits which have limited support or are unsupported and may result in an error.

## Unsupported

### Redundant import statements

:warning: If this traits is detected, the [CLI](cli.md) will exit and display one or more errors.

- Example: `import db "path/to/database"` and `import "path/to/database"`
- Scope: all analyzed Go files.
  - files under [`Ops.Dep.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.Dep.From.FilePath`](config.md#structure))
- Rationale: This duplication interferes with the import statement pruning required after pruning an unused globals which was the only dependents of the import. Also, it's unclear what value is lost by retaining this type of [lint](https://en.wikipedia.org/wiki/Lint_(software)).

### Shadowed import and global identifier names

:warning: If this traits is detected, the [CLI](cli.md) will exit and display one or more errors.

- Scope: all analyzed Go files.
  - files under [`Ops.Dep.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.Dep.From.FilePath`](config.md#structure))
- Examples:
  - `import "time"` + `func f() { var time string }` in the same file
  - Global `var conn Connection` + `func f() { var conn Connection }` in the same file
- Exceptions:
  - method receiver names
  - function/method parameter names
- Rationale: Allowing for these types of ambiguity would complicate the analysis performed to support pruning.

## Untested

### Implementation and test files in the same package

:warning: This trait is not detected but may lead to unexpected results because support has not been tested.

- a.k.a. "white-box" tests
- Scope: all analyzed Go files.
  - files under [`Ops.Dep.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.Dep.From.FilePath`](config.md#structure))
- Rationale: test file refactoring currently relies on [golang.org/x/tools/go/packages](https://godoc.org/golang.org/x/tools/go/packages) which assumes the `*_test` packaging convention during inspection. So it's expected that white-box test cases will be pruned erroneously and other undesired behavior may occur.

### Packages which are not named after their directories

:warning: This trait is not detected but may lead to unexpected results because support has not been tested.

- Example: `db/db.go` contains the clause `package database`
- Scope: all analyzed Go files.
  - files under [`Ops.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.From.LocalFilePath`](config.md#structure))
  - files under [`Ops.Dep.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.Dep.From.FilePath`](config.md#structure))
- Rationale: transplant implementation would become more complicated when this convention cannot be relied upon.

### Files containing multiple init functions

:warning: This trait is not detected but may lead to unexpected results because support has not been tested.

- Scope: all analyzed Go files.
  - files under [`Ops.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.From.LocalFilePath`](config.md#structure))
  - files under [`Ops.Dep.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.Dep.From.FilePath`](config.md#structure))
- Rationale: Unclear how often the syntax feature is used in the wild.

## Partial support

### Dot/unqualified imports

:warning: Importing `Ops.Dep` packages with a `.` import name will prevent pruning and cause all package globals to be included in the copy regardless of use.

- Example: `import . "path/to/database"`
- Scope: all analyzed **implementation** Go files.
  - files under [`Ops.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.From.LocalFilePath`](config.md#structure))
  - files under [`Ops.Dep.From.GoFilePath`](config.md#structure) dirs (default: all under [`Ops.Dep.From.FilePath`](config.md#structure))
- Rationale: Pruning support can be restored by simply avoiding use of this type of import.