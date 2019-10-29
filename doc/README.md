# Topics

- [Command-line interface](cli.md)
- [Configuration](config.md)
- [Features](features.md)
- [Development](dev.md)

# Terminology

# origin

Refers to the module which contains the project being exported.

Also: `origin module`, `origin codebase`, `origin tree`

## Ops.From

Shorthand for an operation's top-level `From` section in the [config file](config.md#structure).

## Ops.Dep

Shorthand for an element of an operation's `Dep` section in the [config file](config.md#structure).

Also: `Dep`

## target project

Refers to the primary code tree(s) of an export/import operation, files identified by an operation's top-level `From` section in the [config file](config.md#structure).

Also: `local project`, `project-local <noun>`

## Shared first-party dependencies

Refers to Go packages which are direct/transitive dependencies of the `target project` which also exist in the `origin`.

Also: `Ops.Dep`, `Dep`

Examples:

- `docode` in https://blog.digitalocean.com/taming-your-go-dependencies/
