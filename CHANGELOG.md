# Change Log

## v0.1.1

> This release fixes a stack overflow and updates several first/third-party dependencies.

- fix
  - Stack overflow during audit phase if a type is encountered which contains a field of the same type, e.g. `type T struct { next *T }`.
  - When a configuration error is detected, the printed error list has two numbered groups when there should only be one (all commands).
- feat
  - --version now prints details about the build's paths and modules.
- notable dependency changes
  - Bump golang.org/x/tools to v0.0.0-20200220155224-947cbf191135.
  - Bump gopkg.in/yaml.v2 to v2.2.8.
  - Bump internal/cage/... to latest from monorepo.
- refactor
  - Migrate to latest cage/cli/handler API (e.g. handler.Session and handler.Input) and conventions (e.g. "func NewCommand").


## v0.1.0

- feat: initial project export/import support
