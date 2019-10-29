// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package stage

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	cage_errors "github.com/codeactual/transplant/internal/cage/errors"
	cage_io "github.com/codeactual/transplant/internal/cage/io"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_file_matcher "github.com/codeactual/transplant/internal/cage/os/file/matcher"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

const (
	newDirMode = 0755
)

type Plan struct {
	// Add holds relative paths of files that exist in the stage but not the destination.
	Add *cage_strings.Set

	// Overwrite holds relative paths of files that exist in the stage and also the destination,
	// and the latter will be overwritten because the content has changed.
	Overwrite *cage_strings.Set

	// Remove holds relative paths of files that do not exist in the stage and but do in the destination.
	Remove *cage_strings.Set
}

type Stage struct {
	// basePath is the root of the stage's file tree.
	basePath string

	// names holds the names of the stage's files relative to Path.
	names *cage_strings.Set

	// objects indexes descriptor objects in the stage by their names relative to Path.
	objects map[string]*os.File

	// renames indexes new paths by old paths. All names are relative to path.
	//
	// During the copy process, each stage file whose relative path matches a key will be
	// created in the destination at the new relative path.
	renames map[string]string

	// overwriteSkips holds absolute paths of files in the destination tree which should not
	// be overwritten or removed, e.g. due to external detection that the pending write contains
	// the same content as the destination file. The paths must be descendants of the desttination
	// root path provided to Output calls.
	overwriteSkips *cage_strings.Set
}

func NewPlan() *Plan {
	return &Plan{
		Add:       cage_strings.NewSet(),
		Overwrite: cage_strings.NewSet(),
		Remove:    cage_strings.NewSet(),
	}
}

func NewStage(basePath string) *Stage {
	return &Stage{
		basePath:       basePath,
		names:          cage_strings.NewSet(),
		objects:        make(map[string]*os.File),
		overwriteSkips: cage_strings.NewSet(),
		renames:        make(map[string]string),
	}
}

// NewTempDirStage create the stage with ioutil.TempDir.
//
// For example, on Linux a prefix of "stage" would create a directory like "/tmp/stage725468197".
func NewTempDirStage(basePathPrefix string) (*Stage, error) {
	basePath, err := ioutil.TempDir("", basePathPrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create stage temp dir w/ prefix [%s]", basePathPrefix)
	}
	return NewStage(basePath), nil
}

// BasePath returns the path to the stage root.
func (s *Stage) BasePath() string {
	return s.basePath
}

type CopyConfig struct {
	// RemovableDirs holds destination absolute paths of all dirs which are allowed to be removed.
	// If it is nil, all destination dirs may be removed.
	RemovableDirs *cage_strings.Set

	// RemovableFiles holds destination absolute paths of all files which are allowed to be removed.
	// If it is nil, all destination files may be removed.
	RemovableFiles *cage_strings.Set

	DryRun bool
}

func (s *Stage) Output() (errs []error) {
	for _, stageRelPath := range s.names.SortedSlice() {
		stageAbsPath := s.Path(stageRelPath)
		fd := s.objects[stageRelPath]

		if err := fd.Sync(); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to sync stage file [%s]", stageAbsPath))
		}

		if err := fd.Close(); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to close stage file [%s]", stageAbsPath))
		}
	}
	return errs
}

// Output makes a copy of the stage's files in a destination directory.
//
// If the destination does not exist, it will be created.
//
// An optional CopyConfig will replace the default (zero) value.
func (s *Stage) Copy(dstPath string, cfgs ...CopyConfig) (plan *Plan, errs []error) {
	var config CopyConfig
	if len(cfgs) >= 1 {
		config = cfgs[0]
	}

	plan = NewPlan()

	// Collect an inventory of all files/directories which exist in the copy destination and may
	// get overwritten/removed when replaced by the staged contents.
	//
	// As staged contents are copied, their names are removed from the inventory sets. Those that
	// remains will be removed in the end to align with the stage.

	dstDirsSet := cage_strings.NewSet()
	dstFilesSet := cage_strings.NewSet()

	exists, _, existsErr := cage_file.Exists(dstPath)
	if existsErr != nil {
		errs = append(errs, errors.Wrapf(existsErr, "failed to check if [%s] exists", dstPath))
		return plan, errs
	}

	if exists {
		finder := cage_file.NewFinder().
			Dir(dstPath).
			DirMatcher(cage_file_matcher.PopulatedDir)

		dstDirs, dstDirsErr := finder.GetDirnameMatches()
		if dstDirsErr != nil {
			errs = append(errs, errors.Wrapf(dstDirsErr, "failed to scan [%s] for existing package directories", dstPath))
			return plan, errs
		}
		for _, d := range dstDirs.Slice() {
			if config.RemovableDirs == nil || config.RemovableDirs.Contains(d) {
				dstDirsSet.Add(d)
			}
		}

		dstFiles, dstFilesErr := finder.GetFilenameMatches()
		if dstFilesErr != nil {
			errs = append(errs, errors.Wrapf(dstFilesErr, "failed to scan [%s] for existing package files", dstPath))
			return plan, errs
		}
		for _, f := range dstFiles.Slice() {
			if config.RemovableFiles != nil && !config.RemovableFiles.Contains(f) {
				continue
			}
			if s.overwriteSkips.Contains(f) {
				continue
			}
			dstFilesSet.Add(f)
		}
	}

	// Copy the staged files to the destination. Update the Plan as we go.

	for _, stageRelPath := range s.names.SortedSlice() {
		// Apply Op.From.RenameFilePath/Op.Dep.From.RenameFilePath changes.

		stageAbsPath := s.Path(stageRelPath)

		fd, err := os.Open(stageAbsPath)
		if err != nil {
			cage_errors.Append(&errs, errors.Wrapf(err, "failed to open [%s]", stageAbsPath))
			continue
		}

		fromStat, err := fd.Stat()
		if err != nil {
			cage_errors.Append(&errs, errors.Wrapf(err, "failed to stat [%s]", stageAbsPath))
			continue
		}

		if _, ok := s.renames[stageRelPath]; ok {
			oldPath := filepath.Join(s.basePath, stageRelPath)
			newPath := filepath.Join(s.basePath, s.renames[stageRelPath])

			newPathDir := filepath.Dir(newPath)
			if mkdirErr := s.MkdirAll(newPathDir, newDirMode); mkdirErr != nil { // ensure the destination tree exists for os.Rename
				cage_errors.Append(&errs, errors.Wrapf(mkdirErr, "failed to make new stage dir [%s]", newPathDir))
				continue
			}

			if renameErr := os.Rename(oldPath, newPath); renameErr != nil {
				cage_errors.Append(&errs, errors.Wrapf(renameErr, "failed to rename [%s] to [%s]", oldPath, newPath))
				continue
			}
			stageRelPath = s.renames[stageRelPath]
		}

		// Update the plan based on copy outcome.

		toFilename := filepath.Join(dstPath, stageRelPath)

		exists, _, err := cage_file.Exists(toFilename)
		if err != nil {
			cage_errors.Append(&errs, errors.Wrapf(err, "failed to check if [%s] exists", toFilename))
			continue
		}

		if exists {
			if !s.overwriteSkips.Contains(toFilename) {
				plan.Overwrite.Add(toFilename)
			}
		} else {
			plan.Add.Add(toFilename)
		}

		dstFilesSet.Remove(toFilename)
		dstDirsSet.Remove(filepath.Dir(toFilename))

		if config.DryRun {
			continue
		}

		if !s.overwriteSkips.Contains(toFilename) {
			// Remove the file separately, instead of just O_TRUNC, so the stage file's mode is always propagated.
			if exists && cage_errors.Append(&errs, errors.WithStack(cage_file.RemoveSafer(toFilename))) {
				continue
			}
		}

		// Add/overwrite the destination file.

		if s.overwriteSkips.Contains(toFilename) {
			continue
		}

		toFile, err := cage_file.CreateFileAll(toFilename, os.O_RDWR, fromStat.Mode(), os.FileMode(newDirMode))
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to create file copy [%s]", toFilename))
			continue
		}
		defer cage_io.CloseOrStderr(toFile, toFilename)

		if err := cage_io.Rewind(fd); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to seek stage file [%s] to start", stageRelPath))
			continue
		}

		if _, err := io.Copy(toFile, fd); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to copy file [%s] to [%s]", stageRelPath, toFilename))
		}
	}

	// Remove files in the destination which are not in the stage.

	for _, dstFilename := range dstFilesSet.SortedSlice() {
		plan.Remove.Add(dstFilename)

		if config.DryRun {
			continue
		}

		if err := cage_file.RemoveSafer(dstFilename); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to remove [%s]", dstFilename))
			continue
		}
	}

	// Remove directories in the destination which are not in the stage.

	// Reverse order to remove descendants before ancestors.
	for _, dstDir := range dstDirsSet.SortedReverseSlice() {
		names, err := cage_file.Readdirnames(dstDir, 0)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to check if [%s] is empty", dstDir))
			continue
		}

		if len(names) > 0 {
			continue
		}

		plan.Remove.Add(dstDir)

		if config.DryRun {
			continue
		}

		if err = cage_file.RemoveSafer(dstDir); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to remove [%s]", dstDir))
			continue
		}
	}

	return plan, []error{}
}

// CreateFileAll creates a new stage file and all non-existent ancestor directories.
func (s *Stage) CreateFileAll(relPath string, fileMode, dirMode os.FileMode) (*os.File, error) {
	absPath := s.Path(relPath)

	fd, err := cage_file.CreateFileAll(absPath, os.O_RDWR, fileMode, dirMode)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create file [%s] in stage [%s]", relPath, s.basePath)
	}

	s.names.Add(relPath)
	s.objects[relPath] = fd

	return fd, nil
}

// MkdirAll creates a new stage directory and all non-existent ancestor directories.
func (s *Stage) MkdirAll(relPath string, dirMode os.FileMode) error {
	dirPath := s.Path(relPath)
	return errors.Wrapf(os.MkdirAll(dirPath, dirMode), "failed to make directory [%s]", dirPath)
}

// CopyFileAll adds a copy of a non-stage file into the stage at the selected relative path
// and all non-existent ancestor directories.
func (s *Stage) CopyFileAll(srcPath, dstRelPath string, fileMode, dirMode os.FileMode) (*os.File, error) {
	srcFd, err := os.Open(srcPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open file [%s]", srcPath)
	}
	defer cage_io.CloseOrStderr(srcFd, srcPath)

	dstFd, err := s.CreateFileAll(dstRelPath, fileMode, dirMode)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create file [%s] in stage [%s]", dstRelPath, s.basePath)
	}

	if _, err = io.Copy(dstFd, srcFd); err != nil {
		return nil, errors.Wrapf(err, "failed to copy file [%s] as [%s] to stage [%s]", srcPath, dstRelPath, s.basePath)
	}

	return dstFd, nil
}

// AddFileByName registers a stage file for copying.
//
// It is similar to AddFileByObject except that it opens the file to obtain the object/descriptor.
func (s *Stage) AddFileByName(relPath string) error {
	fd, err := os.Open(s.Path(relPath))
	if err != nil {
		return errors.Wrapf(err, "failed to open file [%s] to add to stage", relPath)
	}

	s.AddFileByObject(relPath, fd)

	return nil
}

// AddFileByObject a stage file for copying.
//
// It is similar to AddFileByName except that it excepts the object/descriptor as a parameter.
func (s *Stage) AddFileByObject(relPath string, fd *os.File) {
	s.names.Add(relPath)
	s.objects[relPath] = fd
}

// Rename registers a relative path in the stage to be a new relative path during the copy process.
func (s *Stage) Rename(fromRelPath, toRelPath string) {
	s.renames[fromRelPath] = toRelPath
}

// OverwriteSkip marks an absolute path in the destination tree that should not be overwritten or removed.
func (s *Stage) OverwriteSkip(destAbsPath string) {
	s.overwriteSkips.Add(destAbsPath)
}

func (s *Stage) Path(relPathParts ...string) string {
	for _, part := range relPathParts { // safety check
		if strings.Contains(part, "..") {
			panic(fmt.Sprintf("Stage.Path(%q) received relative path parts that may elevator above the base path", relPathParts))
		}
	}
	return filepath.Join(append([]string{s.basePath}, relPathParts...)...)
}
