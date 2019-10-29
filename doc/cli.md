[← Documentation Index](README.md)

### CLI

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Commands](#commands)
  - [Export mode: copy the project out of the origin module](#export-mode-copy-the-project-out-of-the-origin-module)
    - [Maintenance](#maintenance)
  - [Import mode: migrate changes back into the origin module](#import-mode-migrate-changes-back-into-the-origin-module)
    - [Preparation](#preparation)
    - [Error messages](#error-messages)
  - [Check if a file/dir will be copied by a `run` command](#check-if-a-filedir-will-be-copied-by-a-run-command)
  - [Dry-run](#dry-run)
    - [Plan file](#plan-file)
      - [Formats](#formats)
      - [Fields](#fields)
- [Quick walkthrough](#quick-walkthrough)
  - [Config file](#config-file)
  - [Export](#export)
  - [Modify the copy](#modify-the-copy)
  - [Import](#import)
- [Staging directory](#staging-directory)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

* * *

# Commands

- All offer `--help` content.
- All require an `--op <id>` with a user-defined operation ID matching one declared in the [config file](config.md#structure).
- All require `--config <file>` unless the [file](config.md#files) is located at a default location:
  - `./transplant.yml`
  - `./transplant.yaml`
  - `./transplant.json`
  - `./transplant.toml`

## Export mode: copy the project out of the origin module

```
transplant export run --op <id>
```

### Maintenance

:warning: Due to current limitations of `import`, the more changes to those dependencies in the origin that accrue since the most recent export, the more work may be required to reconcile them with changes made to the exported copy when the latter is imported back.

## Import mode: migrate changes back into the origin module

```
transplant import run --op <id>
```

:warning: Currently only [project-local](README.md#terminology) files are included. Modifications to `go.mod/go.sum` ([#2](https://github.com/codeactual/transplant/issues/2)) and [shared first-party dependencies](README.md#shared-first-party-dependencies) ([#1](https://github.com/codeactual/transplant/issues/1)) are not.

### Preparation

- Update the config as needed to account for new files which do not fit the currently selected globs or exact matches.

### Error messages

:warning: Error messages may refer to a From/To value in the config file but label it with the opposite direction. This is due to how the need for separate export/import configurations is avoided by simply reversing the relevant From/To values. ([#3](https://github.com/codeactual/transplant/issues/3))

## Check if a file/dir will be copied by a `run` command

```
transplant export why --op <id> <file or dir>
transplant import why --op <id> <file or dir>
```

The `why` commands provides insight into the configs and processing related to the input path.

## Dry-run

Both commands support a dry-run mode, activated by `--plan <file>`, which writes a [JSON/TOML/YAML description](#plan-file) of the planned actions such as file creations, overwrites, and removals. In this mode, the CLI will exit after the copy is generated in the [staging directory](#staging-directory) but before the configured destination is modified.

### Plan file

:information_source: [Structure](https://godoc.org/github.com/codeactual/transplant/internal/transplant#CopyPlan)

The plan file has two purposes:

1. Inform you about what actions would be taken during a live-run.
1. Provide details that may be useful to copy/paste into bug reports.

Its main content includes lists of the destination files which will be added, overwritten, or removed. Empty lists are omitted.

#### Formats

The format is based on the selected file's extension.

- [JSON](https://learnxinyminutes.com/docs/json/): `--plan file.json`
- [TOML](https://learnxinyminutes.com/docs/toml/): `--plan file.toml`
- [YAML](https://learnxinyminutes.com/docs/yaml/): (default)

#### Fields

Some fields are omitted by default because they tend to be too verbose to see every time. Look for `--plan-fields` in `--help` output for the current list.

# Quick walkthrough

> This export/import cycle is based on the steps taken before publishing the initial release on GitHub.

## Config file

Make some final changes to the [annotated config file](../transplant.yml) which defines both the `export `and `import` behavior.

## Export

The config file is parameterized using the [support for environment variables](config.md#environment), so several of the major fields need to be defined externally.

- The target project is located at `/path/to/module/in/monorepo/tools/transplant`.
- The GitHub clone of this repository is at `/path/to/copy`.
- The copy's `go.mod` should include a `module github.com/codeactual/transplant` directive, and all relevant import statements in the copy should be refactored to use it.
- The user-defined operation ID is `github` (it doesn't relate to any GitHub-specific feature).

```bash
  # Adapted from a Makefile to avoid typing the environment variables each time.
  origin_module_filepath="/path/to/module/in/monorepo" \
    origin_local_filepath="tools/transplant" \
    copy_module_importpath="github.com/codeactual/transplant" \
    copy_module_filepath="/path/to/copy" \
    transplant export run --op github --plan export-plan

  cd /path/to/copy
  git push
```

1. Review the `export-plan` to see what changes will be made in `/path/to/copy`.
1. Run the prior `export` command but without `--plan` which enabled dry-run mode.

## Modify the copy

Updates to various Markdown documentation files were made through GitHub's web file editor.

## Import

```bash
  # Pull the changes made through the web UI.
  cd /path/to/copy
  git pull

  # Adapted from a Makefile to avoid typing the environment variables each time.
  origin_module_filepath="/path/to/module/in/monorepo" \
    origin_local_filepath="tools/transplant" \
    copy_module_importpath="github.com/codeactual/transplant" \
    copy_module_filepath="/path/to/copy" \
    transplant import run --op github --plan import-plan

  cd /path/to/module/in/monorepo
```

1. Review the `import-plan` to see what changes will be made in `/path/to/module/in/monorepo/tools/transplant`.
1. Run the prior `import` command but without `--plan` which enabled dry-run mode.
1. Manually reconcile the modifications seen in`git status/diff`, e.g. due to commits which happened between when the export and import occurred.
1. Commit the final result.

# Staging directory

In both export and import mode, files are first copied into a temporary directory.

If no errors were encountered in that step, only then are files added, overwritten, or removed from the configured destination. If an error does occur, stage's location will be displayed so the contents can be inspected.
