// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package transplant defines the core types used by both egress and ingress logic.
package transplant

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	std_viper "github.com/spf13/viper"

	cage_viper "github.com/codeactual/transplant/internal/cage/config/viper"
	cage_mod "github.com/codeactual/transplant/internal/cage/go/mod"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	cage_structs "github.com/codeactual/transplant/internal/cage/structs"
	cage_template "github.com/codeactual/transplant/internal/cage/text/template"
)

// FilePathQuery defines file selection criteria.
type FilePathQuery struct {
	// Include holds github.com/bmatcuk/doublestar glob patterns which validate a candidate path.
	//
	// If any matches, the candidate path is accepted.
	Include []string

	// Exclude holds github.com/bmatcuk/doublestar glob patterns which invalidate a candidate path.
	//
	// If any matches, the candidate path is rejected.
	Exclude []string
}

func (q FilePathQuery) Copy() (cpy FilePathQuery) {
	cpy.Include = append(cpy.Include, q.Include...)
	cpy.Exclude = append(cpy.Exclude, q.Exclude...)
	return cpy
}

// ResolveTo updates all patterns to absolute path form.
func (q *FilePathQuery) ResolveTo(basePath string) {
	for p := range q.Include {
		q.Include[p] = filepath.Join(basePath, q.Include[p])
	}
	for p := range q.Exclude {
		q.Exclude[p] = filepath.Join(basePath, q.Exclude[p])
	}
}

func (q FilePathQuery) Validate() (errs []error) {
	for n, p := range q.Include {
		if p == "" {
			errs = append(errs, errors.Errorf("Include[%d] is empty", n))
		}
	}
	for n, p := range q.Exclude {
		if p == "" {
			errs = append(errs, errors.Errorf("Exclude[%d] is empty", n))
		}
	}
	return errs
}

// RenameSpec defines a file/directory path change.
//
// Definitions are currently restricted to single file/directory changes in order for them
// to be reverisble in an egress/ingress cycle. For example, if a glob pattern was used
// to select files to move during egress, the same pattern might be insufficient during
// ingress because it matches on additional files that were there before the egress operation.
type RenameSpec struct {
	// Old is a relative path.
	//
	// It is a source path during egress and destination path during ingress.
	//
	// It must identify a file already selected in the operation, e.g. via CopyOnlyFilePath
	// or other *FilePath query.
	Old string

	// New is a relative path.
	//
	// It is a destination path during egress and source path during ingress.
	//
	// It must identify a file already selected in the operation, e.g. via CopyOnlyFilePath
	// or other *FilePath query.
	New string
}

// ReplaceStringSpec defines the scope of string replacements to perform during copy operations.
type ReplaceStringSpec struct {
	// ImportPath matches files which should have Ops.From/Ops.To import paths converted
	// when found outside import statements of parsed files identified via GoFilePath:
	// e.g. CopyOnlyFilePath and GoDescendantFilePath matches
	//
	// Each pattern is relative to the FilePath field in the parent RootFrom/DepFrom.
	ImportPath FilePathQuery
}

// RootFrom describes the origin of a copy operation.
type RootFrom struct {
	// ModuleFilePath is the absolute path to the root of the origin module where the go.mod can be found.
	//
	// It is the base directory for all *FilePath patterns.
	//
	// If it contains a go.sum, and `go mod tidy` execution in the destination during egress also
	// creates one, the destination's hashes will be verified against the origin's.
	//
	// If it contains a vendor/modules.txt file, `go mod vendor` will be used to create one in the destination.
	ModuleFilePath string

	// ModuleImportPath is the import path of the module rooted in ModuleFilePath.
	//
	// It is a copy of Module.Path and present for consistency with RootTo.ModuleImportPath.
	ModuleImportPath string `mapstructure:"-"`

	// LocalFilePath is a path relative to ModuleFilePath which declares the root of all project-local files.
	//
	// It is read from the config file, resolved to an absolute path, and then used as the base path for
	// all Ops.From.*FilePath patterns.
	LocalFilePath string

	// LocalImportPath is the prefix of all packages under LocalFilePath.
	//
	// It is computed based on ModuleImportPath and LocalFilePath.
	LocalImportPath string `mapstructure:"-"`

	// GoFilePath matches directories in which implementation (and optionally test) packages should
	// be copied and analyzed for their dependencies.
	//
	// Matching directories will be inspected.
	// If Tests is true, test packages will be auto-discovered.
	// Only Go files will be copied.
	GoFilePath FilePathQuery

	// CopyOnlyFilePath matches files which should be copied but not inspected or refactored
	// (except for basic string replacement of source file/import paths).
	//
	// Each pattern is relative to FilePath.
	CopyOnlyFilePath FilePathQuery

	// GoDescendantFilePath first matches files which should be copied but not inspected or refactored
	// (except for basic string replacement of source file/import paths), same as CopyOnlyFilePath.
	// It also requires that each file is a descendant of directory which provides a Go package to the copy
	// (i.e. a dependency of a project matched by GoFilePath).
	//
	// For example, it supports the need to exclude files such as test fixtures whose only dependents are in
	// ancestor directories which will not be part of the copy.
	GoDescendantFilePath FilePathQuery

	// Rename defines file/directory path changes to perform during the copy operation.
	RenameFilePath []RenameSpec

	// ReplaceString defines the scope of string replacements to perform during copy operations.
	//
	// Replacements occur on the intersection between its matches and those of all other config patterns,
	// such as GoFilePath and CopyOnlyFilePath, which control the scope of the copy operation itself.
	ReplaceString ReplaceStringSpec

	// ModuleSum is true if <ModuleFilePath>/go.sum was found during config validation/finalization.
	ModuleSum bool `mapstructure:"-"`

	// Tests is true if GoFilePath-matched test packages and their dependencies should be included.
	Tests bool

	// Vendor is true if GOFLAGS contains "-mod=vendor" and <ModuleFilePath>/{go.sum,vendor/modules.txt} exists
	// (as indication that the origin is using the same vendoring tool as we are).
	//
	// go.sum is required because a prior `go mod vendor` run should have created one.
	Vendor bool `mapstructure:"-"`
}

// RootTo describes the destination of a copy operation.
type RootTo struct {
	// ModuleFilePath is the absolute path to the root of the copy's tree.
	ModuleFilePath string

	// ModuleImportPath is the import path of the module copied to ModuleFilePath.
	ModuleImportPath string

	// LocalFilePath is a path relative to ModuleFilePath which declares the root of all copied project-local files.
	LocalFilePath string

	// LocalImportPath is the prefix of all packages under LocalFilePath.
	//
	// It is computed based on ModuleImportPath and LocalFilePath.
	LocalImportPath string `mapstructure:"-"`
}

// DepFrom describes the origin of a specific dependency included in the copy operation.
type DepFrom struct {
	// FilePath is a path relative to Ops.From.ModuleFilePath.
	//
	// It is used as the base path for all *FilePath patterns.
	FilePath string

	// ImportPath is the prefix of all packages under FilePath.
	//
	// It is computed based on Ops.From.ModuleImportPath and this type's FilePath.
	ImportPath string `mapstructure:"-"`

	// GoFilePath matches directories in which implementation (and optionally test) packages should
	// be copied and analyzed for their dependencies.
	//
	// Implementation packages will only be included if they're direct/transitive dependencies
	// of the target project. Tests will only be included if enabled by the Tests config field
	// and they share a directory with an already included implementation packages.
	//
	// Matching directories will be inspected.
	// If Tests is true, test packages will be auto-discovered.
	// Only Go files will be copied.
	GoFilePath FilePathQuery

	// CopyOnlyFilePath matches files which should be copied but not inspected/refactored.
	//
	// Each pattern is relative to FilePath.
	CopyOnlyFilePath FilePathQuery

	// GoDescendantFilePath first matches files which should be copied but not inspected or refactored
	// (except for basic string replacement of source file/import paths), same as CopyOnlyFilePath.
	// It also requires that each file is a descendant of directory which provides a Go package to the copy
	// (i.e. a dependency of a project matched by GoFilePath).
	//
	// For example, it supports the need to exclude files such as test fixtures whose only dependents are in
	// ancestor directories which will not be part of the copy.
	GoDescendantFilePath FilePathQuery

	// ReplaceString defines the scope of string replacements to perform during copy operations.
	//
	// Replacements occur on the intersection between its matches and those of all other config patterns,
	// such as GoFilePath and CopyOnlyFilePath, which control the scope of the copy operation itself.
	ReplaceString ReplaceStringSpec

	// Tests is true if a GoFilePath-matched test packages and their dependencies should be included.
	Tests bool
}

// DepTo describes the destination of a specific included in the copy operation.
type DepTo struct {
	// FilePath is a path relative to Ops.To.ModuleFilePath.
	FilePath string

	// ImportPath is the prefix of all packages under FilePath.
	//
	// It is computed based on Ops.To.ModuleImportPath and this type's FilePath.
	ImportPath string `mapstructure:"-"`
}

// Dep describes a specific dependency included in the copy operation.
type Dep struct {
	From DepFrom
	To   DepTo
}

// Op describes a package/project copy operation.
type Op struct {
	// Id is a copy of the Config.Ops key selected by the user in the config file.
	//
	// It supports cases such as error messages in parts of the code that only receive the Op
	// and not the whole Config from which it belongs.
	Id string

	// From describes the origin module location.
	From RootFrom

	// To describes the standalone-repo location.
	To RootTo

	// Dep describes dependencies of code in From that exists elsewhere in the origin module and is not vendored,
	// e.g. first-party packages/modules centrally shared in the repo.
	Dep []Dep

	// DryRun is true if the operation should perform all steps except creating/modifying Ops.To.FilePath.
	DryRun bool `mapstructure:"-"`

	// Ingress is true if the copy mode/direction is from a standalone module to the origin source tree.
	Ingress bool `mapstructure:"-"`
}

// finalizeEgress updates the Op with direction-specific modifications which are not expressed
// in the direction-agnostic Op structure. Currently a placeholder only.
func (op *Op) finalizeEgress() {
}

// finalizeIngress updates the Op with direction-specific modifications which are not expressed
// in the direction-agnostic Op structure.
func (op *Op) finalizeIngress() {
	from := op.From

	op.From.ModuleFilePath = op.To.ModuleFilePath
	op.From.ModuleImportPath = op.To.ModuleImportPath
	op.From.LocalFilePath = op.To.LocalFilePath
	op.From.LocalImportPath = op.To.LocalImportPath

	var renames []RenameSpec
	for _, r := range op.From.RenameFilePath {
		renames = append(renames, RenameSpec{Old: r.New, New: r.Old})
	}
	op.From.RenameFilePath = renames

	op.To.ModuleFilePath = from.ModuleFilePath
	op.To.ModuleImportPath = from.ModuleImportPath
	op.To.LocalFilePath = from.LocalFilePath
	op.To.LocalImportPath = from.LocalImportPath

	for d := range op.Dep {
		from := op.Dep[d].From

		op.Dep[d].From.FilePath = op.Dep[d].To.FilePath
		op.Dep[d].From.ImportPath = op.Dep[d].To.ImportPath

		op.Dep[d].To.FilePath = from.FilePath
		op.Dep[d].To.ImportPath = from.ImportPath
	}

	op.Ingress = true
}

// Config is the unmarshaled structure of the YAML config file.
type Config struct {
	// Ops holds all refactor operation definitions indexed by a user-defined ID.
	Ops map[string]Op

	// Template holds key/value pairs which can be used in some string fields via {{.key_name}} syntax.
	//
	// Key names must use lowercase due to viper(/mapstructure?) limitation. Convention: "some_key_name".
	// https://github.com/spf13/viper/issues/411
	// https://github.com/spf13/viper/pull/635
	Template map[string]string
}

// ReadFile populates Config fields with values from the named file.
//
// If the name is empty, it checks if transplant.* files in the working directory are present
// (in order: *.yml, *.yaml, *.json, *.toml) and defaults to the first match. If none of the
// default names exist, an error is returned.
//
// It also validates the fields expected to be user-defined and computes others which are derived from the former.
func (c *Config) ReadFile(name string, _opIds ...string) (errs []error) {
	opIds := cage_strings.NewSet().AddSlice(_opIds)

	if name == "" {
		for _, ext := range []string{"yml", "yaml", "json", "toml"} {
			candidate := "transplant." + ext
			if exists, _, err := cage_file.Exists(candidate); err != nil {
			} else if exists {
				name = candidate
				break
			}
		}
	}

	if name == "" {
		return []error{errors.New("no config file selected")}
	}

	file := std_viper.New()
	if err := cage_viper.ReadInConfig(file, name); err != nil {
		return []error{errors.Wrapf(err, "failed to locate config file [%s]", name)}
	}

	if err := file.UnmarshalExact(c); err != nil {
		return []error{errors.Wrapf(err, "failed to parse file [%s]", name)}
	}

	if len(c.Ops) == 0 {
		return []error{errors.Errorf("config file [%s] defined no operations (Ops map)", name)}
	}

	// expand program-defined template variables in the user-defined Template section

	configFilePath := filepath.Dir(name)
	if absErr := cage_filepath.Abs(&configFilePath); absErr != nil {
		return []error{errors.Wrapf(absErr, "failed to resolve absolute path of [%s]", configFilePath)}
	}

	progTemplateData := map[string]string{
		"_config_dir": configFilePath,
	}

	var tmplExpectKeys []string // select which template key/value pairs to expand in the Template section
	for k := range progTemplateData {
		tmplExpectKeys = append(tmplExpectKeys, k)
	}

	tmplDataBuilder := cage_template.NewStringMapBuilder()
	tmplDataBuilder.SetExpectedKey(tmplExpectKeys...).Merge(cage_structs.MergeModeCombine, progTemplateData)

	var mapSaveFuncs []func()
	tmplStrings := []*string{}

	for s := range c.Template {
		// use StringKeyPtr to work around lack of support for &c.Template[<key>] syntax
		valPtr, save, mapErr := cage_strings.StringKeyPtr(&c.Template, s)
		if mapErr != nil {
			errs = append(errs, errors.Wrapf(mapErr, "failed to update to Template[%s] value", s))
		}
		mapSaveFuncs = append(mapSaveFuncs, save)
		tmplStrings = append(tmplStrings, valPtr)
	}

	tmplErr := cage_template.ExpandFromStringMap(tmplDataBuilder.Map(), tmplStrings...)
	if tmplErr != nil {
		errs = append(errs, errors.Wrap(tmplErr, "failed to expand program-defined variables in Template config section"))
	}

	if len(errs) > 0 {
		return errs
	}

	for _, f := range mapSaveFuncs {
		f()
	}

	// - select which template key/value pairs to expand in the Ops section
	// - trim leading/trailing space from value strings
	// - expand environment variables in value strings

	var opTmplExpectKeys []string
	for k := range progTemplateData {
		opTmplExpectKeys = append(opTmplExpectKeys, k)
	}
	for k := range c.Template {
		opTmplExpectKeys = append(opTmplExpectKeys, k)
		c.Template[k] = os.ExpandEnv(c.Template[k])
	}

	for opId, op := range c.Ops {
		if !opIds.Contains(opId) {
			continue
		}

		op.Id = opId

		// expand program/user-defined template variables in the user-defined Ops section

		opTmplDataBuilder := cage_template.NewStringMapBuilder()

		// user-defined pairs
		opTmplDataBuilder.SetExpectedKey(opTmplExpectKeys...).Merge(cage_structs.MergeModeCombine, c.Template)

		// program-defined pairs
		opTmplDataBuilder.Merge(cage_structs.MergeModeOverwrite, progTemplateData)

		opValueStrings := []*string{
			&op.From.ModuleFilePath,
			&op.From.ModuleImportPath,
			&op.From.LocalFilePath,
			&op.From.LocalImportPath,

			&op.To.ModuleFilePath,
			&op.To.ModuleImportPath,
			&op.To.LocalFilePath,
			&op.To.LocalImportPath,
		}

		for s := range op.From.GoFilePath.Include {
			opValueStrings = append(opValueStrings, &op.From.GoFilePath.Include[s])
		}
		for s := range op.From.GoFilePath.Exclude {
			opValueStrings = append(opValueStrings, &op.From.GoFilePath.Exclude[s])
		}

		for s := range op.From.CopyOnlyFilePath.Include {
			opValueStrings = append(opValueStrings, &op.From.CopyOnlyFilePath.Include[s])
		}
		for s := range op.From.CopyOnlyFilePath.Exclude {
			opValueStrings = append(opValueStrings, &op.From.CopyOnlyFilePath.Exclude[s])
		}

		for s := range op.From.GoDescendantFilePath.Include {
			opValueStrings = append(opValueStrings, &op.From.GoDescendantFilePath.Include[s])
		}
		for s := range op.From.GoDescendantFilePath.Exclude {
			opValueStrings = append(opValueStrings, &op.From.GoDescendantFilePath.Exclude[s])
		}

		for s := range op.From.RenameFilePath {
			opValueStrings = append(opValueStrings, &op.From.RenameFilePath[s].Old, &op.From.RenameFilePath[s].New)
		}

		for s := range op.From.ReplaceString.ImportPath.Include {
			opValueStrings = append(opValueStrings, &op.From.ReplaceString.ImportPath.Include[s])
		}
		for s := range op.From.ReplaceString.ImportPath.Exclude {
			opValueStrings = append(opValueStrings, &op.From.ReplaceString.ImportPath.Exclude[s])
		}

		for n := range op.Dep {
			opValueStrings = append(
				opValueStrings,
				&op.Dep[n].From.FilePath,
				&op.Dep[n].From.ImportPath,

				&op.Dep[n].To.FilePath,
				&op.Dep[n].To.ImportPath,
			)

			for s := range op.Dep[n].From.GoFilePath.Include {
				opValueStrings = append(opValueStrings, &op.Dep[n].From.GoFilePath.Include[s])
			}
			for s := range op.Dep[n].From.GoFilePath.Exclude {
				opValueStrings = append(opValueStrings, &op.Dep[n].From.GoFilePath.Exclude[s])
			}

			for s := range op.Dep[n].From.CopyOnlyFilePath.Include {
				opValueStrings = append(opValueStrings, &op.Dep[n].From.CopyOnlyFilePath.Include[s])
			}
			for s := range op.Dep[n].From.CopyOnlyFilePath.Exclude {
				opValueStrings = append(opValueStrings, &op.Dep[n].From.CopyOnlyFilePath.Exclude[s])
			}

			for s := range op.Dep[n].From.GoDescendantFilePath.Include {
				opValueStrings = append(opValueStrings, &op.Dep[n].From.GoDescendantFilePath.Include[s])
			}
			for s := range op.Dep[n].From.GoDescendantFilePath.Exclude {
				opValueStrings = append(opValueStrings, &op.Dep[n].From.GoDescendantFilePath.Exclude[s])
			}

			for s := range op.Dep[n].From.ReplaceString.ImportPath.Include {
				opValueStrings = append(opValueStrings, &op.Dep[n].From.ReplaceString.ImportPath.Include[s])
			}
			for s := range op.Dep[n].From.ReplaceString.ImportPath.Exclude {
				opValueStrings = append(opValueStrings, &op.Dep[n].From.ReplaceString.ImportPath.Exclude[s])
			}
		}

		opTmplErr := cage_template.ExpandFromStringMap(opTmplDataBuilder.Map(), opValueStrings...)
		if opTmplErr != nil {
			errs = append(errs, errors.Wrapf(opTmplErr, "failed to expand template variables in Ops[%s]", opId))
			continue
		}

		for _, s := range opValueStrings {
			*s = strings.TrimSpace(*s)
			*s = os.ExpandEnv(*s)
		}

		// default values

		if len(op.From.GoFilePath.Include) == 0 {
			op.From.GoFilePath.Include = []string{"**/*"}
		}
		for n := range op.Dep {
			if len(op.Dep[n].From.GoFilePath.Include) == 0 {
				op.Dep[n].From.GoFilePath.Include = []string{"**/*"}
			}
		}

		// By default, exclude all testdata directories and their descendant directories
		// from analysis. (But make a selfish exception for transplant's own test fixtures.)
		if !strings.Contains(op.From.ModuleFilePath, string(filepath.Separator)+"testdata"+string(filepath.Separator)) {
			if len(op.From.GoFilePath.Exclude) == 0 {
				op.From.GoFilePath.Exclude = []string{"**/testdata", "**/testdata/**/*"}
			}
			for n := range op.Dep {
				if len(op.Dep[n].From.GoFilePath.Exclude) == 0 {
					op.Dep[n].From.GoFilePath.Exclude = []string{"**/testdata", "**/testdata/**/*"}
				}
			}
		}

		// empty value checks

		if op.From.ModuleFilePath == "" {
			errs = append(errs, errors.Errorf("Ops[%s].From.ModuleFilePath is empty", opId))
		}
		if op.To.ModuleFilePath == "" {
			errs = append(errs, errors.Errorf("Ops[%s].To.ModuleFilePath is empty", opId))
		}

		// The destination path must be defined, while Ops.From.ModuleImportPath is extracted from the go.mod
		// in "computed values" section.
		if op.To.ModuleImportPath == "" {
			errs = append(errs, errors.Errorf("Ops[%s].To.ModuleImportPath is empty", opId))
		}

		if op.From.LocalFilePath == "" {
			errs = append(errs, errors.Errorf("Ops[%s].From.LocalFilePath is empty", opId))
		}
		// Allow Ops.To.LocalFilePath to be empty to support egress of libraries which need to be
		// imported from the root of the project, e.g. "go get domain.com/user/project".

		// Allow Ops.Dep.From.FilePath to be empty to support egress of first-party dependencies
		// from the root of the module. For example a top-level package that exports
		// a Version string.

		// Allow Ops.Dep.To.FilePath to be empty to complement support for empty Ops.Dep.From.FilePath values,
		// where a first-party dependency is extracted from the root of the module and copied
		// to the root of the generated module. For example, copying both a top-level package
		// and ./internal to the same relative positions in the copy. Without allowing this field
		// to be empty, the ./internal in this case would require an extra directory above it for no purpose
		// except to pass validation.
		// computed values

		op.From.ModuleFilePath = FilepathClean(op.From.ModuleFilePath)
		if op.From.ModuleFilePath != "" && !filepath.IsAbs(op.From.ModuleFilePath) {
			errs = append(errs, errors.Errorf("Op[%s].From.ModuleFilePath [%s] cannot be relative", opId, op.From.ModuleFilePath))
		}
		originMod, err := cage_mod.NewModFromFile(FromAbs(op, "go.mod"))
		if err == nil {
			op.From.ModuleImportPath = originMod.Path
		} else {
			errs = append(errs, errors.Wrapf(err, "failed to parse <Ops[%s].From.ModuleFilePath>/go.mod", opId))
		}

		op.From.LocalFilePath = FilepathClean(op.From.LocalFilePath)
		if filepath.IsAbs(op.From.LocalFilePath) {
			errs = append(errs, errors.Errorf("Op[%s].From.LocalFilePath [%s] must be relative (to ModuleFilePath) ", opId, op.From.LocalFilePath))
		} else {
			// assume leaf package name conventionally matches the leaf dir name
			op.From.LocalImportPath = path.Join(op.From.ModuleImportPath, op.From.LocalFilePath)
		}

		op.To.ModuleFilePath = FilepathClean(op.To.ModuleFilePath)
		if op.To.ModuleFilePath != "" && !filepath.IsAbs(op.To.ModuleFilePath) {
			errs = append(errs, errors.Errorf("Op[%s].To.ModuleFilePath [%s] cannot be relative", opId, op.To.ModuleFilePath))
		}
		op.To.LocalFilePath = FilepathClean(op.To.LocalFilePath)
		if filepath.IsAbs(op.To.LocalFilePath) {
			errs = append(errs, errors.Errorf("Op[%s].To.LocalFilePath [%s] must be relative (to ModuleFilePath) ", opId, op.To.LocalFilePath))
		} else {
			// assume leaf package name conventionally matches the leaf dir name
			op.To.LocalImportPath = path.Join(op.To.ModuleImportPath, op.To.LocalFilePath)
		}

		for n := 0; n < len(op.Dep); n++ { // only use 'n' because we need to update ops.Dep[n] by pointer
			op.Dep[n].From.FilePath = FilepathClean(op.Dep[n].From.FilePath)
			if filepath.IsAbs(op.Dep[n].From.FilePath) {
				errs = append(errs, errors.Errorf("Ops[%s].Dep[%s].From.FilePath must be relative (to Ops[%s].From.ModuleFilePath)", opId, op.Dep[n].From.FilePath, opId))
			}

			op.Dep[n].To.FilePath = FilepathClean(op.Dep[n].To.FilePath)
			if filepath.IsAbs(op.Dep[n].To.FilePath) {
				errs = append(errs, errors.Errorf("Ops[%s].Dep[%s].To.FilePath must be relative (to Ops[%s].To.ModuleFilePath)", opId, op.Dep[n].From.FilePath, opId))
			}

			// assume leaf package name conventionally matches the leaf dir name
			op.Dep[n].From.ImportPath = path.Join(op.From.ModuleImportPath, op.Dep[n].From.FilePath)
			op.Dep[n].To.ImportPath = path.Join(op.To.ModuleImportPath, op.Dep[n].To.FilePath)
		}

		for _, r := range op.From.RenameFilePath {
			if r.Old == "" {
				errs = append(errs, errors.Errorf("Ops[%s].From.RenameFilePath.Old is empty", opId))
			}
			if r.New == "" {
				errs = append(errs, errors.Errorf("Ops[%s].From.RenameFilePath.New is empty", opId))
			}
		}

		if queryErrs := op.From.CopyOnlyFilePath.Validate(); len(queryErrs) > 0 {
			for _, queryErr := range queryErrs {
				errs = append(errs, errors.Wrapf(queryErr, "Ops[%s].From.CopyOnlyFilePath contains an empty string", opId))
			}
		}
		for _, dep := range op.Dep {
			if queryErrs := dep.From.CopyOnlyFilePath.Validate(); len(queryErrs) > 0 {
				for _, queryErr := range queryErrs {
					errs = append(errs, errors.Wrapf(queryErr, "Ops[%s].Dep[%s].From.CopyOnlyFilePath contains an empty string", opId, dep.From.FilePath))
				}
			}
		}

		if queryErrs := op.From.GoDescendantFilePath.Validate(); len(queryErrs) > 0 {
			for _, queryErr := range queryErrs {
				errs = append(errs, errors.Wrapf(queryErr, "Ops[%s].From.GoDescendantFilePath contains an empty string", opId))
			}
		}
		for _, dep := range op.Dep {
			if queryErrs := dep.From.GoDescendantFilePath.Validate(); len(queryErrs) > 0 {
				for _, queryErr := range queryErrs {
					errs = append(errs, errors.Wrapf(queryErr, "Ops[%s].Dep[%s].From.GoDescendantFilePath contains an empty string", opId, dep.From.FilePath))
				}
			}
		}

		if queryErrs := op.From.GoFilePath.Validate(); len(queryErrs) > 0 {
			for _, queryErr := range queryErrs {
				errs = append(errs, errors.Wrapf(queryErr, "Ops[%s].From.GoFilePath contains an empty string", opId))
			}
		}
		for _, dep := range op.Dep {
			if queryErrs := dep.From.GoFilePath.Validate(); len(queryErrs) > 0 {
				for _, queryErr := range queryErrs {
					errs = append(errs, errors.Wrapf(queryErr, "Ops[%s].Dep[%s].From.GoFilePath contains an empty string", opId, dep.From.FilePath))
				}
			}
		}

		if queryErrs := op.From.ReplaceString.ImportPath.Validate(); len(queryErrs) > 0 {
			for _, queryErr := range queryErrs {
				errs = append(errs, errors.Wrapf(queryErr, "Ops[%s].From.ReplaceString.ImportPath contains an empty string", opId))
			}
		}
		for _, dep := range op.Dep {
			if queryErrs := dep.From.ReplaceString.ImportPath.Validate(); len(queryErrs) > 0 {
				for _, queryErr := range queryErrs {
					errs = append(errs, errors.Wrapf(queryErr, "Ops[%s].Dep[%s].From.ReplaceString.ImportPath contains an empty string", opId, dep.From.FilePath))
				}
			}
		}

		// disallowed value checks

		if strings.Contains(op.From.LocalFilePath, "..") {
			errs = append(errs, errors.Errorf("Ops[%s].From.LocalFilePath [%s] cannot contain '..'", opId, op.From.LocalFilePath))
		}
		if strings.Contains(op.To.LocalFilePath, "..") {
			errs = append(errs, errors.Errorf("Ops[%s].To.LocalFilePath [%s] cannot contain '..'", opId, op.To.LocalFilePath))
		}
		for _, dep := range op.Dep {
			if strings.Contains(dep.From.FilePath, "..") {
				errs = append(errs, errors.Errorf("Ops[%s].Dep[%s].From.FilePath cannot contain '..'", opId, dep.From.FilePath))
			}
			if strings.Contains(dep.To.FilePath, "..") {
				errs = append(errs, errors.Errorf("Ops[%s].Dep[%s].To.FilePath cannot contain '..'", opId, dep.To.FilePath))
			}
		}

		// Ops.To / Ops.Dep.To duplicate/overlap checks

		// Disallow the destination trees to overlap to avoid the issue of local/dep files being copied
		// into the same package/package. This is mainly to catch typos in the config because it's unclear
		// why that scenario would be desired.
		toLocalFilePath := ToAbs(op, op.To.LocalFilePath)
		for n, dep := range op.Dep {
			depToFilePath := ToAbs(op, dep.To.FilePath)
			if depToFilePath == toLocalFilePath {
				errs = append(errs, errors.Errorf("Ops[%s].Dep[%d].To.FilePath [%s] conflicts with Ops.To.LocalFilePath [%s].", op.Id, n, dep.To.FilePath, op.To.LocalFilePath))
			}
		}

		// file-exists checks

		exists, _, existsErr := cage_file.Exists(filepath.Join(op.From.ModuleFilePath, "go.sum"))
		if existsErr != nil {
			errs = append(errs, errors.Wrapf(existsErr, "failed to check if <Ops[%s].From.ModuleFilePath>/go.sum exists", opId))
		} else if exists {
			op.From.ModuleSum = true
		}

		if strings.Contains(os.Getenv("GOFLAGS"), "-mod=vendor") {
			exists, _, existsErr = cage_file.Exists(filepath.Join(op.From.ModuleFilePath, "vendor", "modules.txt"))
			if existsErr != nil {
				errs = append(errs, errors.Wrapf(existsErr, "failed to check if <Ops[%s].From.ModuleFilePath>/vendor/modules.txt exists", opId))
			} else if exists {
				if op.From.ModuleSum {
					op.From.Vendor = true
				} else {
					// `go mod vendor` creates them at the same time, so also expect the go.sum.
					errs = append(errs, errors.Errorf("Ops[%s].From.ModuleFilePath contains a vendor/modules.txt without a go.sum", opId))
				}
			}
		}

		c.Ops[opId] = op
	}

	return errs
}

// FromAbs resolves the relative path parts to the origin module's root directory.
func FromAbs(op Op, parts ...string) string {
	return filepath.Join(append([]string{op.From.ModuleFilePath}, parts...)...)
}

// ToAbs resolves the relative path parts to the destination module's root directory.
func ToAbs(op Op, parts ...string) string {
	return filepath.Join(append([]string{op.To.ModuleFilePath}, parts...)...)
}

// FilepathClean prevents empty config paths from being converted to ".".
func FilepathClean(p string) string {
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}

// AllImportPathReplacer returns a replacer covering Ops.From.LocalImportPath and all Ops.Dep.From.ImportPath values.
func AllImportPathReplacer(op Op) *cage_strings.ReplaceSet {
	r := &cage_strings.ReplaceSet{}
	r.Add(op.From.LocalImportPath, op.To.LocalImportPath, -1)
	for _, dep := range op.Dep {
		r.Add(dep.From.ImportPath, dep.To.ImportPath, -1)
	}
	return r
}

// DepImportPathReplacer returns a replacer covering all Ops.Dep.From.ImportPath values.
func DepImportPathReplacer(op Op) *cage_strings.ReplaceSet {
	r := &cage_strings.ReplaceSet{}
	for _, dep := range op.Dep {
		r.Add(dep.From.ImportPath, dep.To.ImportPath, -1)
	}
	return r
}
