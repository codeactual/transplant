[← Documentation Index](readme.md)

### Support

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


  - [Known Issues / Limitations](#known-issues--limitations)
  - [Support](#support)
  - [Limitations](#limitations)
- [Pruning](#pruning)
  - [Support](#support-1)
- [Additional refactoring](#additional-refactoring)
- [Unsupported code traits](#unsupported-code-traits)
- [Modules](#modules)
  - [Vendoring](#vendoring)
- [Topologies](#topologies)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

* * *

## Known Issues / Limitations

- if a PR updates a `go.{mod,sum}` dependency, it must be propagated to the monorepo manually (allowing a natural opportunity to evaluate the effects on other dependents in the monorepo, etc.)
- document why ingress does not currently copy any `Ops.Dep` globals
  - eval `$EXP/transplant/ingress/ast/algo/algo_test.go` TODO comments which already describe several of the reasons
  - Think of `Ops.Dep` as similar to `require` in `go.mod`, or a `vendor/`, except that the dependencies are first-party. However, just like third-party dependencies, altering them can have many side-effects in known/unknown dependents. This alone favors stepping back from trying to automating `Ops.Dep` maintenance during ingress, aside from the technical reasons focused on in `algo_test.go`.
- no support for multiple init functions are defined within a single file: their dependencies may not be detected and get omitted from the copy

## Support

> transplant always attempts to create a `go.mod`, `go.sum`, and `go mod vendor` generated `vendor` directory.
>
> If `Op.From.ModuleFilePath` is configured to indicate the `Op.From.FilePath` is inside of, or is itself, a module, then the copy's `go.mod` requirements are synced to match the origin's semver selections.

## Limitations

- While `require` directives are propagated to the egress copy, all others such as `replace` are not.
  - Rationale: currently a new `go.mod` is created each time via `go mod init <Op.To.ImportPath>` and it's unclear whether retaining other directives is desired. It's also unclear how to provide configuration options to support the need for different directive values in each repo, i.e. what they should be after an egress operation versus an ingress.
- Egress copies may contain `go.{mod,sum}` files with versions of `indirect` requirements that differ from the source when:
  - The source is not a module with a `go.mod` to sync with.
  - `go mod tidy` has not been used in the source module and the egress module is a `main` module.
    - transplant always runs `go mod tidy` in the copy stage directory in order to ensure the `go.mod` requirements are added to the file before they're synced with the source module's `go.mod`. `tidy` is also used to clean up unused versions from the copy's `go.sum`.
    - As of `go1.11.4` there is no guarantee that `go mod init` will seed the `require (...)` directives, and there is no other `go mod *` command to perform that function except `tidy`.

# Pruning

- expand description of test support and the impact of enabling it on pruning: because tests are not pruned, and their transitive dependencies are satisfied, that means potentially many implementation globals (not needed to compile the implementation) may get included simply so the tests compile
  - refine the above based on #435 outcome

The intent is minimize the copy's scope by omitted all `Op.From.FilePath`-external, non-vendored dependencies from the egress copy. This is based on several assumptions:

- The origin module may provide common/shared packages which are large, numereous, or both, and including them whole is not desired for any reason. The desire is to co-locate only the common/shared code that is required for the egress copy to build and test.
- Pruning vendored dependencies is out-of-scope and the responsibility/domain of a separate package management tool. Also, it's unclear how OSS licenses affect the automated refactoring of third-party source code.

## Support

> transplant always attempts to omit unused global identifiers from all `Op.Dep.From.FilePath` implementation packages it copies to the destination.
>
> Test packages are not pruned, although aspects such as import paths will be rewritten to align with those selected in the config file.

- `Op.From.FilePath` packages always remain.
  - Their direct/transitive dependencies influence which `Dep.From.FilePath` packages remain.
  - Rationale: even when the module has one or more `main` packages from which dependency roots can be assumed, it's assumed that any exported globals should remain available to maintain the public API surface.
- `Op.Dep.From.FilePath` packages
  - Global exported/non-exported types, functions, constants, and variables remain if they are direct/transitive dependencies of `Op.From.FilePath` packages.
  - All `init` functions from `Dep.From.FilePath` packages remain if the packages are direct/transitive dependencies of `Op.From.FilePath` packages.
    - Their own direct/transitive dependencies are satisfied recursively, which may cause more `init` functions to remain, and so on.
    - Rationale: due to the nature of `init` functions, it's safer to assume they're required for the package to operate as intended.
  - If at least one method of type `T` is used, all `T` methods remain.
    - Rationale: avoid issues such as a `*T` no longer satisfying an interface it otherwise would.
  - Global-scope blank identifiers (`_`) are pruned unless they're used in value assignments where global identifiers on both sides are otherwise required as direct/transitive dependencies of `Op.From.FilePath` packages.
    - For example, interface assertion `var _ I = (*T)(nil)` is pruned as a whole if interface type `I` or concrete type `T` are not otherwise dependencies.
    - Rationale: the interface assertion case was the only one considered where its critical for the identifer to remain. In that case, the assertion is irrelevant if interface conformance is never exercised (e.g. in a function parameter type) or a potential implementation of it is never used.
- Constants with `iota`-based values are not pruned.
 - Rationale: to avoid special handling effort, e.g. keeping all of them if any in a group are used or remapping the `= iota` to the first in the group of identifiers that remain which might cause consistency issues if the values are serialized/transmitted/etc.

# Additional refactoring

- If a package is copied from the top-level of a `Op.[Dep.]From.FilePath` directory, and contains implementation or test Go files which are named after the directory, the naming convention is maintained in an egress copy using the`Op.[Dep.]To.FilePath` names.
- In test files, `Ops.From.{File,Import}Path` config values are rewritten to their respective `Ops.To.{File,Import}Path` values in string literals in order to support test cases which contain those types of hard-coded paths.

# Unsupported code traits

- new code trait: files which import the same path twice but with different names (because it interferes with import pruning that follows up pruning of unused globals); issue that came up in #426
- package names which do not align with dir names (e.g. to allow computed `*.ImportPath` config values based on concatenating the module import path with a relative file path)
- expand explanation about why global shadowing isn't supported (see #419)
- clarify whether the unsupported code traits apply to `Ops.From` and/or `Ops.Dep.From` files when listing them

- Shadowed import and global identifier names
  - Rationale: it's unknown whether their usage is common enough to justify updating the global-usage-and-shadowing-related code to handle import/identifier name conflicts/ambiguity.
- `import .` syntax
  - Rationale: it's unknown whether their usage is common enough to justify updating the global-usage-and-shadowing-related code to handle identifier name conflicts/ambiguity.
  - Exceptions:
    - Usage in test files.
- Test files in the same packages as implementations
  - Rationale: test file refactoring currently relies on [golang.org/x/tools/go/packages](https://godoc.org/golang.org/x/tools/go/packages) which assumes the `*_test` packaging convention during inspection. So it's expected that white-box test cases will be pruned erroneously and other undesired behavior may occur.


> ----------------------------------------------------------------

# Modules

transplant configuration is currently module-centric. For example, it requires you to specify the location of the origin's module root, where to generate the copy's module, and the import path of the latter. Even the support for vendoring is based on `go mod vendor`.

The `require` and `replace` directives of the origin module's `go.mod` are propagated to the copy's `go.mod`.

## Vendoring

transplant populates a `vendor` directory in the copy if all of these are true:

1. `GOFLAGS` contains `-mod=vendor`
1. `<Ops.From.ModuleFilePath>/go.sum` exists
1. `<Ops.From.ModuleFilePath>/vendor/modules.txt` exists

The latter two requirements are used as an indication that `go mod vendor` has already been used in the origin.

# Topologies

For more information about the supported origin/copy topologies, see the [topologies section of the configuration docs](config.md#topologies).
