[← Documentation Index](README.md)

### Configuration

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Files](#files)
- [Structure](#structure)
  - [`Ops`](#ops)
  - [`Template`](#template)
  - [`Include` and `Exclude`](#include-and-exclude)
  - [`CopyOnlyFilePath`](#copyonlyfilepath)
- [Precedence](#precedence)
  - [File paths](#file-paths)
- [Variables](#variables)
  - [Program-defined](#program-defined)
  - [User-defined](#user-defined)
  - [Environment](#environment)
- [Examples](#examples)
- [Tests](#tests)
- [Topologies](#topologies)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

* * *

# Files

- The [CLI](cli.md#commands) accepts [YAML](https://learnxinyminutes.com/docs/yaml/), [JSON](https://learnxinyminutes.com/docs/json/), and [TOML](https://learnxinyminutes.com/docs/toml/) files.
- In JSON/TOML files, use the same casing as in the YAML [reference](#structure) and [examples](#examples) in this document, e.g. `FilePath`.

# Structure

## `Ops`

> This root-level section holds keys which declare the operation's ID (e.g. `github` below) for use in [`--op`](cli.md#commands) selection, while values hold the operation's configuration.

- To manage repetition or parameterize values, [user-defined and environment variables](#variables) are supported.

<!-- YAML lines need to be maintained to about this length for rendering on GitHub. -->

```yaml
Ops:
  github:
    # From defines how to find the files in the origin module for copying, as well as
    # control some refactoring behaviors.
    #
    # - Required
    From:
      # ModuleFilePath is the origin module's go.mod directory.
      #
      # - Required
      ModuleFilePath: '/path/to/dir'

      # LocalFilePath is the target project's base path under ModuleFilePath.
      #
      # - Required
      # - Must be relative to ModuleFilePath.
      LocalFilePath: 'rel/path/to/dir'

      # GoFilePath identifies the scope of the search for Go packages. Packages in matching
      # directories will be analyzed and their discovered dependencies will be included in
      # the copy, either as shared first-party dependencies (identified in Ops.Dep) or as
      # third-party in a generated go.mod with versions synced to the origin's.
      #
      # Use it to identify the the local packages of the target project.
      #
      # Use Ops.Dep.From.GoFilePath, instead of this one, to identify shared first-party
      # dependencies which exist outside LocalFilePath.
      #
      # - Optional
      #   - default:
      #     - Include: LocalFilePath and all descendant directories
      #     - Exclude: all testdata directories and their descendant directories
      GoFilePath:

        # A directory will be selected if it matches any glob. The pattern must be relative
        # to LocalFilePath.
        Include:
          - 'rel/pattern/to/dir'

        # A directory will be omitted if it matches any glob. The pattern must be relative
        # to LocalFilePath.
        Exclude:
          - 'rel/pattern/to/dir'

      # CopyOnlyFilePath identifies files, Go or non-Go, which should be included in the copy.
      #
      # Use it to select files such as Makefile, LICENSE, documentation, or even Go files
      # which do not provide implementation code or test cases (e.g. test fixtures).
      #
      # A separate section in this document provides additional details about this config.
      #
      # - Optional
      CopyOnlyFilePath:

        # A file will be selected if it matches any glob. The pattern must be relative
        # to LocalFilePath.
        Include:
          - 'rel/pattern/to/dir'

        # A file will be omitted if it matches any glob. The pattern must be relative
        # to LocalFilePath.
        Exclude:
          - 'rel/pattern/to/dir'

      # GoDescendantFilePath is identical to CopyOnlyFilePath with an additional requirement:
      # files must be directly in, or in a descendant directory of, a package directory
      # selected by GoFilePath.
      #
      # For example, use it to select non-Go files which are (topographically) related to
      # copied Go packages.
      #
      # - Optional
      GoDescendantFilePath:

        # A file will be selected if it matches any glob. The pattern must be relative
        # to LocalFilePath.
        Include:
          - 'rel/pattern/to/dir'

        # A file will be omitted if it matches any glob. The pattern must be relative
        # to LocalFilePath.
        Exclude:
          - 'rel/pattern/to/dir'

      # RenameFilePath identifies files which should have different names in the copy.
      #
      # - Optional
      # - Paths must be exact (no globbing).
      # - Directories are not currently supported.
      # - If the Old files do not exist, their absolute paths will be present in `--plan <file>`
      #   content RenameNotFound field.
      RenameFilePath:

          # Must be relative to Ops.From.ModuleFilePath.
        - Old: 'rel/path/to/file'

          # Must be relative to Ops.To.ModuleFilePath.
          New: 'rel/path/to/file'

      # ReplaceString defines global string replacements to perform on copied files.
      #
      # - Optional
      ReplaceString:

        # ImportPath identifies files in which all instances of the import path of
        # Ops.From.ModuleFilePath (go.mod "module" directive value) should be replaced with the
        # Ops.To.ModuleFilePath.
        #
        # The files must also be identified by GoFilePath (as a copied member of a matched directory),
        # CopyOnlyFilePath, or GoDescendantFilePath.
        #
        # (Import statements in GoFilePath matches will always be replaced to align with the
        # destination. This config can be used in case there are other instances in source
        # files which need replacement, such as in string literals.)
        #
        # Warning: no errors are emitted if the patterns match zero files.
        ImportPath:

          # A file will be selected if it matches any glob. The pattern must be relative
          # to LocalFilePath.
          Include:
            - 'rel/pattern/to/dir'

          # A file will be omitted if it matches any glob. The pattern must be relative
          # to LocalFilePath.
          Exclude:
            - 'rel/pattern/to/dir'

      # Tests enables the inclusion of test packages found in the same directories as
      # implementation packages identified by GoFilePath.
      #
      # All dependencies of tests will be included in the copy.
      #
      # - Optional (default: false)
      Tests: true

    # To defines where to create (or update) the standalone module.
    To:

      # ModuleImportPath defines the value of the "module" directive in the copy's go.mod,
      # and determines the replacement value of ReplaceString.ImportPath matches.
      #
      # - Required
      ModuleImportPath: 'copy.tld/user/proj'

      # ModuleFilePath is where root directory of the copy to be created or updated.
      #
      # - Required
      ModuleFilePath: '/path/to/dir'

      # LocalFilePath is the destination of the Ops.From.LocalFilePath file tree.
      #
      # Use it to indicate where the Ops.From.LocalFilePath tree should be written relative
      # to ModuleFilePath.
      #
      # - Optional
      LocalFilePath: 'rel/path/to/dir'

    # Dep elements define From/To sections which perform the same function as Ops.From/Ops.To
    # sections but for shared first-party dependencies in the origin module. The exact packages
    # required by Ops.From.GoFilePath files are automatically detected, so even if a
    # Dep.From.FilePath is a large tree, only the minimum packages required will be copied.
    Dep:

        # From fields here are identical to Ops.From except for the differences noted below:
      - From:

          # FilePath is the location of the dependency's file tree relative
          # to Ops.From.ModuleFilePath.
          #
          # It is the Dep version of Ops.From.LocalFilePath.
          #
          # Omit this field if the origin module contains shared first-party dependencies at
          # the root.
          #
          # - Optional
          FilePath: 'rel/path/to/dir'

          # Tests enables the same functionality as Ops.From.Tests. The important difference in
          # behavior is that all project-local tests are included when Ops.From.Tests is true.
          # Here, a test package is only included if it shares a directory with an implementation
          # package that is a direct/transitive dependency of a Ops.From.LocalFilePath package.
          # This decision is based on the conventional placement of tests in Go code, and allows
          # the copy operation to skip tests which cover implementation packages that won't be
          # included anyway.
          #
          # - Optional (default: false)
          Tests: true

          # RenameFilePath is not supported here because Ops.From.RenameFilePath already supports
          # paths relative to (From/To) ModuleFilePath values.

        # To fields here are identical to Ops.To except for the differences noted below:
        To:

          # FilePath is the location of the dependency's file tree relative
          # to Ops.To.ModuleFilePath.
          #
          # It is the Dep version of Ops.To.LocalFilePath.
          #
          # Omit this field if the copy should contain this Dep's file tree starting at the
          # root of the destination module. Otherwise select a relative path,
          # e.g. "internal/my_dep" for internal APIs.
          #
          # - Optional
          FilePath: 'rel/path/to/dir'
```

## `Template`

> This root-level section allows you to define [variables](#user-defined) available in config values of all `Ops` elements.

- Optional
- Variables **must be** used inside single/double-quoted strings.
- Keys **must be** lower-cased.

```yaml
Template:
  # Now {{.key}} can be used in any text field under Ops.
  key: 'value'
```

## `Include` and `Exclude`

> Multiple config sections support `Include` and `Exclude` pattern lists for selecting files and directories.

- `Exclude` takes precedence.
- Each pattern must be relative to its associated base path, either `Ops.From.LocalFilePath` or `Ops.Dep.From.FilePath`.
  - :information_source: Search results of an all-inclusive pattern like `**/*` will include the associated base path itself. If some/all of the direct contents of that directory need to be omitted, you will need to replace the pattern with a more specific set. For example, replace `**/*` with `target1/**/*` and `target2/**/*` in order to avoid file `skip1` and directory `skip2/`.
- [Globbing](https://en.wikipedia.org/wiki/Glob_%28programming%29) is supported via the [bmatcuk/doublestar](https://github.com/bmatcuk/doublestar#patterns) library.
- When defining recursive patterns to select directories with a named directory, such as for `GoFilePath`, it's important to decide whether to include the named directory itself or not.
  - `dirname`: selects the directory itself
  - `dirname/**/*`: selects all descendant directories (but not the directory itself)
- Examples:
  - Match all Go files at any depth under a top-level `cmd` directory:
    - `cmd/**/*.go`
  - Match directories named `subdir` at any depth and all their contents recursively:
    - `**/subdir`
    - `**/subdir/**/*`
  - Match dot-prefixed directories at any depth and all their contents recursively:
    - `**/.*`
    - `**/.*/**/*`

## `CopyOnlyFilePath`

- Go files will be minimally processed (unlike `GoFilePath` matches). They will not be analyzed or pruned, and their dependencies will not be satisfied. However they will be processed by [`format.Source`](https://golang.org/pkg/go/format/#Source) (to clean up import path rewrites). If an error occurs, such as due to syntax errors, it will not cause the run end. Instead, it will be collected in the `GoFormatErr` section of the [`--plan`](cli.md#plan-file) output.

# Precedence

## File paths

- `CopyOnlyFilePath`
- `GoDescendantFilePath`
- `GoFilePath`
  - This config matches directories instead of files, so directories containing any files matched by the above configs will be omitted from `GoFilePath` results.

Use `RenameFilePath` to match files included by the above configs.

# Variables

All config values support the variable types below.

## Program-defined

- `_config_dir`: absolute path to the config file's directory

## User-defined

The top-level [`Template`](#template) section can hold key/value pairs using the`{{.key}}` syntax.

- :warning: Variables must be used inside single/double-quoted strings.
- :warning: Keys must be lower-cased.

## Environment

Both `$key` and `${key}` formats are supported.

# Examples

- transplant's own [annotated config file](../transplant.yml)
- Other configs on GitHub
  - [codeactual/aws-exec-cmd](https://github.com/codeactual/aws-exec-cmd/blob/master/transplant.yml)
  - [codeactual/aws-mockery](https://github.com/codeactual/aws-mockery/blob/master/transplant.yml)
  - [codeactual/aws-req](https://github.com/codeactual/aws-req/blob/master/transplant.yml)
  - [codeactual/boone](https://github.com/codeactual/boone/blob/master/transplant.yml)
  - [codeactual/ec2-mount-volume](https://github.com/codeactual/ec2-mount-volume/blob/master/transplant.yml)
  - [codeactual/gomodfuzz](https://github.com/codeactual/gomodfuzz/blob/master/transplant.yml)
  - [codeactual/testecho](https://github.com/codeactual/testecho/blob/master/transplant.yml)

# Tests

To include any test packages found via `Ops.From.GoFilePath` or `Ops.Dep.From.GoFilePath`, set `Ops.From.Tests` or `Ops.Dep.From.Tests` to `true`.

# Topologies

transplant supports a variety of file tree topologies of [origin modules](README.md#origin) and the per-project copies extracted from them.

In short, the aim is to support any topology which satisfies these contraints:

- `Ops.From.LocalFilePath` must be non-empty
  - Rationale: An empty value implies the project begins at the root of the origin module, which suggests the module and project are more or less the same thing: all first-party dependencies are already a part of its file tree. This use case is outside the scope of this tool.
- `Ops.Dep.From.FilePath` is a descendant of `Ops.From.LocalFilePath`
  - Rationale: Same as above.
- One `Ops.Dep.From.FilePath` overlaps with another
  - Rationale: This suggests a higher-level file tree root should be selected instead which includes both of them. The copy will only include the packages which are directly/transitively used by the project.
- One `Ops.Dep.To.FilePath` overlaps with another
  - Rationale: Support for this would add unexplored complexity for a use case that is currently unknown. Also, if support for importing `Ops.Dep` changes from the copy back to the origin is explored, allowing this type of overlapping in the copy would likely complicate it.
