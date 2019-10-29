// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/codeactual/transplant/cmd/transplant/why"
	cage_crypto "github.com/codeactual/transplant/internal/cage/crypto"
	cage_errors "github.com/codeactual/transplant/internal/cage/errors"
	cage_go_list "github.com/codeactual/transplant/internal/cage/go/list"
	cage_mod "github.com/codeactual/transplant/internal/cage/go/mod"
	cage_io "github.com/codeactual/transplant/internal/cage/io"
	cage_exec "github.com/codeactual/transplant/internal/cage/os/exec"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_file_matcher "github.com/codeactual/transplant/internal/cage/os/file/matcher"
	cage_file_stage "github.com/codeactual/transplant/internal/cage/os/file/stage"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_runtime "github.com/codeactual/transplant/internal/cage/runtime"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

const (
	stagePathPrefix     = "transplant"
	newFileMode         = 0644
	newDirMode          = 0755
	skipOverwriteLogMsg = "omitted from final copy operation (content unchanged)"
)

// Copier performs both egress and ingress copy operations.
type Copier struct {
	// Audit results inform the copier operation by providing data such as the file lists and globals found in the AST.
	Audit *Audit

	// Ctx is applied to context-aware operations to support externally-defined cancellation.
	Ctx context.Context

	// Op is the configuration data used by both the Audit and copy operation to determine their scope, modes, etc.
	Op Op

	// OverwriteMin is true if the copy should only populate the staging dir, and overwrite files in the
	// destination dir, if the inbound content is different. Files with the same content will be added to
	// CopyPlan.OverwriteSkip.
	OverwriteMin bool

	// ModuleRequire is true if the Go modules step should be performed where the module is initialized
	// and receives requirements from the origin module.
	ModuleRequire bool

	// Plan enumerates the copy actions which would run, to support dry-run mode.
	Plan CopyPlan

	// ProgressCore receives messages describing core copy steps and runtimes.
	ProgressCore io.Writer

	// ProgressModule receives stdout messages from module-related Go commands.
	ProgressModule io.Writer

	// Stderr receives messages about errors which are printed but considered serious enough to be
	// collected and cause the operation to exit early.
	Stderr io.Writer

	// Stage abstracts a temporary directory which is populated fully before the actual copy operation
	// in order to minimize side-effects if an error is encountered.
	Stage *cage_file_stage.Stage

	// WhyLog if non-nil will receive updates which support `{egress,ingress} file` queries.
	WhyLog why.Log
}

// NewCopier returns an initialized instance.
func NewCopier(ctx context.Context, audit *Audit) (c *Copier, err error) {
	c = &Copier{
		Audit:          audit,
		Ctx:            ctx,
		Op:             audit.Op(),
		ProgressCore:   ioutil.Discard,
		ProgressModule: ioutil.Discard,
		Stderr:         os.Stderr,
	}

	// Account for the files which were pruned because they were not directly/transitively imported by audit.LocalGoFiles.
	for _, name := range c.Audit.AllDepGoFiles.SortedSlice() {
		switch {
		case c.Audit.UsedDepGoFiles.Contains(name):
		case c.Audit.DepGoTestFiles.Contains(name):
		default:
			c.Plan.PruneGoFiles = append(c.Plan.PruneGoFiles, name)
		}
	}

	c.Stage, err = cage_file_stage.NewTempDirStage(stagePathPrefix)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	c.Plan.StagePath = c.Stage.BasePath()

	return c, nil
}

// Run performs the copy operation in stages, calling out to separate methods which implement them.
// The separation of methods is for both readability and to simplfy logic which determines which
// stages execute based on configuration.
func (c *Copier) Run() (_ CopyPlan, errs []error) {
	steps := []struct {
		title string
		f     func() []error
		time  time.Duration
	}{
		{title: "register Ops.From.RenameFilePath changes", f: c.localRenames},
		{title: "copy Ops.From implementation files to stage", f: c.localGoFiles},
		{title: "copy Ops.From test files to stage", f: c.localGoTestFiles},
		{title: "copy Ops.From.CopyOnlyFilePath files to stage", f: c.localCopyOnlyFiles},
		{title: "copy Ops.Dep implementation files to stage", f: c.usedDepGoFiles},
		{title: "copy Ops.Dep test files to stage", f: c.depGoTestFiles},
		{title: "copy Ops.Dep.CopyOnlyFilePath files to stage", f: c.depCopyOnlyFiles},
		{title: "output stage", f: c.outputStage},
		{title: "copy module requirements to stage", f: c.moduleRequirements},
		{title: "copy stage to Ops.To", f: c.copyStage},
	}

	for n := range steps {
		// If we're going to echo the output from Go toolchain commands, don't bother showing
		// the start/end messages which will get split/lost anyway in the toolchain output.
		printProgressCore := steps[n].title != "copy module requirements to stage" || c.ProgressModule == ioutil.Discard

		if printProgressCore {
			fmt.Fprintf(c.ProgressCore, "copy [%s] ... ", steps[n].title)
		}

		start := time.Now()
		if errs := steps[n].f(); len(errs) > 0 {
			return c.Plan, errs
		}
		steps[n].time = time.Since(start)

		if printProgressCore {
			fmt.Fprintf(c.ProgressCore, "%s\n", steps[n].time)
		}
	}

	c.Plan.Env = make(map[string]string)
	c.Plan.Env["GO111MODULE"] = os.Getenv("GO111MODULE")
	c.Plan.Env["GOFLAGS"] = os.Getenv("GOFLAGS")
	c.Plan.Env["GOPATH"] = os.Getenv("GOPATH")
	c.Plan.Env["runtime.GOARCH"] = runtime.GOARCH
	c.Plan.Env["runtime.GOOS"] = runtime.GOOS
	c.Plan.Env["runtime.GOROOT"] = runtime.GOROOT()
	c.Plan.Env["runtime.Version"] = runtime.Version()

	return c.Plan, []error{}
}

// localRenames configures the stage to rename files during the copy process.
func (c *Copier) localRenames() (errs []error) {
	for _, p := range c.Op.From.RenameFilePath {
		renameOld := FromAbs(c.Op, p.Old)

		// During egress, we expect the rename target to exist in the origin.
		// During ingress, the renamed file in the copy may have been removed and we need to propagate the removal.
		//
		// This check was already performed during Audit.finalizeConfig, so we don't need to collect
		// any errors if the file does not exist. We do it again simply to know whether to adjust
		// the file's path for the stage.
		exists, _, existsErr := cage_file.Exists(renameOld)
		if cage_errors.Append(&errs, errors.Wrapf(existsErr, "failed to check if [%s] exists", renameOld)) {
			return errs
		}
		if !exists {
			c.Plan.RenameNotFound = append(c.Plan.RenameNotFound, renameOld)
			continue
		}

		if c.Op.Ingress { // renamed file exists in the copy, ensure the origin file and its dir remain
			toNew := ToAbs(c.Op, p.New)
			c.Audit.IngressRemovableFiles.Remove(toNew)
			c.Audit.IngressRemovableDirs.Remove(filepath.Dir(toNew))
		}

		// Old/New must be relative to stage root, not From.ModuleFilePath. Remove the current prefix
		// and prepend the destination prefix.
		if c.Op.From.LocalFilePath != "" {
			p.Old = strings.TrimPrefix(p.Old, c.Op.From.LocalFilePath+string(filepath.Separator))
		}
		if c.Op.To.LocalFilePath != "" {
			p.Old = filepath.Join(c.Op.To.LocalFilePath, p.Old)
		}

		c.Stage.Rename(p.Old, p.New) // rename is made pending, nothing is written yet
	}
	return errs
}

// skipWrite returns true if the input Ops.From file should be written to its equivalent
// Ops.To location because the latter's bytes match the input Reader's.
func (c *Copier) skipWrite(toAbsPath string, pendingContent io.ReadSeeker) (skip bool, err error) {
	currentContent, err := os.Open(toAbsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to open [%s] for checksum", toAbsPath)
	}

	same, _, err := cage_crypto.ReaderHashSumsEqual(sha256.New(), pendingContent, currentContent)
	if err != nil {
		return false, errors.Wrapf(err, "failed to compare checksum of [%s] for checksum", toAbsPath)
	}

	if same {
		c.Plan.OverwriteSkip = append(c.Plan.OverwriteSkip, toAbsPath)
		c.Stage.OverwriteSkip(toAbsPath)
	}

	return same, nil
}

// localGoFiles adds Op.From.LocalFilePath non-test Go files to the stage.
func (c *Copier) localGoFiles() (errs []error) {
	fromLocalFilePath := FromAbs(c.Op, c.Op.From.LocalFilePath)

	for _, filename := range c.Audit.LocalGoFiles.SortedSlice() {
		file, err := NewFile(c.Audit, filename)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		file.RenameIfGoFileNamedAfterRootPackage(c.Op.From.LocalImportPath, c.Op.To.LocalImportPath, c.Audit.LocalGoFiles)

		updatedNode := file.Apply(func(cursor *astutil.Cursor) bool {
			err := localFilePreApply(c.Audit, file, c.Op, cursor)
			return !cage_errors.Append(&errs, errors.WithStack(err))
		})

		stageFileBytes, err := file.GetNodeBytes(updatedNode)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		stageFileBytes = file.RenamePackageClause(fromLocalFilePath, c.Op.From.LocalImportPath, c.Op.To.LocalImportPath, stageFileBytes)

		stageFileBytes = c.rewriteLocalFileText(filename, stageFileBytes)

		// Reformat in case import strings need to be re-sorted.
		if formatted, err := format.Source(stageFileBytes); err == nil {
			stageFileBytes = formatted
		} else {
			// Tolerate types of files such as fixture with intended syntax errors.
			c.Plan.GoFormatErr = append(c.Plan.GoFormatErr, CopyFileError{Name: filename, Err: err.Error()})
		}

		toAbsPath, toRelPath := file.LocalDestPaths(c.Op)
		if skip, err := c.skipWrite(toAbsPath, bytes.NewReader(stageFileBytes)); err == nil {
			c.logFileActivity(toAbsPath, "added to stage as a local implementation file")
			if skip {
				c.logFileActivity(toAbsPath, skipOverwriteLogMsg)
			}
		} else {
			cage_errors.Append(&errs, errors.WithStack(err))
		}

		fd, err := c.Stage.CreateFileAll(toRelPath, os.FileMode(newFileMode), os.FileMode(newDirMode))
		if cage_errors.Append(&errs, errors.Wrapf(err, "failed to create stage file [%s]", filename)) {
			continue
		}

		_, err = fd.Write(stageFileBytes)
		cage_errors.Append(&errs, errors.Wrapf(err, "failed to write to stage file [%s]", filename))
	}

	return errs
}

// localGoTestFiles adds Go test files from Op.From.GoFilePath to the stage.
func (c *Copier) localGoTestFiles() (errs []error) {
	fromLocalFilePath := FromAbs(c.Op, c.Op.From.LocalFilePath)

	for _, filename := range c.Audit.LocalGoTestFiles.SortedSlice() {
		file, err := NewFile(c.Audit, filename)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		file.RenameIfGoFileNamedAfterRootPackage(c.Op.From.LocalImportPath, c.Op.To.LocalImportPath, c.Audit.LocalGoTestFiles)

		updatedNode := file.Apply(func(cursor *astutil.Cursor) bool {
			err := localFilePreApply(c.Audit, file, c.Op, cursor)
			return !cage_errors.Append(&errs, errors.WithStack(err))
		})

		stageFileBytes, err := file.GetNodeBytes(updatedNode)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		stageFileBytes = file.RenamePackageClause(fromLocalFilePath, c.Op.From.LocalImportPath, c.Op.To.LocalImportPath, stageFileBytes)

		stageFileBytes = c.rewriteLocalFileText(filename, stageFileBytes)

		// Reformat in case import strings need to be re-sorted.
		if formatted, err := format.Source(stageFileBytes); err == nil {
			stageFileBytes = formatted
		} else {
			// Tolerate types of files such as fixture with intended syntax errors.
			c.Plan.GoFormatErr = append(c.Plan.GoFormatErr, CopyFileError{Name: filename, Err: err.Error()})
		}

		toAbsPath, toRelPath := file.LocalDestPaths(c.Op)
		if skip, err := c.skipWrite(toAbsPath, bytes.NewReader(stageFileBytes)); err == nil {
			c.logFileActivity(toAbsPath, "added to stage as a local test file")
			if skip {
				c.logFileActivity(toAbsPath, skipOverwriteLogMsg)
			}
		} else {
			cage_errors.Append(&errs, errors.WithStack(err))
		}

		fd, err := c.Stage.CreateFileAll(toRelPath, os.FileMode(newFileMode), os.FileMode(newDirMode))
		if cage_errors.Append(&errs, errors.Wrapf(err, "failed to create stage file [%s]", filename)) {
			continue
		}

		_, err = fd.Write(stageFileBytes)
		cage_errors.Append(&errs, errors.Wrapf(err, "failed to write to stage file [%s]", filename))
	}
	return errs
}

// localCopyOnlyFiles adds Op.From.CopyOnlyFilePath Go/non-Go files to stage.
func (c *Copier) localCopyOnlyFiles() (errs []error) {
	// Reuse CopyOnlyFilePath logic to copy GoDescendantFilePath matches
	c.Audit.LocalCopyOnlyFiles.AddSet(c.Audit.LocalGoDescendantFiles)

	for _, filename := range c.Audit.LocalCopyOnlyFiles.SortedSlice() {
		file, err := NewFile(c.Audit, filename)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		file.RenameIfGoFileNamedAfterRootPackage(
			c.Op.From.LocalImportPath,
			c.Op.To.LocalImportPath,
			c.Audit.LocalCopyOnlyFiles,
		)

		stageFileBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to read file [%s] for updating", filename))
			continue
		}

		stageFileBytes = c.rewriteLocalFileText(filename, stageFileBytes)

		// Reformat in case import strings need to be re-sorted.
		if cage_filepath.IsGoFile(filename) {
			if formatted, err := format.Source(stageFileBytes); err == nil {
				stageFileBytes = formatted
			} else {
				// Tolerate types of files such as fixture with intended syntax errors.
				c.Plan.GoFormatErr = append(c.Plan.GoFormatErr, CopyFileError{Name: filename, Err: err.Error()})
			}
		}

		toAbsPath, toRelPath := file.LocalDestPaths(c.Op)
		if skip, err := c.skipWrite(toAbsPath, bytes.NewReader(stageFileBytes)); err == nil {
			if c.Audit.LocalGoDescendantFiles.Contains(filename) {
				c.logFileActivity(toAbsPath, "added to stage as a local GoDescendantFilePath file")
			} else {
				c.logFileActivity(toAbsPath, "added to stage as a local CopyOnlyFilePath file")
			}
			if skip {
				c.logFileActivity(toAbsPath, skipOverwriteLogMsg)
			}
		} else {
			cage_errors.Append(&errs, errors.WithStack(err))
		}

		fd, err := c.Stage.CreateFileAll(toRelPath, os.FileMode(newFileMode), os.FileMode(newDirMode))
		if cage_errors.Append(&errs, errors.Wrapf(err, "failed to create stage file [%s]", filename)) {
			continue
		}

		err = fd.Chmod(file.Mode)
		if cage_errors.Append(&errs, errors.Wrapf(err, "failed to set mode of stage file [%s]", filename)) {
			continue
		}

		_, err = fd.Write(stageFileBytes)
		cage_errors.Append(&errs, errors.Wrapf(err, "failed to write to stage file [%s]", filename))
	}
	return errs
}

// usedDepGoFiles adds Op.Dep.From.FilePath non-test Go files.
func (c *Copier) usedDepGoFiles() (errs []error) {
	// Ingress does not attempt to update the origin's Ops.Dep packages by design. See README.md for the rationale.
	if c.Op.Ingress {
		return []error{}
	}

	for _, filename := range c.Audit.UsedDepGoFiles.SortedSlice() {
		if c.Audit.isLocalFile(filename) { // e.g. Ops.From.GoDescendantFilePath is non-empty
			continue
		}

		var dep Dep

		for _, d := range c.Op.Dep {
			if strings.HasPrefix(filename, filepath.Join(c.Op.From.ModuleFilePath, d.From.FilePath)) {
				dep = d
				break
			}
		}

		if dep.From.ImportPath == "" {
			cage_errors.Append(&errs, errors.Errorf("failed to match file [%s] with its config", filename))
			continue
		}

		file, err := NewPrunableFile(c.Audit, filename)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		file.RenameIfGoFileNamedAfterRootPackage(dep.From.ImportPath, dep.To.ImportPath, c.Audit.UsedDepGoFiles)

		prunedGlobalIds, astErrs := file.UpdateDepAst(c.Audit, c.Op)
		if len(astErrs) > 0 {
			for n := range astErrs {
				cage_errors.Append(&errs, errors.WithStack(astErrs[n]))
			}
			continue
		}
		c.Plan.PruneGlobalIds = append(c.Plan.PruneGlobalIds, prunedGlobalIds.Slice()...)

		stageFileBytes, err := file.GetNodeBytes(file.DecoratedFile)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		stageFileBytes = file.RenamePackageClause(FromAbs(c.Op, dep.From.FilePath), dep.From.ImportPath, dep.To.ImportPath, stageFileBytes)
		stageFileBytes = c.rewriteDepFileText(filename, stageFileBytes)

		// Reformat in case import strings need to be re-sorted.
		if formatted, err := format.Source(stageFileBytes); err == nil {
			stageFileBytes = formatted
		} else {
			// Tolerate types of files such as fixture with intended syntax errors.
			c.Plan.GoFormatErr = append(c.Plan.GoFormatErr, CopyFileError{Name: filename, Err: err.Error()})
		}

		toAbsPath, toRelPath := file.DepDestPaths(c.Op, dep)
		if skip, err := c.skipWrite(toAbsPath, bytes.NewReader(stageFileBytes)); err == nil {
			c.logFileActivity(toAbsPath, "added to stage as an Ops.Dep implementation file")
			if skip {
				c.logFileActivity(toAbsPath, skipOverwriteLogMsg)
			}
		} else {
			cage_errors.Append(&errs, errors.WithStack(err))
		}

		// stageRelPath := filepath.Join(dep.To.FilePath, toRelPath)
		fd, err := c.Stage.CreateFileAll(toRelPath, os.FileMode(newFileMode), os.FileMode(newDirMode))
		if cage_errors.Append(&errs, errors.Wrapf(err, "failed to create stage file [%s]", filename)) {
			continue
		}

		_, err = fd.Write(stageFileBytes)
		cage_errors.Append(&errs, errors.Wrapf(err, "failed to write to stage file [%s]", filename))
	}
	return errs
}

// depGoTestFiles adds Go test files from Op.Dep.From.GoFilePath to the stage.
func (c *Copier) depGoTestFiles() (errs []error) {
	// Ingress does not attempt to update the origin's Ops.Dep packages by design. See README.md for the rationale.
	if c.Op.Ingress {
		return []error{}
	}

	for _, filename := range c.Audit.DepGoTestFiles.SortedSlice() {
		if c.Audit.isLocalFile(filename) { // e.g. Ops.From.GoDescendantFilePath is non-empty
			continue
		}

		var dep Dep

		for _, d := range c.Op.Dep {
			if strings.HasPrefix(filename, filepath.Join(c.Op.From.ModuleFilePath, d.From.FilePath)) {
				dep = d
				break
			}
		}

		if dep.From.ImportPath == "" {
			cage_errors.Append(&errs, errors.Errorf("failed to match file [%s] with its config", filename))
			continue
		}

		file, err := NewFile(c.Audit, filename)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		file.RenameIfGoFileNamedAfterRootPackage(dep.From.ImportPath, dep.To.ImportPath, c.Audit.DepGoTestFiles)

		updatedNode := file.Apply(func(cursor *astutil.Cursor) bool {
			if err := file.RewriteImportsInNode(c.Audit, c.Op, cursor, false); err != nil {
				errs = append(errs, errors.WithStack(err))
				return false
			}

			return true
		})

		stageFileBytes, err := file.GetNodeBytes(updatedNode)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		stageFileBytes = file.RenamePackageClause(FromAbs(c.Op, dep.From.FilePath), dep.From.ImportPath, dep.To.ImportPath, stageFileBytes)
		stageFileBytes = c.rewriteDepFileText(filename, stageFileBytes)

		// Reformat in case import strings need to be re-sorted.
		if formatted, err := format.Source(stageFileBytes); err == nil {
			stageFileBytes = formatted
		} else {
			// Tolerate types of files such as fixture with intended syntax errors.
			c.Plan.GoFormatErr = append(c.Plan.GoFormatErr, CopyFileError{Name: filename, Err: err.Error()})
		}

		toAbsPath, toRelPath := file.DepDestPaths(c.Op, dep)
		if skip, err := c.skipWrite(toAbsPath, bytes.NewReader(stageFileBytes)); err == nil {
			c.logFileActivity(toAbsPath, "added to stage as an Ops.Dep test file")
			if skip {
				c.logFileActivity(toAbsPath, skipOverwriteLogMsg)
			}
		} else {
			cage_errors.Append(&errs, errors.WithStack(err))
		}

		fd, err := c.Stage.CreateFileAll(toRelPath, os.FileMode(newFileMode), os.FileMode(newDirMode))
		if cage_errors.Append(&errs, errors.Wrapf(err, "failed to create stage file [%s]", filename)) {
			continue
		}

		_, err = fd.Write(stageFileBytes)
		cage_errors.Append(&errs, errors.Wrapf(err, "failed to write to stage file [%s]", filename))
	}
	return errs
}

// depCopyOnlyFiles adds Op.Dep.From.CopyOnlyFilePath Go/non-Go files to stage.
func (c *Copier) depCopyOnlyFiles() (errs []error) {
	// Ingress does not attempt to update the origin's Ops.Dep packages by design. See README.md for the rationale.
	if c.Op.Ingress {
		return []error{}
	}

	// Reuse CopyOnlyFilePath logic to copy GoDescendantFilePath matches
	c.Audit.DepCopyOnlyFiles.AddSet(c.Audit.DepGoDescendantFiles)

	for _, filename := range c.Audit.DepCopyOnlyFiles.SortedSlice() {
		var dep Dep

		for _, d := range c.Op.Dep {
			if strings.HasPrefix(filename, filepath.Join(c.Op.From.ModuleFilePath, d.From.FilePath)) {
				dep = d
				break
			}
		}

		if dep.From.ImportPath == "" {
			cage_errors.Append(&errs, errors.Errorf("failed to match file [%s] with its config", filename))
			continue
		}

		file, err := NewFile(c.Audit, filename)
		if cage_errors.Append(&errs, errors.WithStack(err)) {
			continue
		}

		file.RenameIfGoFileNamedAfterRootPackage(dep.From.ImportPath, dep.To.ImportPath, c.Audit.DepCopyOnlyFiles)

		stageFileBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to read file [%s] for updating", filename))
			continue
		}

		stageFileBytes = c.rewriteDepFileText(filename, stageFileBytes)

		// Reformat in case import strings need to be re-sorted.
		if cage_filepath.IsGoFile(filename) {
			if formatted, err := format.Source(stageFileBytes); err == nil {
				stageFileBytes = formatted
			} else {
				// Tolerate types of files such as fixture with intended syntax errors.
				c.Plan.GoFormatErr = append(c.Plan.GoFormatErr, CopyFileError{Name: filename, Err: err.Error()})
			}
		}

		toAbsPath, toRelPath := file.DepDestPaths(c.Op, dep)
		if skip, err := c.skipWrite(toAbsPath, bytes.NewReader(stageFileBytes)); err == nil {
			if c.Audit.DepGoDescendantFiles.Contains(filename) {
				c.logFileActivity(toAbsPath, "added to stage as an Ops.Dep GoDescendantFilePath file")
			} else {
				c.logFileActivity(toAbsPath, "added to stage as an Ops.Dep CopyOnlyFilePath file")
			}
			if skip {
				c.logFileActivity(toAbsPath, skipOverwriteLogMsg)
			}
		} else {
			cage_errors.Append(&errs, errors.WithStack(err))
		}

		// stageRelPath := filepath.Join(dep.To.FilePath, toRelPath)
		fd, err := c.Stage.CreateFileAll(toRelPath, os.FileMode(newFileMode), os.FileMode(newDirMode))
		if cage_errors.Append(&errs, errors.Wrapf(err, "failed to create stage file [%s]", filename)) {
			continue
		}

		err = fd.Chmod(file.Mode)
		if cage_errors.Append(&errs, errors.Wrapf(err, "failed to set mode of stage file [%s]", filename)) {
			continue
		}

		_, err = fd.Write(stageFileBytes)
		cage_errors.Append(&errs, errors.Wrapf(err, "failed to write to stage file [%s]", filename))
	}
	return errs
}

func (c *Copier) outputStage() (errs []error) {
	if outErrs := c.Stage.Output(); len(outErrs) > 0 {
		for n := range outErrs {
			cage_errors.Append(&errs, errors.WithStack(outErrs[n]))
		}
	}
	return errs
}

// moduleRequirements adds go.mod, go.sum, and vendor/ to the stage.
func (c *Copier) moduleRequirements() (errs []error) {
	if os.Getenv("TRANSPLANT_PPROF") == "1" {
		return []error{}
	}

	// Ingress does not attempt to update the origin's module by design. See README.md for the rationale.
	if c.Op.Ingress {
		return []error{}
	}

	if !c.ModuleRequire {
		return []error{}
	}

	if c.Op.From.ModuleFilePath == "" { // origin file tree is not a module
		return []error{}
	}

	executor := cage_exec.CommonExecutor{}
	stageGomodPath := c.Stage.Path("go.mod")
	stageGosumPath := c.Stage.Path("go.sum")
	originVendorPath := FromAbs(c.Op, "vendor")

	// In go1.12, "go get" can fail due to "go get: disabled by -mod=vendor".
	goGetEnv := os.Environ()
	for n := range goGetEnv {
		if strings.HasPrefix(goGetEnv[n], "GOFLAGS=") && strings.Contains(goGetEnv[n], "-mod=vendor") {
			goGetEnv[n] = strings.Replace(goGetEnv[n], "-mod=vendor", "", 1)
			break
		}
	}

	// Create the stage go.mod.

	cmd := exec.CommandContext(c.Ctx, "go", "mod", "init", c.Op.To.ModuleImportPath)
	cmd.Dir = c.Stage.Path()
	_, err := executor.Standard(c.Ctx, c.ProgressModule, c.ProgressModule, nil, cmd)
	if err != nil {
		return []error{errors.WithStack(err)}
	}

	// Ensure go.mod "require (...)" is populated.
	//
	// It's unclear how "go mod init" decides whether to populate it or not.

	cmd = exec.CommandContext(c.Ctx, "go", "mod", "tidy", "-v")
	cmd.Dir = c.Stage.Path()
	_, err = executor.Standard(c.Ctx, c.ProgressModule, c.ProgressModule, nil, cmd)
	if err != nil {
		return []error{errors.WithStack(err)}
	}

	if err := c.Stage.AddFileByName("go.mod"); err != nil {
		return []error{errors.WithStack(err)}
	}

	// Sync the stage's module dependencies with the origin's by comparing their go.mod versions.
	// For each misaligned dependency (e.g. the origin is using an older version than found during
	// the initial "go mod tidy"), use "go get" to update it.
	//
	// If this step was skipped, some dependencies would get updated to their newest releases every time
	// the egress copy is refreshed.
	//
	// This allows dependency upgrades to propagate from Op.From.ModuleFilePath, increasing the reproducibility
	// of the copy operation itself by reducing "magical" upgrades.

	// Collect the origin module's requirements.

	originGomodPath := FromAbs(c.Op, "go.mod")
	originGomod, err := cage_mod.NewModFromFile(originGomodPath)
	if err != nil {
		return []error{errors.Wrapf(err, "failed to parse the origin's go.mod [%s]", originGomodPath)}
	}

	// Collect the staged module's requirements.

	stageGomod, err := cage_mod.NewModFromFile(stageGomodPath)
	if err != nil {
		return []error{errors.Wrapf(err, "failed to parse the stage's go.mod [%s]", stageGomodPath)}
	}

	stageGomodRequires := stageGomod.Requires()

	if len(stageGomodRequires) > 0 {
		// Reset the stage go.mod/go.sum to avoid running "go get" with potentially newer requirements
		// in those files than the origin's so that the command's behavior is not affected by them.
		//
		// As of go1.12.9, if we have two modules which are identical except that one has a go.mod
		// with the latest available versions (via `go mod init`) and another has all older versions,
		// simply running "go get <path>@<older version>" in the former module will not lead to a
		// go.mod/go.sum identical to those in the module that always had older versions.
		//
		// In this example, all requirements are out-of-date except for golang.org/x/sync.
		//
		// require (
		//         github.com/mitchellh/mapstructure v1.1.2
		//         github.com/onsi/gomega v1.6.0
		//         github.com/pkg/errors v0.8.1
		//         github.com/stretchr/testify v1.4.0
		//         golang.org/x/sync v0.0.0-20190423024810-112230192c58
		// )
		//
		// After running "go get <path>@<older version>" on the first four dependencies, selecting
		// the versions seen below, "gopkg.in/yaml.v2 v2.2.2" is added as an indirect. According
		// to "go mod graph", it is present due to "origin gopkg.in/yaml.v2@v2.2.2". If the indirect
		// is to be added at all, it seems like it should be due to "github.com/onsi/gomega@v1.4.3 gopkg.in/yaml.v2@v2.2.1"
		// also found in the graph. "go mod tidy" does not remove it.
		//
		// require (
		//         github.com/mitchellh/mapstructure v1.1.0
		//         github.com/onsi/gomega v1.4.3
		//         github.com/pkg/errors v0.8.1
		//         github.com/stretchr/testify v1.3.0
		//         golang.org/x/sync v0.0.0-20190423024810-112230192c58
		//         gopkg.in/yaml.v2 v2.2.2 // indirect
		// )

		if err = cage_file.RemoveSafer(stageGomodPath); err != nil {
			cage_errors.Append(&errs, errors.Wrapf(err, "failed to reset stage go.mod [%s]", stageGomodPath))
			return errs
		}
		stageSumExists, _, stageSumExistsErr := cage_file.Exists(stageGosumPath)
		if stageSumExistsErr != nil {
			return []error{errors.Wrapf(stageSumExistsErr, "failed to check if [%s] exists", stageGosumPath)}
		}
		if stageSumExists {
			if err = cage_file.RemoveSafer(stageGosumPath); err != nil {
				cage_errors.Append(&errs, errors.Wrapf(err, "failed to reset stage go.sum [%s]", stageGosumPath))
				return errs
			}
		}

		cmd = exec.CommandContext(c.Ctx, "go", "mod", "init", c.Op.To.ModuleImportPath)
		cmd.Dir = c.Stage.Path()
		_, err := executor.Standard(c.Ctx, c.ProgressModule, c.ProgressModule, nil, cmd)
		if err != nil {
			return []error{errors.WithStack(err)}
		}

		var ranGoGet bool

		// Use "go get" to update the stage go.sum for each dependency in the stage which is at
		// a version which differs from Op.From.ModuleFilePath (because a newer version was found).
		for _, stageRequire := range stageGomodRequires {
			originRequire, found := originGomod.GetRequire(stageRequire.Path)

			if !found || originRequire.Version == stageRequire.Version {
				continue
			}

			// -m/-d: To avoid "build constraints exclude all Go files" errors
			//     https://github.com/golang/go/issues/33526
			//     https://github.com/golang/go/issues/29268
			goGetWorkaroundFlag := "-m"
			atLeastGo113, err := cage_runtime.VersionAtLeast("1.13")
			if err != nil {
				return []error{errors.WithStack(err)}
			}
			if atLeastGo113 {
				goGetWorkaroundFlag = "-d" // -m was removed in 1.13
			}

			getCmd := exec.CommandContext(c.Ctx, "go", "get", goGetWorkaroundFlag, "-v", stageRequire.Path+"@"+originRequire.Version)
			getCmd.Env = goGetEnv
			ranGoGet = true

			getCmd.Dir = c.Stage.Path()
			_, err = executor.Standard(c.Ctx, c.ProgressModule, c.ProgressModule, nil, getCmd)
			if err != nil {
				return []error{errors.WithStack(err)}
			}
		}

		// Clean up unused versions from go.sum that are no longer needed after the "go get" operations.
		if ranGoGet {
			cmd = exec.CommandContext(c.Ctx, "go", "mod", "tidy", "-v")
			cmd.Dir = c.Stage.Path()
			_, err = executor.Standard(c.Ctx, c.ProgressModule, c.ProgressModule, nil, cmd)
			if err != nil {
				return []error{errors.WithStack(err)}
			}
		}
	}

	// Append any `replace` directives, from the origin go.mod, which target dependencies found in the stage's
	// full dependency list.

	originReplaces := originGomod.Replaces()

	if len(originReplaces) > 0 {
		// Collect 'go list -m all' from the staged copy.
		stageGolist, err := cage_go_list.NewQuery(executor, c.Stage.Path()).AllModules().Run(c.Ctx)
		if err != nil {
			return []error{errors.Wrap(err, "failed to collect stage's full dependency list")}
		}

		var replaceStr strings.Builder

		// The line spacing doesn't need to be precise because it will be fixed by a later "go mod tidy".
		for _, replace := range originReplaces {
			if stageGolist.GetByPath(replace.Old) != nil {
				replaceStr.WriteString("\n")
				replaceStr.WriteString(replace.String())
			}
		}

		stageGomodFile, err := os.OpenFile(stageGomodPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return []error{errors.Wrapf(err, "failed to open stage's go.mod [%s]", stageGomodPath)}
		}

		if _, err = stageGomodFile.WriteString(replaceStr.String()); err != nil {
			return []error{errors.Wrapf(err, "failed to append 'replace' directives to stage go.mod [%s]", stageGomodPath)}
		}

		if err = cage_io.SyncClose(stageGomodFile, stageGomodPath); err != nil {
			return []error{errors.WithStack(err)}
		}
	}

	// Use the synchronized go.mod to vendor the dependencies.

	stageVendorPath := c.Stage.Path("vendor")

	if c.Op.From.Vendor {
		if len(errs) > 0 {
			return errs
		}

		c.Plan.GoModVendor = true

		cmd = exec.CommandContext(c.Ctx, "go", "mod", "vendor", "-v")
		cmd.Dir = c.Stage.Path()
		_, err = executor.Standard(c.Ctx, c.ProgressModule, c.ProgressModule, nil, cmd)
		if err != nil {
			return []error{errors.WithStack(err)}
		}

		// Register vendor/ in the stage.

		stageVendorExists, _, err := cage_file.Exists(stageVendorPath)
		if err != nil {
			return []error{errors.WithStack(err)}
		}
		if !stageVendorExists {
			return []error{errors.Errorf(
				"`go mod vendor` executed because [%s] detected in origin and GOFLAGS [%s] contains '-mod=vendor', but command did not create [%s] in stage",
				originVendorPath, os.Getenv("GOFLAGS"), stageVendorPath,
			)}
		}

		finder := cage_file.NewFinder().
			Dir(stageVendorPath).
			DirMatcher(
				cage_file_matcher.PopulatedDir,
			)
		if matches, matchesErr := finder.GetFilenameMatches(); matchesErr == nil {
			for _, vendorPath := range matches.SortedSlice() {
				if addErr := c.Stage.AddFileByName(strings.TrimPrefix(vendorPath, c.Stage.Path()+string(string(filepath.Separator)))); addErr != nil {
					cage_errors.Append(&errs, errors.Wrapf(addErr, "failed to add vendor file to stage [%s]", vendorPath))
					return errs
				}
			}
		} else {
			cage_errors.Append(&errs, errors.Wrapf(matchesErr, "failed to collect vendor filenames [%s]", stageVendorPath))
			return errs
		}
	}

	// Clean up unused versions from go.sum that may have been created by the initial tidy
	// and derived from a pre-sync go.mod.

	cmd = exec.CommandContext(c.Ctx, "go", "mod", "tidy", "-v")
	cmd.Dir = c.Stage.Path()
	_, err = executor.Standard(c.Ctx, c.ProgressModule, c.ProgressModule, nil, cmd)
	if err != nil {
		return []error{errors.WithStack(err)}
	}

	// Schedule the go.sum to be copied if the above "go mod tidy" usage has created it.

	stageSumExists, _, stageSumExistsErr := cage_file.Exists(stageGosumPath)
	if stageSumExistsErr != nil {
		return []error{errors.Wrapf(stageSumExistsErr, "failed to check if [%s] exists", stageGosumPath)}
	}
	if stageSumExists {
		if err = c.Stage.AddFileByName("go.sum"); err != nil {
			return []error{errors.WithStack(err)}
		}
	}

	// Now that replacements are ready in the stage, we can remove the old ones.

	if !c.Op.DryRun {
		existsPath := ToAbs(c.Op, c.Op.To.LocalFilePath, "go.mod")
		exists, _, existsErr := cage_file.Exists(existsPath)
		if cage_errors.Append(&errs, errors.Wrapf(existsErr, "failed to check if destination file [%s] exists", existsPath)) {
			return errs
		}
		if exists {
			if err = cage_file.RemoveSafer(existsPath); err != nil {
				cage_errors.Append(&errs, errors.Wrapf(err, "failed to remove destination file [%s]", existsPath))
				return errs
			}
		}

		existsPath = ToAbs(c.Op, c.Op.To.LocalFilePath, "go.sum")
		if stageSumExists {
			exists, _, existsErr := cage_file.Exists(existsPath)
			if cage_errors.Append(&errs, errors.Wrapf(existsErr, "failed to check if destination file [%s] exists", existsPath)) {
				return errs
			}
			if exists {
				if err = cage_file.RemoveSafer(ToAbs(c.Op, c.Op.To.LocalFilePath, "go.sum")); err != nil {
					cage_errors.Append(&errs, errors.Wrapf(err, "failed to remove destination file [%s]", existsPath))
					return errs
				}
			}
		}

		if c.Op.From.Vendor {
			existsPath = ToAbs(c.Op, "vendor")
			exists, _, existsErr := cage_file.Exists(existsPath)
			if cage_errors.Append(&errs, errors.Wrapf(existsErr, "failed to check if destination file [%s] exists", existsPath)) {
				return errs
			}
			if exists {
				if err = cage_file.RemoveAllSafer(existsPath); err != nil {
					cage_errors.Append(&errs, errors.Wrapf(err, "failed to remove destination file [%s]", existsPath))
					return errs
				}
			}
		}
	}

	return errs
}

func (c *Copier) copyStage() (errs []error) {
	copyCfg := cage_file_stage.CopyConfig{DryRun: c.Op.DryRun}

	if c.Op.Ingress {
		copyCfg.RemovableDirs = c.Audit.IngressRemovableDirs
		copyCfg.RemovableFiles = c.Audit.IngressRemovableFiles
	}

	stagePlan, copyErrs := c.Stage.Copy(c.Op.To.ModuleFilePath, copyCfg)

	if len(copyErrs) > 0 {
		for n := range copyErrs {
			cage_errors.Append(&errs, errors.WithStack(copyErrs[n]))
		}
	}

	c.Plan.Add = stagePlan.Add.SortedSlice()
	c.Plan.Overwrite = stagePlan.Overwrite.SortedSlice()
	c.Plan.Remove = stagePlan.Remove.SortedSlice()

	cage_strings.SortStable(c.Plan.OverwriteSkip)
	cage_strings.SortStable(c.Plan.PruneGlobalIds)
	cage_strings.SortStable(c.Plan.PruneGoFiles)

	return errs
}

// rewriteLocalFileText replaces all origin import path substrings with destination paths
// in an Ops.From file's text.
func (c *Copier) rewriteLocalFileText(fromAbsPath string, subject []byte) []byte {
	if c.Audit.LocalReplaceStringFiles.ImportPath.Contains(fromAbsPath) {
		return c.Audit.AllImportPathReplacer.InByte(subject)
	}
	return subject
}

// rewriteDepFileText replaces all origin import path substrings with destination paths
// in an Ops.Dep.From file's text.
func (c *Copier) rewriteDepFileText(fromAbsPath string, subject []byte) []byte {
	if c.Audit.LocalReplaceStringFiles.ImportPath.Contains(fromAbsPath) || c.Audit.DepReplaceStringFiles.ImportPath.Contains(fromAbsPath) {
		return c.Audit.DepImportPathReplacer.InByte(subject)
	}
	return subject
}

// localFilePreApply is the "pre" function parameter for dstutil.Apply operations on inspected Ops.From files.
func localFilePreApply(audit *Audit, f *File, op Op, cursor *astutil.Cursor) error {
	err := f.RewriteImportsInNode(audit, op, cursor, true)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
