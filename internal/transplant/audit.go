// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant

import (
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	std_packages "golang.org/x/tools/go/packages"

	"github.com/codeactual/transplant/cmd/transplant/why"
	cage_pkgs "github.com/codeactual/transplant/internal/cage/go/packages"
	cage_dag "github.com/codeactual/transplant/internal/cage/graph/dag"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_file_matcher "github.com/codeactual/transplant/internal/cage/os/file/matcher"
	cage_filepath "github.com/codeactual/transplant/internal/cage/path/filepath"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// ReplaceStringFiles holds file absolute paths found via Ops.From.ReplaceString/Ops.Dep.From.ReplaceString
// config patterns.
type ReplaceStringFiles struct {
	// ImportPath holds Ops.From.ReplaceString.ImportPath/Ops.Dep.From.ReplaceString.ImportPath matches.
	ImportPath *cage_strings.Set
}

func NewReplaceStringFiles() *ReplaceStringFiles {
	return &ReplaceStringFiles{
		ImportPath: cage_strings.NewSet(),
	}
}

// ConfigMatcher holds the file/dir matchers computed from FilePathQuery config patterns.
type ConfigMatcher struct {
	// BaseFilePath is the original base path selected when creating the full-path matchers
	BaseFilePath string

	CopyOnlyFilesFull   cage_file.FileMatcher
	CopyOnlyFilesSuffix cage_file.FileMatcher

	GoDescendantFilesFull   cage_file.FileMatcher
	GoDescendantFilesSuffix cage_file.FileMatcher

	InspectDirsFull   cage_file.DirMatcher
	InspectDirsSuffix cage_file.DirMatcher
}

func NewConfigMatcher(baseFilePath string) *ConfigMatcher {
	return &ConfigMatcher{BaseFilePath: baseFilePath}
}

// FromFinderInput holds config patterns for finding Ops.From.*FilePath/Ops.Dep.From.*FilePath matches.
type FromFinderInput struct {
	// BaseFilePath is the base path of Include/Exclude suffix-targeting matchers.
	BaseFilePath string

	// Include holds the file/dir matchers computed from Ops.From or Ops.Dep.From config.
	Include *ConfigMatcher

	// Exclude allows the search to avoid overlapping result sets.
	// When Include defines an Ops.From search, Exclude holds Ops.Dep.From matchers.
	// When Include defines an Ops.Dep.From search, it holds a Ops.From matcher.
	Exclude []*ConfigMatcher

	// ReplaceString specs match subsets of Include matches which are targeted for string replacements.
	ReplaceString ReplaceStringSpec
}

// FromFinderOutput holds file/directory absolute paths found via Ops.From.*FilePath/Ops.Dep.From.*FilePath
// config patterns.
type FromFinderOutput struct {
	AllFiles cage_file.DirToFinderMatchFiles

	CopyOnlyFiles     *cage_strings.Set
	GoDescendantFiles *cage_strings.Set

	InspectDirs       *cage_strings.Set
	InspectIgnoreDirs *cage_strings.Set

	ReplaceStringFiles *ReplaceStringFiles
}

func NewFromFinderOutput() *FromFinderOutput {
	return &FromFinderOutput{
		AllFiles: make(cage_file.DirToFinderMatchFiles),

		CopyOnlyFiles:     cage_strings.NewSet(),
		GoDescendantFiles: cage_strings.NewSet(),

		InspectDirs:       cage_strings.NewSet(),
		InspectIgnoreDirs: cage_strings.NewSet(),

		ReplaceStringFiles: NewReplaceStringFiles(),
	}
}

// Audit provides detail about the origin module collected during an initial analysis-only walk.
//
// It is used in the later replication pass to inform/limit copy operations.
type Audit struct {
	// AllDepGoFiles holds absolute paths to all files found recursively in all Dep.From.FilePath.
	AllDepGoFiles *cage_strings.Set

	// UsedDepGoFiles holds absolute paths to files directly/transitively depended on by LocalGoFiles
	// by examining the global identifier usage into the latter.
	//
	// All package files will be included even if LocalGoFiles only depends on one file. This excess
	// is currently used to ensure all init functions are included.
	//
	// When a filename is added, its path should be added to UsedDepImportPaths (depDirToImportPath may help).
	UsedDepGoFiles *cage_strings.Set

	// UsedDepImportPaths holds the import paths of packages directly/transitively depended on by LocalGoFiles.
	UsedDepImportPaths *cage_strings.Set

	// LocalGoFiles holds absolute paths of Ops.From.FilePath Go files.
	LocalGoFiles *cage_strings.Set

	// LocalGoTestFiles holds absolute paths of Ops.From.FilePath Go test files.
	LocalGoTestFiles *cage_strings.Set

	// LocalCopyOnlyFiles include all Ops.From.FilePath contents matching Ops.From.CopyOnlyFilePath.
	LocalCopyOnlyFiles *cage_strings.Set

	// LocalGoDescendantFiles include Ops.From.FilePath contents matching Ops.From.GoDescendantFilePath.
	//
	// They are filtered based on having an ancestor Go directory during the copy operation.
	LocalGoDescendantFiles *cage_strings.Set

	// LocalIncludeDirs holds absolute paths of Ops.From.FilePath dirs/files which are/match an
	// Ops.From.* inclusion pattern (that was not invalidated by a related exclusion pattern).
	LocalIncludeDirs *cage_strings.Set

	// AllDepDirs holds absolute paths of directories which match Ops.Dep.From.GoFilePath queries.
	//
	// It differs from the narrower scope of UsedDepGoFiles and covers directories which may include
	// packages which are not dependencies of LocalGoFiles.
	AllDepDirs *cage_strings.Set

	// UnconfiguredDirs holds absolute paths of directories whose packages were imported into Ops.From
	// or Ops.Dep.From packages but were not covered by those configuration.
	//
	// It is indexed by the import path of the packages which perform the imports.
	UnconfiguredDirs map[string]*cage_strings.Set

	// UnconfiguredDirImporters holds the import path keys of UnconfiguredDirs.
	UnconfiguredDirImporters *cage_strings.Set

	// DepCopyOnlyFiles include all Ops.Dep.From.FilePath contents matching Ops.Dep.From.CopyOnlyFilePath.
	DepCopyOnlyFiles *cage_strings.Set

	// DepGoDescendantFiles include Ops.Dep.From.FilePath contents matching Ops.Dep.From.GoDescendantFilePath.
	//
	// They are filtered based on having an ancestor Go directory during the copy operation.
	DepGoDescendantFiles *cage_strings.Set

	// DirectDepImportsIntoLocal enumerates the Ops.Dep.From packages directly imported into Ops.From packages.
	DirectDepImportsIntoLocal *cage_pkgs.ImportList

	// AllDepImportsIntoLocal enumerates the Ops.Dep.From packages directly or transitively imported into
	// Ops.From packages. It includes all elements of DirectDepImportsIntoLocal.
	AllDepImportsIntoLocal *cage_pkgs.ImportList

	// DepGlobalIdUsageDag graphs usage of global identifiers in Ops.Dep packages (except for their
	// init functions) to support pruning the latter during a refactor operation.
	//
	// It graphs connections between LocalGoFiles (as LocalGoFilesDagRoot) and all identifiers observed during
	// a recursive walk of global functions/methods in the Ops.Dep packages.
	DepGlobalIdUsageDag cage_dag.Graph

	// LocalGoFilesDagRoot represents all LocalGoFiles in the DepGlobalIdUsageDag.
	//
	// A Ops.Dep global can be pruned if there is no path from this root to the global in either DAG.
	LocalGoFilesDagRoot cage_pkgs.GlobalId

	// DepGoTestFiles holds absolute paths to files which were found in the same directory as a
	// Ops.Dep package which is directly/transitively used by Ops.Dep packages that are
	// directly/transitively used by LocalGoFiles.
	DepGoTestFiles *cage_strings.Set

	// LocalInspectDirs holds the Ops.From.FilePath directories provided to Inspector for
	// loading by x/tools/go/packages.
	LocalInspectDirs *cage_strings.Set

	// DepInspectDirs holds the Ops.Dep.From.FilePath directories provided to Inspector for
	// loading by x/tools/go/packages.
	DepInspectDirs *cage_strings.Set

	// UsedDepExports indexes used Ops.Dep global identifiers first by import path, then global name.
	UsedDepExports map[string]map[string]cage_pkgs.GlobalId

	// IngressRemovableDirs holds (egress) Ops.From.FilePath dirs which may be removed during ingress because
	// they match config patterns of the egress operation.
	IngressRemovableDirs *cage_strings.Set

	// IngressRemovableFiles holds (egress) Ops.From.FilePath files which may be removed during ingress because
	// they match config patterns of the egress operation.
	IngressRemovableFiles *cage_strings.Set

	// LocalReplaceStringFiles holds Ops.From.ReplaceString matches.
	LocalReplaceStringFiles *ReplaceStringFiles

	// DepReplaceStringFiles holds Ops.Dep.From.ReplaceString matches.
	DepReplaceStringFiles *ReplaceStringFiles

	// Progress receives messages describing analysis steps and runtimes.
	Progress io.Writer

	// WhyLog if non-nil will receive updates which support `{egress,ingress} file` queries.
	WhyLog why.Log

	// AllImportPathReplacer replaces Ops.From.LocalImportPath and all Ops.Dep.From.ImportPath substrings
	// with their To counterparts.
	AllImportPathReplacer *cage_strings.ReplaceSet

	// DepImportPathReplacer replaces Ops.Dep.From.ImportPath substrings with its To counterpart.
	DepImportPathReplacer *cage_strings.ReplaceSet

	// inspectedDirToDep indexes Dep configs by the directories which they selected for inclusion via
	// Dep.From.GoFilePath.
	inspectedDirToDep map[string]*Dep

	// inspector provides AST analysis to determine import/identifier dependencies.
	inspector *cage_pkgs.Inspector

	// inspectIgnoreDirs holds absolute paths of directories with files that match {CopyOnly,GoDescendant}FilePath
	// in order to omit them from inspection such as AST walks.
	//
	// It allows those patterns to take precedence in order to define sparse inclusions,
	// e.g. one-off inclusion in a directory covered by GoFilePath.
	inspectIgnoreDirs *cage_strings.Set

	// localConfigMatcher is computed from Ops.From FilePathQuery patterns.
	localConfigMatcher *ConfigMatcher

	// origLocalConfigMatcher is ingress-only and computed from Ops.From FilePathQuery patterns
	// (pre-reversal of From/To direction).
	origLocalConfigMatcher *ConfigMatcher

	// depConfigMatcher is computed from Ops.Dep.From FilePathQuery patterns. It is indexed by
	// Ops.Dep.From.FilePath values.
	depConfigMatcher map[string]*ConfigMatcher

	// op defines the external refactor operation which configures, and will be informed by, this audit.
	//
	// In egress mode, e.g. when the Audit is created by the `egress {file,run}` commands, it reflects
	// the config file. But in ingress mode, From/To paths are reversed to reflect the copy direction.
	op Op

	// origOp is identical to the op field except that it always reflects the config file instead of
	// the product of any direction-specific changes made by NewEgressAudit/NewIngressAudit.
	origOp Op

	// pkgCache speeds up ast.Package fetches identified by import paths.
	pkgCache *cage_pkgs.Cache

	// usedDepGlobalIdStr holds GlobalId.String() values of all Ops.Dep identifiers used directly/transitively
	// by LocalGoFiles. It supports pruning decisions.
	usedDepGlobalIdStr *cage_strings.Set
}

func newAudit(op Op) *Audit {
	a := &Audit{
		Progress: ioutil.Discard,

		op:     op,
		origOp: op,
	}

	a.pkgCache = cage_pkgs.NewCache()

	a.init()

	return a
}

func NewEgressAudit(op Op) *Audit {
	a := newAudit(op)
	a.op.finalizeEgress()

	a.AllImportPathReplacer = AllImportPathReplacer(a.op)
	a.DepImportPathReplacer = DepImportPathReplacer(a.op)
	return a
}

func NewIngressAudit(op Op) *Audit {
	a := newAudit(op)
	a.op.finalizeIngress()

	a.AllImportPathReplacer = AllImportPathReplacer(a.op)
	a.DepImportPathReplacer = DepImportPathReplacer(a.op)

	return a
}

func (a *Audit) init() {
	a.AllDepDirs = cage_strings.NewSet()
	a.AllDepGoFiles = cage_strings.NewSet()

	a.UnconfiguredDirs = make(map[string]*cage_strings.Set)
	a.UnconfiguredDirImporters = cage_strings.NewSet()

	a.UsedDepGoFiles = cage_strings.NewSet()
	a.DepGoTestFiles = cage_strings.NewSet()
	a.UsedDepExports = make(map[string]map[string]cage_pkgs.GlobalId)
	a.UsedDepImportPaths = cage_strings.NewSet()

	a.LocalGoFiles = cage_strings.NewSet()
	a.LocalGoTestFiles = cage_strings.NewSet()

	a.LocalIncludeDirs = cage_strings.NewSet()

	a.DepCopyOnlyFiles = cage_strings.NewSet()
	a.LocalCopyOnlyFiles = cage_strings.NewSet()

	a.DepGoDescendantFiles = cage_strings.NewSet()
	a.LocalGoDescendantFiles = cage_strings.NewSet()

	a.DepInspectDirs = cage_strings.NewSet()
	a.LocalInspectDirs = cage_strings.NewSet()

	a.DepGlobalIdUsageDag = cage_dag.NewGraph()

	a.DirectDepImportsIntoLocal = cage_pkgs.NewImportList()
	a.AllDepImportsIntoLocal = cage_pkgs.NewImportList()

	a.LocalGoFilesDagRoot = cage_pkgs.NewGlobalId(
		"LocalGoFilesDagRoot",
		"LocalGoFilesDagRoot",
		"LocalGoFilesDagRoot",
		"LocalGoFilesDagRoot",
	)

	a.usedDepGlobalIdStr = cage_strings.NewSet()

	a.addDepGlobalUsageVertex(a.LocalGoFilesDagRoot)

	a.IngressRemovableDirs = cage_strings.NewSet()
	a.IngressRemovableFiles = cage_strings.NewSet()

	a.LocalReplaceStringFiles = NewReplaceStringFiles()
	a.DepReplaceStringFiles = NewReplaceStringFiles()

	a.inspectedDirToDep = make(map[string]*Dep)

	a.inspectIgnoreDirs = cage_strings.NewSet()
}

func (a *Audit) Op() Op {
	return a.op
}

// Generate examines all files selected in the current Operation and collects details about imports
// and criteria for pruning packages during the copy stage.

// Multiple errors may be returned due to accumulation during operations like ast.Inspect,
// or in cases where a full report of all detected problems might assist bulk remediation
// instead of one-at-a-time.
func (a *Audit) Generate() (errs []error) {
	if len(filepath.SplitList(build.Default.GOPATH)) > 1 {
		return []error{errors.New("multiple GOPATH directories is not supported")}
	}

	steps := []struct {
		title       string
		f           func() []error
		time        time.Duration
		ingressSkip bool
		egressSkip  bool
	}{
		{title: "validate/finalize config values", f: a.finalizeConfig},
		{title: "find Ops.From files", f: a.findLocalFiles},
		{title: "find Ops.Dep.From files", f: a.findDepFiles},
		{title: "find Ops.Dep.From packages transitively used by Ops.From", f: a.findUsedDepPkgs, ingressSkip: true},
		{title: "inspect files", f: a.inspectGoFiles},
		{title: "validate files", f: a.validateFiles},
		{title: "group Ops.From files", f: a.groupLocalGoFiles},
		{title: "collect transitive Ops.Dep global use by Ops.From", f: a.findDepUsage, ingressSkip: true},
		{title: "find Ops.From.GoDescendant files", f: a.findLocalGoDescendantFiles},
		{title: "find Ops.Dep.From.GoDescendant files", f: a.findDepGoDescendantFiles},
		{title: "find origin files eligible for removal during ingress", f: a.findIngressRemovableFiles, egressSkip: true},
	}

	for n := range steps {
		if a.op.Ingress && steps[n].ingressSkip {
			continue
		}
		if !a.op.Ingress && steps[n].egressSkip {
			continue
		}

		fmt.Fprintf(a.Progress, "audit [%s] ... ", steps[n].title)

		start := time.Now()
		if errs := steps[n].f(); len(errs) > 0 {
			return errs
		}
		steps[n].time = time.Since(start)

		fmt.Fprintf(a.Progress, "%s\n", steps[n].time)
	}

	return []error{}
}

// finalizeConfig performs Config.ReadFile-like checks and finalization that we want to
// only perform if an operation is actually attempted, rather than forcing the CLI
// to display all issues across all configured operations at the same time.
//
// It also allows a single fixture config file to serve multiple test cases, some of which
// require intentionally invalid Ops sections.
func (a *Audit) finalizeConfig() (errs []error) {
	fromLocalFilePath := FromAbs(a.op, a.op.From.LocalFilePath)

	// file-existence checks

	if a.op.From.LocalFilePath != "" {
		exists, _, existsErr := cage_file.Exists(fromLocalFilePath)
		if existsErr != nil {
			errs = append(errs, errors.Wrapf(existsErr, "failed to check if Ops[%s].From.LocalFilePath exists", a.op.Id))
		} else if !exists {
			errs = append(errs, errors.Errorf("Ops[%s].From.LocalFilePath not found [%s]", a.op.Id, fromLocalFilePath))
		}
	}

	for _, dep := range a.op.Dep {
		if dep.From.FilePath != "" {
			depFromFilePath := FromAbs(a.op, dep.From.FilePath)
			exists, _, existsErr := cage_file.Exists(depFromFilePath)
			if existsErr != nil {
				errs = append(errs, errors.Wrapf(existsErr, "failed to check if Ops[%s].Dep[%s].From.FilePath exists", a.op.Id, dep.From.ImportPath))
			} else if !exists {
				errs = append(errs, errors.Errorf("Ops[%s].Dep[%s].From.FilePath not found [%s]", a.op.Id, dep.From.ImportPath, depFromFilePath))
			}
		}
	}

	// During egress, we expect the rename target to exist in the origin.
	// During ingress, the renamed file in the copy may have been removed and we need to propagate the removal.
	if !a.op.Ingress {
		for _, p := range a.op.From.RenameFilePath {
			renameOld := FromAbs(a.op, p.Old)
			exists, _, existsErr := cage_file.Exists(renameOld)
			if existsErr != nil {
				errs = append(errs, errors.Wrapf(existsErr, "failed to check if [%s] exists", renameOld))
			} else if !exists {
				if a.op.Ingress {
					a.IngressRemovableFiles.Add(ToAbs(a.op, p.New))
				} else {
					errs = append(errs, errors.Errorf("Ops[%s].From.RenameFilePath [%s] not found", a.op.Id, renameOld))
				}
			}
		}
	}

	// Ops.From / Ops.Dep.From duplicate/overlap checks

	// Disallow Ops.Dep.From at/under Ops.From during egress because it indicates there's no distinction between
	// the local files of the project being exported and the first-party dependency at/under it. In short,
	// the dependency is already part of the project-local file tree and suggests this Ops.Dep element is redundant.
	// During ingress we expect the overlap, e.g. a library that is copied to the root of the destination module
	// tree and first-party dependencies to a path under ./internal.
	if !a.op.Ingress {
		for _, dep := range a.op.Dep {
			if dep.From.FilePath != "" {
				depFromFilePath := FromAbs(a.op, dep.From.FilePath)
				if strings.HasPrefix(depFromFilePath, fromLocalFilePath) {
					errs = append(errs, errors.Errorf("Ops[%s].Dep[%s].From.FilePath [%s] overlaps with Ops.From.FilePath [%s]. Consider updating Ops.From to include sets of files.", a.op.Id, dep.From.ImportPath, depFromFilePath, fromLocalFilePath))
				}
			}
		}
	}

	// Inter-Ops.Dep duplicate/overlap checks

	depDupeFromFilePath := cage_strings.NewSet()
	depDupeToFilePath := cage_strings.NewSet()

	depOverFromFilePath := cage_strings.NewSet()
	depOverToFilePath := cage_strings.NewSet()

	for s, subject := range a.op.Dep {
		for o, other := range a.op.Dep {
			if s == o {
				continue
			}

			if !depDupeFromFilePath.Contains(subject.From.FilePath) && subject.From.FilePath == other.From.FilePath {
				errs = append(errs, errors.Errorf("Ops[%s].Dep.From.FilePath [%s] is selected multiple times", a.op.Id, subject.From.FilePath))
				depDupeFromFilePath.Add(subject.From.FilePath)
				depOverFromFilePath.Add(subject.From.FilePath)
			}
			if !depDupeToFilePath.Contains(subject.To.FilePath) && subject.To.FilePath == other.To.FilePath {
				errs = append(errs, errors.Errorf("Ops[%s].Dep.To.FilePath [%s] is selected multiple times", a.op.Id, subject.To.FilePath))
				depDupeToFilePath.Add(subject.To.FilePath)
				depOverToFilePath.Add(subject.To.FilePath)
			}

			if !(depOverFromFilePath.Contains(subject.From.FilePath) || depOverFromFilePath.Contains(other.From.FilePath)) && strings.HasPrefix(subject.From.FilePath, other.From.FilePath) {
				errs = append(errs, errors.Errorf("Ops[%s].Dep.From.FilePath [%s] overlaps with another [%s]", a.op.Id, subject.From.FilePath, other.From.FilePath))
				depOverFromFilePath.AddSlice([]string{subject.From.FilePath, other.From.FilePath})
			}
			if !(depOverToFilePath.Contains(subject.To.FilePath) || depOverToFilePath.Contains(other.To.FilePath)) && strings.HasPrefix(subject.To.FilePath, other.To.FilePath) {
				errs = append(errs, errors.Errorf("Ops[%s].Dep.To.FilePath [%s] overlaps with another [%s]", a.op.Id, subject.To.FilePath, other.To.FilePath))
				depOverToFilePath.AddSlice([]string{subject.To.FilePath, other.To.FilePath})
			}
		}
	}

	// computed values

	a.localConfigMatcher = NewConfigMatcher(fromLocalFilePath)
	a.origLocalConfigMatcher = NewConfigMatcher(FromAbs(a.origOp, a.origOp.From.LocalFilePath))

	a.depConfigMatcher = make(map[string]*ConfigMatcher)
	for _, dep := range a.op.Dep {
		a.depConfigMatcher[dep.From.FilePath] = NewConfigMatcher(FromAbs(a.op, dep.From.FilePath))
	}

	// Create the matcher for including CopyOnlyFilePath matches (e.g. to check if an Ops.From match
	// conflicts with an Ops.Dep.From match). Resolve all the patterns so to avoid any patterns like
	// "**/*" from matching all candidates.
	a.localConfigMatcher.CopyOnlyFilesSuffix = MatchAnyFileRelPath(a.localConfigMatcher.BaseFilePath, cage_filepath.MatchAnyInput{
		Include: a.op.From.CopyOnlyFilePath.Include,
		Exclude: a.op.From.CopyOnlyFilePath.Exclude,
	})

	// Create the matcher for excluding CopyOnlyFilePath matches (e.g. to check if an Ops.From match
	// conflicts with an Ops.Dep.From match). Resolve all the patterns so to avoid any patterns like
	// "**/*" from matching all candidates.
	absQuery := a.op.From.CopyOnlyFilePath.Copy()
	absQuery.ResolveTo(a.localConfigMatcher.BaseFilePath)
	a.localConfigMatcher.CopyOnlyFilesFull = cage_file_matcher.MatchAnyFile(cage_filepath.MatchAnyInput{
		Include: absQuery.Include,
		Exclude: absQuery.Exclude,
	})

	// Create the matcher for including GoDescendantFilePath matches (e.g. to check if an Ops.From match
	// conflicts with an Ops.Dep.From match). Resolve all the patterns so to avoid any patterns like
	// "**/*" from matching all candidates.
	a.localConfigMatcher.GoDescendantFilesSuffix = MatchAnyFileRelPath(a.localConfigMatcher.BaseFilePath, cage_filepath.MatchAnyInput{
		Include: a.op.From.GoDescendantFilePath.Include,
		Exclude: a.op.From.GoDescendantFilePath.Exclude,
	})

	// Create the matcher for excluding GoDescendantFilePath matches (e.g. to check if an Ops.From match
	// conflicts with an Ops.Dep.From match). Resolve all the patterns so to avoid any patterns like
	// "**/*" from matching all candidates.
	absQuery = a.op.From.GoDescendantFilePath.Copy()
	absQuery.ResolveTo(a.localConfigMatcher.BaseFilePath)
	a.localConfigMatcher.GoDescendantFilesFull = cage_file_matcher.MatchAnyFile(cage_filepath.MatchAnyInput{
		Include: absQuery.Include,
		Exclude: absQuery.Exclude,
	})

	// Create the matcher for including GoFilePath matches (e.g. to check if an Ops.From match
	// conflicts with an Ops.Dep.From match). Resolve all the patterns so to avoid any patterns like
	// "**/*" from matching all candidates.
	a.localConfigMatcher.InspectDirsSuffix = MatchAnyDirRelPath(a.localConfigMatcher.BaseFilePath, cage_filepath.MatchAnyInput{
		Include: a.op.From.GoFilePath.Include,
		Exclude: a.op.From.GoFilePath.Exclude,
	})

	// Create the matcher for excluding GoFilePath matches (e.g. to check if an Ops.From match
	// conflicts with an Ops.Dep.From match). Resolve all the patterns so to avoid any patterns like
	// "**/*" from matching all candidates.
	//
	// - Append the base path of the search to the inclusion patterns to align with the behavior
	//   of suffix-based matches to always match against the base path if the latter is a candidate.
	//   Here we ensure the exclusion matcher will always exclude the base path if the latter is a candidate.
	absQuery = a.op.From.GoFilePath.Copy()
	absQuery.ResolveTo(a.localConfigMatcher.BaseFilePath)
	absQuery.Include = append(absQuery.Include, a.localConfigMatcher.BaseFilePath)
	a.localConfigMatcher.InspectDirsFull = cage_file_matcher.MatchAnyDir(cage_filepath.MatchAnyInput{
		Include: absQuery.Include,
		Exclude: absQuery.Exclude,
	})

	// Perform the same matcher creation steps for Ops.Dep configs as were performed above for Ops.From configs.
	for _, dep := range a.op.Dep {
		matcher := a.depConfigMatcher[dep.From.FilePath]

		absQuery = dep.From.CopyOnlyFilePath.Copy()
		absQuery.ResolveTo(matcher.BaseFilePath)
		matcher.CopyOnlyFilesFull = cage_file_matcher.MatchAnyFile(cage_filepath.MatchAnyInput{
			Include: absQuery.Include,
			Exclude: absQuery.Exclude,
		})
		matcher.CopyOnlyFilesSuffix = MatchAnyFileRelPath(matcher.BaseFilePath, cage_filepath.MatchAnyInput{
			Include: dep.From.CopyOnlyFilePath.Include,
			Exclude: dep.From.CopyOnlyFilePath.Exclude,
		})

		absQuery = dep.From.GoDescendantFilePath.Copy()
		absQuery.ResolveTo(matcher.BaseFilePath)
		matcher.GoDescendantFilesFull = cage_file_matcher.MatchAnyFile(cage_filepath.MatchAnyInput{
			Include: absQuery.Include,
			Exclude: absQuery.Exclude,
		})
		matcher.GoDescendantFilesSuffix = MatchAnyFileRelPath(matcher.BaseFilePath, cage_filepath.MatchAnyInput{
			Include: dep.From.GoDescendantFilePath.Include,
			Exclude: dep.From.GoDescendantFilePath.Exclude,
		})

		absQuery = dep.From.GoFilePath.Copy()
		absQuery.ResolveTo(matcher.BaseFilePath)
		absQuery.Include = append(absQuery.Include, matcher.BaseFilePath)
		matcher.InspectDirsFull = cage_file_matcher.MatchAnyDir(cage_filepath.MatchAnyInput{
			Include: absQuery.Include,
			Exclude: absQuery.Exclude,
		})
		matcher.InspectDirsSuffix = MatchAnyDirRelPath(matcher.BaseFilePath, cage_filepath.MatchAnyInput{
			Include: dep.From.GoFilePath.Include,
			Exclude: dep.From.GoFilePath.Exclude,
		})

		a.depConfigMatcher[dep.From.FilePath] = matcher
	}

	// origLocalConfigMatcher is used during ingress to match the local files in the origin.
	//
	// Note the use of Audit.origOp, instead of Audit.op, in configuring the matcher to target the module-relative
	// paths of the origin. Audit.op cannot be used because it includes updates made by NewIngressAudit.
	if a.op.Ingress {
		a.origLocalConfigMatcher.CopyOnlyFilesSuffix = MatchAnyFileRelPath(a.origLocalConfigMatcher.BaseFilePath, cage_filepath.MatchAnyInput{
			Include: a.origOp.From.CopyOnlyFilePath.Include,
			Exclude: a.origOp.From.CopyOnlyFilePath.Exclude,
		})
		a.origLocalConfigMatcher.GoDescendantFilesSuffix = MatchAnyFileRelPath(a.origLocalConfigMatcher.BaseFilePath, cage_filepath.MatchAnyInput{
			Include: a.origOp.From.GoDescendantFilePath.Include,
			Exclude: a.origOp.From.GoDescendantFilePath.Exclude,
		})
		a.origLocalConfigMatcher.InspectDirsSuffix = MatchAnyDirRelPath(a.origLocalConfigMatcher.BaseFilePath, cage_filepath.MatchAnyInput{
			Include: a.origOp.From.GoFilePath.Include,
			Exclude: a.origOp.From.GoFilePath.Exclude,
		})
	}

	return errs
}

func (a *Audit) findLocalFiles() (errs []error) {
	var excludes []*ConfigMatcher
	for _, e := range a.depConfigMatcher {
		excludes = append(excludes, e)
	}

	paths, errs := a.findFiles(FromFinderInput{
		BaseFilePath:  FromAbs(a.op, a.op.From.LocalFilePath),
		Include:       a.localConfigMatcher,
		Exclude:       excludes,
		ReplaceString: a.op.From.ReplaceString,
	})

	if len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.WithStack(errs[n])
		}
		return errs
	}

	for d, files := range paths.AllFiles {
		a.logFileActivity(d, "found under Op.From.LocalFilePath")
		for f := range files {
			a.logFileActivity(f, "found under Op.From.LocalFilePath")
		}
	}

	for _, f := range paths.CopyOnlyFiles.Slice() {
		a.logFileActivity(f, "matched Op.From.CopyOnlyFilePath")
		a.LocalIncludeDirs.Add(filepath.Dir(f))
	}
	a.LocalCopyOnlyFiles.AddSet(paths.CopyOnlyFiles)

	// This is only the initial list which matches the patterns but has not been filtered
	// based on a check for an ancestor Go dir which is scheduled for the copy.
	for _, f := range paths.GoDescendantFiles.Slice() {
		a.logFileActivity(f, "matched Op.From.GoDescendantFilePath pattern (before ancestor check)")
		a.LocalIncludeDirs.Add(filepath.Dir(f))
	}
	a.LocalGoDescendantFiles.AddSet(paths.GoDescendantFiles)

	for _, f := range paths.ReplaceStringFiles.ImportPath.Slice() {
		a.logFileActivity(f, "matched Op.From.ReplaceString.ImportPath")
		a.LocalReplaceStringFiles.ImportPath.Add(f)
		a.LocalIncludeDirs.Add(filepath.Dir(f))
	}

	for _, d := range paths.InspectDirs.Slice() {
		a.logFileActivity(d, "matched Op.From.GoFilePath")
	}
	a.LocalInspectDirs.AddSet(paths.InspectDirs)
	a.LocalIncludeDirs.AddSet(paths.InspectDirs)

	a.inspectIgnoreDirs.AddSet(paths.InspectIgnoreDirs)

	return errs
}

func (a *Audit) findLocalGoDescendantFiles() (errs []error) {
	goDirs := cage_strings.NewSet()

	for _, f := range a.LocalGoFiles.Slice() {
		goDirs.Add(filepath.Dir(f))
	}
	for _, f := range a.LocalGoTestFiles.Slice() {
		goDirs.Add(filepath.Dir(f))
	}

	for _, f := range a.LocalGoDescendantFiles.Slice() {
		var foundAncestor bool
		for _, goDir := range goDirs.Slice() {
			if strings.HasPrefix(f, goDir) {
				foundAncestor = true
				break
			}
		}
		if foundAncestor {
			a.logFileActivity(f, "matched Op.From.GoDescendantFilePath pattern (found ancestor)")
		} else {
			a.LocalGoDescendantFiles.Remove(f)
		}
	}

	return []error{}
}

func (a *Audit) findDepGoDescendantFiles() (errs []error) {
	if a.op.Ingress {
		a.DepGoDescendantFiles.Clear()
		return []error{}
	}

	goDirs := cage_strings.NewSet()

	for _, f := range a.UsedDepGoFiles.Slice() {
		if a.isLocalFile(f) {
			continue
		}
		goDirs.Add(filepath.Dir(f))
	}

	for _, f := range a.DepGoDescendantFiles.Slice() {
		var foundAncestor bool
		for _, goDir := range goDirs.Slice() {
			if strings.HasPrefix(f, goDir) {
				foundAncestor = true
				break
			}
		}
		if foundAncestor {
			a.logFileActivity(f, "matched Op.Dep.From.GoDescendantFilePath pattern (found ancestor)")
		} else {
			a.DepGoDescendantFiles.Remove(f)
		}
	}

	return []error{}
}

// findIngressRemovableFiles collects the absolute paths of files/dirs in the origin which are
// eligible for removal when propagating deletions made to the copy.
//
// Currently it only targets Ops.From files.
func (a *Audit) findIngressRemovableFiles() (errs []error) {
	// Note the use of Audit.origOp, instead of Audit.op, in configuring the matcher to target the module-relative
	// paths of the origin. Audit.op cannot be used because it includes updates made by NewIngressAudit.

	if !a.op.Ingress {
		return []error{}
	}

	paths, errs := a.findFiles(FromFinderInput{
		BaseFilePath: FromAbs(a.origOp, a.origOp.From.LocalFilePath),
		Include:      a.origLocalConfigMatcher,
	})

	if len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.WithStack(errs[n])
		}
		return errs
	}

	a.IngressRemovableFiles.AddSet(paths.CopyOnlyFiles)
	a.IngressRemovableFiles.AddSet(paths.GoDescendantFiles)

	for _, d := range paths.InspectDirs.SortedSlice() {
		if files := paths.AllFiles[d]; files != nil {
			for _, f := range files {
				a.IngressRemovableFiles.Add(f.AbsPath)
			}
		}
	}

	for _, f := range a.IngressRemovableFiles.Slice() {
		a.IngressRemovableDirs.Add(filepath.Dir(f))
	}

	return errs
}

func (a *Audit) findDepFiles() (errs []error) {
	if a.op.Ingress {
		return []error{}
	}

	for n, dep := range a.op.Dep {
		paths, errs := a.findFiles(FromFinderInput{
			BaseFilePath:  FromAbs(a.op, dep.From.FilePath),
			Include:       a.depConfigMatcher[dep.From.FilePath],
			Exclude:       []*ConfigMatcher{a.localConfigMatcher},
			ReplaceString: dep.From.ReplaceString,
		})

		if len(errs) > 0 {
			for n := range errs {
				errs[n] = errors.WithStack(errs[n])
			}
			return errs
		}

		for d, files := range paths.AllFiles {
			a.logFileActivity(d, "found under Dep.From.FilePath")
			for f := range files {
				a.logFileActivity(f, "found under Dep.From.FilePath")
			}
		}

		for _, f := range paths.CopyOnlyFiles.SortedSlice() {
			a.logFileActivity(f, "matched Dep.From.CopyOnlyFilePath")
		}
		a.DepCopyOnlyFiles.AddSet(paths.CopyOnlyFiles)

		// This is only the initial list which matches the patterns but has not been filtered
		// based on a check for an ancestor Go dir which is scheduled for the copy.
		for _, f := range paths.GoDescendantFiles.SortedSlice() {
			a.logFileActivity(f, "matched Dep.From.GoDescendantFilePath (before ancestor check)")
		}
		a.DepGoDescendantFiles.AddSet(paths.GoDescendantFiles)

		for _, f := range paths.ReplaceStringFiles.ImportPath.SortedSlice() {
			a.logFileActivity(f, "matched Dep.From.ReplaceString.ImportPath")
		}
		a.DepReplaceStringFiles.ImportPath.AddSet(paths.ReplaceStringFiles.ImportPath)

		for _, d := range paths.InspectDirs.SortedSlice() {
			a.logFileActivity(d, "matched Dep.From.GoFilePath")
			a.inspectedDirToDep[d] = &a.op.Dep[n]
		}
		a.AllDepDirs.AddSet(paths.InspectDirs)

		a.inspectIgnoreDirs.AddSet(paths.InspectIgnoreDirs)

		for _, files := range paths.AllFiles {
			for _, f := range files {
				// Collect all Ops.Dep.From Go files, even those which will not be inspected/copied.
				//
				// Use cage_filepath.GoFiles, instead of relying on findUsedDepPkgs's traversal, to include files which are not
				// found by a transitive import walk or are not included due to build tags/context.
				//
				// This inclusive set allows us to identify pruned files by comparing it with UsedDepGoFiles.
				if cage_filepath.IsGoFile(f.AbsPath) {
					a.AllDepGoFiles.Add(f.AbsPath)
				}
			}
		}
	}

	return errs
}

func (a *Audit) fileMatchExcluded(includeBaseFilePath string, excludes []*ConfigMatcher, f cage_file.FinderFile) (excluded bool, err error) {
	dir := filepath.Dir(f.AbsPath)
	var matcherErr error

	for _, matcher := range excludes {
		// Do not evaluate the exclusion matcher if its base path is an ancestor of the inclusion base path
		// defined by the caller. This prevents recursive exclusions from invalidating entire inclusion
		// pattners between the former is always a superset.
		if strings.HasPrefix(includeBaseFilePath, matcher.BaseFilePath+string(filepath.Separator)) {
			return false, nil
		}

		// We can pass a nil FinderMatchFiles parameter because the matcher only evaluates the path,
		// not the readdir results.
		excluded, matcherErr = matcher.InspectDirsFull(dir, nil)
		if matcherErr != nil {
			return false, errors.Wrapf(matcherErr, "failed to evaluate GoFilePath exclusions against [%s]", dir)
		}
		if excluded {
			break
		}

		excluded, matcherErr = matcher.CopyOnlyFilesFull(f)
		if matcherErr != nil {
			return false, errors.Wrapf(matcherErr, "failed to evaluate CopyOnlyFilePath exclusions against [%s]", f.AbsPath)
		}
		if excluded {
			break
		}

		excluded, matcherErr = matcher.GoDescendantFilesFull(f)
		if matcherErr != nil {
			return false, errors.Wrapf(matcherErr, "failed to evaluate GoDescendantFilePath exclusions against [%s]", f.AbsPath)
		}
		if excluded {
			break
		}
	}

	return excluded, nil
}

// findFiles returns Ops.From/Ops.Dep.From config pattern search results.
func (a *Audit) findFiles(cfg FromFinderInput) (paths *FromFinderOutput, errs []error) {
	paths = NewFromFinderOutput()

	// Collect all *.From files.

	var allFiles []cage_file.FinderFile

	finder := cage_file.NewFinder().
		Dir(cfg.BaseFilePath).
		DirMatcher(
			cage_file_matcher.PopulatedDir,
		)

	fileMatcher := MatchAnyFileRelPath(cfg.BaseFilePath, cage_filepath.MatchAnyInput{
		Include: []string{"**/*"},
	})

	matches, matchesErr := finder.GetFileObjectMatches(fileMatcher)
	if matchesErr != nil {
		errs = append(errs, errors.Wrapf(matchesErr, "failed to collect files under [%s]", cfg.BaseFilePath))
	}

	for _, f := range matches {
		excluded, excludeErr := a.fileMatchExcluded(cfg.Include.BaseFilePath, cfg.Exclude, f)
		if excludeErr != nil {
			errs = append(errs, errors.WithStack(excludeErr))
			continue
		}

		if excluded {
			continue
		}

		allFiles = append(allFiles, f)

		d := filepath.Dir(f.AbsPath)
		if paths.AllFiles[d] == nil {
			paths.AllFiles[d] = make(cage_file.FinderMatchFiles)
		}
		paths.AllFiles[d][f.AbsPath] = f
	}

	replaceImportPathMatcher := MatchAnyFileRelPath(cfg.BaseFilePath, cage_filepath.MatchAnyInput{
		Include: cfg.ReplaceString.ImportPath.Include,
		Exclude: cfg.ReplaceString.ImportPath.Exclude,
	})

	for _, f := range allFiles {
		// *.From.ReplaceString.ImportPath
		//
		// Evaluate it first because the remaining matching is exclusive and will move to the next
		// iteration on a match.

		match, matcherErr := replaceImportPathMatcher(f)
		if matcherErr != nil {
			errs = append(errs, errors.Wrapf(matcherErr, "failed to collect ReplaceString.ImportPath files under [%s]", cfg.BaseFilePath))
			continue
		}

		if match {
			paths.ReplaceStringFiles.ImportPath.Add(f.AbsPath)
		}

		// *.From.CopyOnlyFilePath

		includeMatch, matcherErr := cfg.Include.CopyOnlyFilesSuffix(f)
		if matcherErr != nil {
			errs = append(errs, errors.Wrapf(matcherErr, "failed to collect CopyOnlyFilePath files under [%s]", cfg.BaseFilePath))
			continue
		}

		if includeMatch {
			paths.CopyOnlyFiles.Add(f.AbsPath)
			if cage_filepath.IsGoFile(f.AbsPath) {
				paths.InspectIgnoreDirs.Add(filepath.Dir(f.AbsPath))
			}
			continue
		}

		// *.From.GoDescendantFilePath

		includeMatch, matcherErr = cfg.Include.GoDescendantFilesSuffix(f)
		if matcherErr != nil {
			errs = append(errs, errors.Wrapf(matcherErr, "failed to collect GoDescendantFilePath files under [%s]", cfg.BaseFilePath))
			continue
		}

		if includeMatch {
			paths.GoDescendantFiles.Add(f.AbsPath)
			if cage_filepath.IsGoFile(f.AbsPath) {
				paths.InspectIgnoreDirs.Add(filepath.Dir(f.AbsPath))
			}
			continue // redundant but retain it to indicate the exclusivity of the match
		}
	}

	// Collect *.From.GoFilePath dirs which contain Go files.
	//
	// Omit conflicts with CopyOnlyFilePath.
	//
	// The reason GoFilePath patterns match against directories is due to the current decision to
	// pass a list of directories to x/tools/go/packages.Load instead of individual "file=X" queries.
	// It's not clear whether "file=X" queries would be feasible for large source trees, given Load's
	// dependency on the `go list` command, but it may be possible. The main trade-off in the current
	// approach is that individual files cannot be excluded, but it's unclear if that's a common use case.

	for d, readdir := range paths.AllFiles {
		includeMatch, matcherErr := cfg.Include.InspectDirsSuffix(d, readdir)
		if matcherErr != nil {
			errs = append(errs, errors.Wrapf(matcherErr, "failed to collect GoFilePath dirs under [%s]", cfg.BaseFilePath))
			continue
		}
		if !includeMatch {
			continue
		}

		// Avoid x/tools/go/packages.Package.[]Errors with "no Go files in ..." errors.
		includeMatch, matcherErr = cage_file_matcher.GoDir(d, readdir)
		if matcherErr != nil {
			errs = append(errs, errors.Wrapf(matcherErr, "failed to collect GoFilePath dirs under [%s]", cfg.BaseFilePath))
			continue
		}
		if !includeMatch {
			continue
		}

		if !paths.InspectIgnoreDirs.Contains(d) {
			paths.InspectDirs.Add(d)
		}
	}

	return paths, errs
}

// findUsedDepPkgs returns absolute paths to all Ops.Dep dirs which contain packages
// that are directly/transitively used by packages in the input Ops.From directories.
//
// The input list should omit directories which contain no Go files and include all
// targets individually because they are not searched recursively.
func (a *Audit) findUsedDepPkgs() (errs []error) {
	dirs := cage_strings.NewSet()
	seen := cage_strings.NewSet()
	localGoDirs := a.LocalInspectDirs.SortedSlice()
	fromLocalFilePath := FromAbs(a.op, a.op.From.LocalFilePath)

	// If an iteration enqueues an import path, record the "current" one so that error messages
	// and other record-keeping can obtain that detail.
	//
	// map[<newly enqueued path>]<most recently dequeued path>
	enqueueCauses := make(map[string]string)

	// Catch dependencies not accounted for in Ops.From/Ops.Dep when the egress origin
	// In that case, we are not aware of where the dependencies will be satisfied.
	// In the non-vendoring case, we continue w/o issue under the assumption
	// that the module-based toolchain will resolve the dependencies as needed.
	detectUnconfiguredDirectImport := func(importerPath, importedDir string) (unconfigured bool) {
		if a.LocalIncludeDirs.Contains(importedDir) || a.AllDepDirs.Contains(importedDir) {
			return false
		}

		a.addUnconfiguredDir(importerPath, importedDir)

		return true
	}

	// Catch dependencies not accounted for in Ops.From/Ops.Dep/vendor when the egress origin
	// contains a vendor directory. In that case, we are not aware of where the dependencies
	// will be satisfied. In the non-vendoring case, we continue w/o issue under the assumption
	// that the module-based toolchain will resolve the dependencies as needed.
	detectUnconfiguredTransitiveImport := func(importerPath, importedDir string) (unconfigured bool) {
		// assume package cache hit
		if strings.Contains(importedDir, "/pkg/mod/") && strings.Contains(importedDir, "@") {
			return false
		}

		if a.LocalIncludeDirs.Contains(importedDir) || a.AllDepDirs.Contains(importedDir) {
			return false
		}

		if a.op.From.Vendor {
			if a.isVendorFilePath(importedDir) {
				return false
			}
			a.addUnconfiguredDir(importerPath, importedDir)
			return true
		}

		a.addUnconfiguredDir(importerPath, importedDir)
		return true
	}

	var queue []string
	for _, d := range localGoDirs {
		enqueue := a.localDirToImportPath(d)
		queue = append(queue, enqueue)
		enqueueCauses[enqueue] = d
	}

	for len(queue) > 0 {
		if len(errs) > 0 {
			break
		}

		var dequeued string
		dequeued, queue = queue[0], queue[1:]

		dequeuedPkgs, dequeuedPkgErr := a.pkgCache.LoadImportPathWithBuild(dequeued, fromLocalFilePath, 0)
		if dequeuedPkgErr != nil {
			errs = append(errs, errors.Wrapf(
				dequeuedPkgErr,
				"failed to load Ops[%s].Dep package [%s], it occurred while walking dependencies of [%s]",
				a.op.Id, dequeued, enqueueCauses[dequeued],
			))
			continue
		}

		for _, dequeuedPkg := range dequeuedPkgs {
			isDequeuedDep := a.AllDepDirs.Contains(dequeuedPkg.Dir)

			if strings.HasSuffix(dequeuedPkg.Name, "_test") {
				if isDequeuedDep {
					dep := a.inspectedDirToDep[dequeuedPkg.Dir]
					if dep == nil {
						errs = append(errs, errors.Errorf(
							"failed to find the Ops.Dep config for dir [%s]",
							dequeuedPkg.Dir,
						))
						continue
					}
					if !dep.From.Tests {
						continue
					}
				} else {
					if !a.op.From.Tests {
						continue
					}
				}
			}

			detectUnconfiguredDirectImport(dequeued, dequeuedPkg.Dir)

			// Add the dir of the used Ops.Dep package.

			if isDequeuedDep {
				dirs.Add(dequeuedPkg.Dir)
			}

			// Enqueue Ops.Dep dependencies of the used package.

			importedPaths := cage_strings.NewSet() // Sort iteration to make error lists more stable.
			for p := range dequeuedPkg.Imports {
				if seen.Contains(p) {
					continue
				}
				seen.Add(p)
				importedPaths.Add(p)
			}

			for _, importedPath := range importedPaths.SortedSlice() {
				importedPkgs, importedPkgErr := a.pkgCache.LoadImportPathWithBuild(importedPath, dequeuedPkg.Dir, 0)
				if importedPkgErr != nil {
					errs = append(errs, errors.Wrapf(
						importedPkgErr,
						"failed to load Ops[%s].Dep package's [%s] import [%s]",
						a.op.Id, dequeuedPkg.PkgPath, importedPath,
					))
					continue
				}

				for _, importedPkg := range importedPkgs {
					if a.LocalIncludeDirs.Contains(importedPkg.Dir) {
						continue
					}

					if strings.HasSuffix(importedPkg.Name, "_test") {
						continue
					}

					if importedPkg.Goroot { // standard library
						continue
					}

					if a.AllDepDirs.Contains(importedPkg.Dir) {
						queue = append(queue, importedPath)
						enqueueCauses[importedPath] = dequeued

						a.AllDepImportsIntoLocal.Add(cage_pkgs.NewImportFromPkg(importedPkg))
						if a.LocalInspectDirs.Contains(dequeuedPkg.Dir) {
							a.DirectDepImportsIntoLocal.Add(cage_pkgs.NewImportFromPkg(importedPkg))
						}
					} else {
						detectUnconfiguredTransitiveImport(dequeued, importedPkg.Dir)
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}

	a.DepInspectDirs.AddSet(dirs)

	return []error{}
}

func (a *Audit) inspectGoFiles() (errs []error) {
	// Inspect the Ops.From and Ops.Dep directories found above, except those from CopyOnlyFilePath/GoDescendantFilePath
	// matches which we assume may contain files which cannot compile for one reason or another (e.g. fixtures that are
	// non-Go, fixtures that contain Go code with intended issues, etc.).

	inspectDirs := cage_strings.NewSet().AddSet(a.LocalInspectDirs, a.DepInspectDirs).SortedSlice()

	// Ideally we would perform multiple inspections and query for tests at a granularity matching
	// the config support. Until then, if tests are enabled anywhere in the operation config,
	// we query for tests universally.
	inspectTests := a.op.From.Tests
	if !inspectTests {
		for _, dep := range a.op.Dep {
			if dep.From.Tests {
				inspectTests = true
				break
			}
		}
	}

	a.inspector = cage_pkgs.NewInspector(
		cage_pkgs.NewConfig(&std_packages.Config{
			Dir:   FromAbs(a.op, a.op.From.LocalFilePath),
			Mode:  cage_pkgs.LoadSyntax,
			Tests: inspectTests,
		}),
		inspectDirs...,
	)
	a.inspector.SetPackageCache(a.pkgCache)

	inspectErrs := a.inspector.Inspect()

	if len(inspectErrs) > 0 {
		for _, inspectErr := range inspectErrs {
			errs = append(errs, errors.WithStack(inspectErr))
		}
	}

	return errs
}

func (a *Audit) validateFiles() (errs []error) {
	for _, t := range a.inspector.UnsupportedTraits {
		// Currently all avoided traits are related to their complications for pruning. Tolerate them
		// when we can in order to support a wider variety of codebases.
		if a.isLocalFile(t.FileOrDir) || a.LocalIncludeDirs.Contains(t.FileOrDir) {
			continue
		}

		switch t.Type {

		// These trait is caught here because of its impact of it did in fact cause an import pruning
		// problem. For example, it scenario where the user hits a compilation error, e.g. " imported and not used",
		// when the underlying issue could have been caught before the copy even starts.
		case cage_pkgs.TraitDuplicateImport:
			errs = append(errs, errors.Errorf(
				"files with duplicate imports are not currently supported, usage found in file/dir [%s], package [%s] (%s)", t.FileOrDir, t.PkgPath, t.Msg))

		}
	}

	// Shadowing must end the operation early because the current implementation makes assumptions such as:
	// if identifier "X" is used in a function, and "X" is the name of a global identifier, the used
	// identifier is that global of the same name.
	if len(a.inspector.GlobalIdShadows) > 0 {
		for _, pkgShadows := range a.inspector.GlobalIdShadows {
			for _, fileShadows := range pkgShadows {
				for filename, funcOrMethodShadows := range fileShadows {
					// Currently all avoided traits are related to their complications for pruning. Tolerate them
					// when we can in order to support a wider variety of codebases.
					if a.isLocalFile(filename) || a.isTestFilename(filename) {
						continue
					}

					for funcOrMethodName, shadowedIds := range funcOrMethodShadows {
						for _, idName := range shadowedIds.SortedSlice() {
							errs = append(errs, errors.Errorf(
								"global identifier or import name [%s] is shadowed in function/method [%s] in file [%s]",
								idName, funcOrMethodName, filename,
							))
						}
					}
				}
			}
		}
	}

	return errs
}

func (a *Audit) groupLocalGoFiles() (errs []error) {
	for dir, dirFiles := range a.inspector.GoFiles {
		if a.LocalInspectDirs.Contains(dir) {
			for _, pkgFiles := range dirFiles {
				for _, f := range pkgFiles.SortedSlice() {
					if a.isTestFilename(f) {
						if a.op.From.Tests {
							a.LocalGoTestFiles.Add(f)
							a.logFileActivity(f, "detected as Ops.From test file")
						}
					} else {
						a.LocalGoFiles.Add(f)
						a.logFileActivity(f, "detected as Ops.From implementation file")
					}
				}
			}
		}
	}
	return errs
}

// collectDirectUsageOfDepGlobals identifies the Ops.Dep globals used directly in LocalGoFiles in order
// to seeds lists such as UsedDepGoFiles and mark exports as used in UsedDepExports. (findUsedDepGlobals
// relies on UsedDepExports to seed the DepGlobalIdUsageDag with its first layer of edges from the root.)
func (a *Audit) collectDirectUsageOfDepGlobals() (errs []error) {
	var currentGlobalId cage_pkgs.GlobalId

	walkFn := func(used cage_pkgs.IdUsedByNode) {
		if a.DirectDepImportsIntoLocal.Get(used.IdentInfo.PkgPath) == nil {
			return
		}

		_, err := a.inspector.GlobalIdNode(
			filepath.Dir(used.IdentInfo.Position.Filename), used.IdentInfo.PkgName, used.Name,
		)
		if err != nil {
			errs = append(errs, errors.Wrapf(err,
				"failed to find inspection data about global [%s], "+
					"the data was queried while scanning dependencies of [%s]",
				used.GlobalId(), currentGlobalId,
			))
			return
		}

		globalId := cage_pkgs.NewGlobalId(
			used.IdentInfo.PkgPath,
			used.IdentInfo.PkgName,
			used.IdentInfo.Position.Filename,
			used.Name,
		)

		if _, ok := a.UsedDepExports[used.IdentInfo.PkgPath]; !ok {
			a.UsedDepExports[used.IdentInfo.PkgPath] = make(map[string]cage_pkgs.GlobalId)
		}
		a.UsedDepExports[used.IdentInfo.PkgPath][used.Name] = globalId

		if !a.UsedDepGoFiles.Contains(used.IdentInfo.Position.Filename) {
			if a.addUsedDepGoFile(used.IdentInfo.Position.Filename) {
				a.UsedDepImportPaths.Add(used.IdentInfo.PkgPath)
				a.logFileActivity(
					used.IdentInfo.Position.Filename,
					fmt.Sprintf("detected direct dependency, [%s] uses [%s]", currentGlobalId, globalId),
				)
			}
		}
	}

	for _, dir := range a.inspector.GlobalIdNodes.SortedDirs() {
		if !a.LocalInspectDirs.Contains(dir) { // only process LocalGoFiles
			continue
		}

		dirNodes := a.inspector.GlobalIdNodes[dir]

		for _, pkgName := range dirNodes.SortedPkgNames() {
			pkgNodes := dirNodes[pkgName]

			for _, idName := range pkgNodes.SortedIds() {
				currentGlobalId = cage_pkgs.NewGlobalId(
					pkgNodes[idName].InspectInfo.PkgPath,
					pkgName,
					pkgNodes[idName].InspectInfo.Filename,
					idName,
				)
				walkErrs := a.inspector.WalkGlobalIdsUsedByGlobal(dir, pkgName, idName, walkFn)
				for _, walkErr := range walkErrs {
					errs = append(errs, errors.Wrapf(
						walkErr,
						"failed to find inspection data about global [%s]",
						currentGlobalId,
					))
				}
			}
		}
	}

	return errs
}

// addUsedDepGoFile adds a path to UsedDepGoFiles only if it was not already added to another list.
//
// It returns true if the path was added.
func (a *Audit) addUsedDepGoFile(absPath string) bool {
	if a.isLocalFile(absPath) {
		return false
	}
	if a.DepGoTestFiles.Contains(absPath) {
		return false
	}
	a.UsedDepGoFiles.Add(absPath)
	return true
}

// addDepGoTestFile adds a path to DepGoTestFiles only if it was not already added to another list.
//
// It returns true if the path was added.
func (a *Audit) addDepGoTestFile(absPath string) bool {
	if a.isLocalFile(absPath) {
		return false
	}
	if a.UsedDepGoFiles.Contains(absPath) {
		return false
	}
	a.DepGoTestFiles.Add(absPath)
	return true
}

// findDepUsage builds the Ops.Dep dependency DAGs and also collects metadata such as filenames
// (used for the copy step) from the recursive walks to build the former.
//
// Earlier transplant builds relied on the DAG for pruning decisions, but that approach was replaced
// by a map (Audit.usedDepGlobalIdStr) lookup. While there may be some value of the DAG for CLI commands
// which assist troubleshooting, the DAG itself is currently unnecesary.
func (a *Audit) findDepUsage() (errs []error) {
	var searchRootNodes []cage_pkgs.GlobalId

	// Collect Ops.Dep nodes directly used by Ops.From packages (except the init functions
	// used via blank imports).

	searchRootNodes, errs = a.directlyUsedDepNodes()
	if len(errs) > 0 {
		for n := range errs {
			errs[n] = errors.WithStack(errs[n])
		}
		return errs
	}

	// Seed the initial findUsedDepGlobals search with:
	//
	// - Ops.Dep init function nodes "used" by Ops.From packages via blank/dot imports.
	// - Ops.Dep global nodes (including init) "used" by Ops.From packages via dot imports.
	//
	// This complements the enqueuing of transitively used nodes, of the above types, by findUsedDepGlobals.

	// seenInitFuncDirs stores directory absolute paths to Ops.Dep packages already scanned for init functions.
	seenInitFuncDirs := cage_strings.NewSet()

	// seenNonInitFuncDirs stores directory absolute paths to Ops.Dep packages already scanned for non-init globals.
	seenNonInitFuncDirs := cage_strings.NewSet()

	findInitNodes := func(fileImports cage_pkgs.FileImportPaths) {
		for _, pathsImported := range fileImports {
			for _, pathImported := range pathsImported.SortedSlice() {
				i := a.DirectDepImportsIntoLocal.Get(pathImported)
				if i == nil { // only process Ops.Dep paths
					continue
				}

				if seenInitFuncDirs.Contains(i.Dir) {
					continue
				}

				// Only collect implementation init functions because findUsedDepPkgs has the responsibility
				// of deciding whether test packages should be included based factors including configuration.
				for _, initNode := range a.implInitFuncNodesInDir(i.Dir) {
					searchRootNodes = append(searchRootNodes, initNode)
				}

				seenInitFuncDirs.Add(i.Dir)
			}
		}
	}

	for _, dir := range a.LocalInspectDirs.SortedSlice() {
		// Collect Ops.Dep init function nodes "used" by Ops.From packages via blank imports.
		if a.inspector.BlankImports[dir] != nil {
			for _, fileImports := range a.inspector.BlankImports[dir] {
				findInitNodes(fileImports)
			}
		}

		// Collect Ops.Dep global nodes (including init) "used" by Ops.From packages via dot imports.
		if a.inspector.DotImports[dir] != nil {
			for _, fileImports := range a.inspector.DotImports[dir] {
				findInitNodes(fileImports)

				for _, pathsImported := range fileImports { // find non-init-function nodes
					for _, pathImported := range pathsImported.SortedSlice() {
						i := a.DirectDepImportsIntoLocal.Get(pathImported)
						if i == nil { // only process Ops.Dep paths
							continue
						}

						if seenNonInitFuncDirs.Contains(i.Dir) {
							continue
						}

						// Only collect implementation nodes because findUsedDepPkgs has the responsibility
						// of deciding whether test packages should be included based factors including configuration.
						for _, initNode := range a.implNonInitFuncNodesInDir(i.Dir) {
							searchRootNodes = append(searchRootNodes, initNode)
						}

						seenNonInitFuncDirs.Add(i.Dir)
					}
				}
			}
		}
	}

	// Ensure used Ops.Dep globals are not pruned through a transitive dependency search.
	// Seed the search with the nodes collected above.

	inputGlobalsType := fmt.Sprintf("Ops[%s].From.FilePath file", a.op.Id)
	if dagErrs := a.findUsedDepGlobals(searchRootNodes, inputGlobalsType); len(dagErrs) > 0 {
		for _, dagErr := range dagErrs {
			errs = append(errs, errors.WithStack(dagErr))
		}
		return errs
	}

	// Ensure iota-valued constants (and their direct/transitive dependencies) are not pruned in order
	// to avoid value shifting side-effects.

	searchRootNodes = a.getDepIotaConstGlobalIds()
	inputGlobalsType = "const single/multi-declaration with at least one iota value"
	if dagErrs := a.findUsedDepGlobals(searchRootNodes, inputGlobalsType); len(dagErrs) > 0 {
		for _, dagErr := range dagErrs {
			errs = append(errs, errors.WithStack(dagErr))
		}
		return errs
	}

	// Ensure dependencies of blank identifiers are not pruned.

	for _, dir := range a.inspector.GlobalIdNodes.SortedDirs() {
		dirNodes := a.inspector.GlobalIdNodes[dir]
		for _, pkgName := range dirNodes.SortedPkgNames() {
			pkgNodes := dirNodes[pkgName]
			for _, idName := range pkgNodes.SortedIds() {
				if !strings.HasPrefix(idName, cage_pkgs.BlankIdNamePrefix) {
					continue
				}
				node := pkgNodes[idName]
				if addErrs := a.addBlankIdToDepGlobalIdDag(dir, pkgName, idName, node); len(addErrs) > 0 {
					for _, addErr := range addErrs {
						errs = append(errs, errors.WithStack(addErr))
					}
				}
			}
		}
	}

	return errs
}

func (a *Audit) addDepGlobalUsageVertex(id cage_pkgs.GlobalId) {
	a.usedDepGlobalIdStr.Add(id.String())
	a.DepGlobalIdUsageDag.Add(id)
}

func (a *Audit) addDepGlobalUsageEdge(from, to cage_pkgs.GlobalId) error {
	if err := a.DepGlobalIdUsageDag.Connect(from, to); err != nil {
		return errors.Wrapf(err, "failed to create DAG edge from [%s] to [%s]", from, to)
	}
	return nil
}

func (a *Audit) findUsedDepGlobals(usedGlobalIds []cage_pkgs.GlobalId, usedGlobalType string) (errs []error) {
	var queue []cage_pkgs.GlobalId

	// If an iteration enqueues an ID, record the "current" ID so that error messages can include
	// a hint about the origin of any ID. Keys and values are both cage_pkgs.GlobalId.String() values.
	idCauses := make(map[string]string)

	// seenInitFuncDirs stores directory absolute paths to avoid reprocessing implementation packages
	// when looking for init functions to enqueue.
	seenInitFuncDirs := cage_strings.NewSet()

	// seenNonInitFuncDirs stores directory absolute paths to avoid reprocessing implementation packages
	// when looking for non-init-function global nodes to enqueue.
	seenNonInitFuncDirs := cage_strings.NewSet()

	// seenTestPkgs stores import paths to avoid reprocessing test packages when collecting
	// their files to include in the copy.
	seenTestPkgs := cage_strings.NewSet()

	// seenCandidateTestDirs stores absolute paths to package directories already checked for test packages.
	seenCandidateTestDirs := cage_strings.NewSet()

	registerUsage := func(from, to cage_pkgs.GlobalId, cause string) {
		if !a.DepGlobalIdUsageDag.HasVertex(to) {
			queue = append(queue, to)
			a.addDepGlobalUsageVertex(to)
			idCauses[to.String()] = cause
		}

		if connectErr := a.addDepGlobalUsageEdge(from, to); connectErr != nil {
			errs = append(errs, errors.WithStack(connectErr))
		}
	}

	// addDirVertex returns a directory-only vertex and, if needed, adds and connects it to the DAG.
	addDirVertex := func(dir string) cage_pkgs.GlobalId {
		dirVertex := cage_pkgs.NewGlobalId("", "", dir, "")

		if !a.DepGlobalIdUsageDag.HasVertex(dirVertex) {
			a.addDepGlobalUsageVertex(dirVertex)
			if connectErr := a.addDepGlobalUsageEdge(a.LocalGoFilesDagRoot, dirVertex); connectErr != nil {
				errs = append(errs, errors.WithStack(connectErr))
			}
		}

		return dirVertex
	}

	// Update the DAG with edges extending from the root to these types of GlobalId vertexes: those with only
	// the Dir field populated in order to represent a test package directory, and those with only the Dir and
	// Pkg fields populated in order to represent a test package.
	//
	// Once a path passes through directory-only/package-only vertexes and reaches a GlobalId vertex that
	// represents a Ops.Dep global, then all descendant vertices also represent globals.
	addTestPkg := func(testPkgDir, testPkgName string) {
		dirNodes := a.inspector.GlobalIdNodes[testPkgDir]
		if dirNodes == nil {
			return
		}

		pkgNodes := a.inspector.GlobalIdNodes[testPkgDir][testPkgName]
		if pkgNodes == nil {
			return
		}

		dirVertex := addDirVertex(testPkgDir)
		testPkgVertex := cage_pkgs.NewGlobalId("", testPkgName, testPkgDir, "")

		if a.DepGlobalIdUsageDag.HasVertex(testPkgVertex) {
			// implies we already queued its global in the loop below, so we can end early
			return
		}

		a.addDepGlobalUsageVertex(testPkgVertex)
		if connectErr := a.addDepGlobalUsageEdge(dirVertex, testPkgVertex); connectErr != nil {
			errs = append(errs, errors.WithStack(connectErr))
			return
		}

		for _, idName := range pkgNodes.SortedIds() {
			idVertex := cage_pkgs.NewGlobalId(pkgNodes[idName].InspectInfo.PkgPath, testPkgName, testPkgDir, idName)
			registerUsage(testPkgVertex, idVertex, fmt.Sprintf("a %s file in the dir %s", testPkgName, testPkgDir))
		}
	}

	addTestFiles := func(importPath, dir string) {
		if seenCandidateTestDirs.Contains(dir) {
			return
		}
		seenCandidateTestDirs.Add(dir)

		// Collect test package names if any exist in the same directory as the current file.

		buildPkgs, buildPkgsErr := a.pkgCache.LoadImportPathWithBuild(importPath, dir, 0)
		if buildPkgsErr != nil {
			errs = append(errs, errors.Wrapf(
				buildPkgsErr,
				"failed to search for test files in package [%s] dir [%s]", importPath, dir,
			))
			return
		}

		for _, buildPkg := range buildPkgs {
			if !strings.HasSuffix(buildPkg.Name, "_test") {
				continue
			}

			if !strings.HasSuffix(buildPkg.Name, "_test") || seenTestPkgs.Contains(buildPkg.PkgPath) {
				continue
			}

			for _, d := range a.inspector.GlobalIdNodes.SortedDirs() {
				if d != dir {
					continue
				}

				dirNodes := a.inspector.GlobalIdNodes[d]

				for _, pkgName := range dirNodes.SortedPkgNames() {
					if pkgName != buildPkg.Name {
						continue
					}

					for _, testFilename := range buildPkg.GoFiles {
						if !a.DepGoTestFiles.Contains(testFilename) {
							a.addDepGoTestFile(testFilename)

							a.logFileActivity(testFilename, "detected as Ops.Dep test file")
						}
					}

					addTestPkg(dir, buildPkg.Name)
					seenTestPkgs.Add(buildPkg.PkgPath)

					return
				}
			}
		}
	}

	// Build Audit.DepGlobalIdUsageDag to represent the direct/transitive dependencies of LocalGoFiles on
	// on Ops.Dep globals.

	for _, idVertex := range usedGlobalIds {
		registerUsage(addDirVertex(idVertex.Dir()), idVertex, usedGlobalType)
	}

	for len(queue) > 0 {
		if len(errs) > 0 {
			break
		}

		var dequeued cage_pkgs.GlobalId
		dequeued, queue = queue[0], queue[1:]
		dequeuedDir := dequeued.Dir()

		if !a.AllDepDirs.Contains(dequeuedDir) {
			continue
		}

		idSource := idCauses[dequeued.String()]

		// Query for information about the dequeued node.

		node, nodeErr := a.inspector.GlobalIdNode(dequeuedDir, dequeued.PkgName, dequeued.Name)
		if nodeErr != nil {
			errs = append(errs, errors.Wrapf(
				nodeErr,
				"failed to load inspection results about global [%s], they were queried while walking dependencies of [%s]",
				dequeued, idSource,
			))
			continue
		}

		// Add the node to the DAG, extending an edge from the vertex which identifies the node's package.

		dirVertex := addDirVertex(dequeuedDir)
		if connectErr := a.addDepGlobalUsageEdge(dirVertex, dequeued); connectErr != nil {
			errs = append(errs, errors.WithStack(connectErr))
		}

		// Add test packages to the DAG (based on configuration and node type).

		dep := a.inspectedDirToDep[node.InspectInfo.Dirname]
		if dep == nil {
			errs = append(errs, errors.Errorf(
				"failed to find the Ops.Dep config for file [%s]",
				node.InspectInfo.Filename,
			))
			continue
		}

		// Do not search for tests if the current node is an init function because the usage of the latter
		// is not a reliable signal that there are any used implementation globals to test in the first place
		// (e.g. the init function was only enqueued because of a blank import of its package).
		// Rather than examine other signals here, such as whether any implementation nodes from the same
		// package have already been processed, we will simply let those other iterations lead to test
		// inclusion naturally (which may have already happened).
		if dep.From.Tests && node.InspectInfo.InitFuncPos == -1 {
			addTestFiles(node.InspectInfo.PkgPath, node.InspectInfo.Dirname)
		}

		// Collect the filename of the dequeued node.

		if a.isTestFilename(node.InspectInfo.Filename) {
			if dep.From.Tests && !a.DepGoTestFiles.Contains(node.InspectInfo.Filename) {
				a.addDepGoTestFile(node.InspectInfo.Filename)
				a.logFileActivity(
					node.InspectInfo.Filename,
					fmt.Sprintf("transitive dependency, cause [%s]", idSource),
				)
			}
		} else {
			if !a.UsedDepGoFiles.Contains(node.InspectInfo.Filename) {
				if a.addUsedDepGoFile(node.InspectInfo.Filename) {
					a.UsedDepImportPaths.Add(node.InspectInfo.PkgPath)
					a.logFileActivity(
						node.InspectInfo.Filename,
						fmt.Sprintf("transitive dependency, cause [%s]", idSource),
					)
				}
			}
		}

		// Query for the global nodes used by the dequeued node.

		usedMap, idsErrs := a.inspector.GlobalIdsUsedByGlobal(dequeuedDir, dequeued.PkgName, dequeued.Name)
		if len(idsErrs) > 0 {
			for n := range idsErrs {
				errs = append(errs, errors.Wrapf(
					idsErrs[n],
					"failed to load inspection results about global [%s], they were queried while walking dependencies of [%s]",
					dequeued, idSource,
				))
			}
			break
		}

		// Enqueue, and Collect info about, all the global nodes used by the dequeued node.

		var ids []cage_pkgs.GlobalId
		for _, used := range usedMap {
			ids = append(ids, used.GlobalId())
		}

		// If dequeued is a struct type, adds its methods unconditionally in order to effectively add the entire struct.
		// This is to entirely avoid the problem of pruning methods which satisfy an interface somewhere in the codebase.
		ids = append(ids, a.getMethodIds(dequeued)...)

		keyToObj := make(map[string]*cage_pkgs.GlobalId) // Sort iteration to make error lists more stable.
		keys := cage_strings.NewSet()
		for n, id := range ids {
			key := id.String()
			keyToObj[key] = &ids[n]
			keys.Add(key)
		}

		for _, key := range keys.SortedSlice() {
			id := keyToObj[key]
			vertex := cage_pkgs.NewGlobalId(id.PkgPath, id.PkgName, id.Filename, id.Name)
			registerUsage(dequeued, vertex, dequeued.String())
		}

		// Enqueue all init functions found in the same directory as the dequeued node.

		if !seenInitFuncDirs.Contains(dequeuedDir) {
			// Only collect implementation init functions because earlier queue iteration logic decides whether test
			// packages should be included based factors including configuration.
			for _, initNode := range a.implInitFuncNodesInDir(dequeuedDir) {
				queue = append(queue, initNode)
				registerUsage(dirVertex, initNode, fmt.Sprintf(
					"directory [%s] contains at least one used global [%s]", dequeuedDir, dequeued.String(),
				))
			}
			seenInitFuncDirs.Add(dequeuedDir)
		}

		// Enqueue all init functions found in the every package imported, using a blank "_" import name,
		// by the dequeued node's file.

		pathsImportedAsBlank := a.inspector.BlankImportsInFile(dequeuedDir, dequeued.PkgName, node.InspectInfo.Filename)
		if pathsImportedAsBlank != nil {
			for _, pathImportedAsBlank := range pathsImportedAsBlank.SortedSlice() {
				i := a.AllDepImportsIntoLocal.Get(pathImportedAsBlank)
				if i == nil { // only process Ops.Dep paths
					continue
				}

				if seenInitFuncDirs.Contains(i.Dir) {
					continue
				}

				// Only collect implementation init functions because earlier queue iteration logic decides whether test
				// packages should be included based factors including configuration.
				for _, initNode := range a.implInitFuncNodesInDir(i.Dir) {
					queue = append(queue, initNode)
					registerUsage(dirVertex, initNode, fmt.Sprintf(
						"file [%s] imported package [%s] with a blank import name", node.InspectInfo.Filename, pathImportedAsBlank,
					))
				}

				seenInitFuncDirs.Add(i.Dir)
			}
		}

		// Enqueue all global nodes (including init) found in the every package imported, using a dot import name,
		// by the dequeued node's file.

		pathsImportedAsDot := a.inspector.DotImportsInFile(dequeuedDir, dequeued.PkgName, node.InspectInfo.Filename)
		if pathsImportedAsDot != nil {
			for _, pathImportedAsDot := range pathsImportedAsDot.SortedSlice() {
				i := a.AllDepImportsIntoLocal.Get(pathImportedAsDot)
				if i == nil { // only process Ops.Dep paths
					continue
				}

				// Enqueue all init functions.
				if seenInitFuncDirs.Contains(i.Dir) {
					continue
				} else {
					// Only collect implementation init functions because earlier queue iteration logic decides whether test
					// packages should be included based factors including configuration.
					for _, initNode := range a.implInitFuncNodesInDir(i.Dir) {
						queue = append(queue, initNode)
						registerUsage(dirVertex, initNode, fmt.Sprintf(
							"file [%s] imported package [%s] with a dot import name", node.InspectInfo.Filename, pathImportedAsDot,
						))
					}

					seenInitFuncDirs.Add(i.Dir)
				}

				// Enqueue all non-init-function global nodes.
				if seenNonInitFuncDirs.Contains(i.Dir) {
					continue
				} else {

					// Only collect implementation init functions because earlier queue iteration logic decides whether test
					// packages should be included based factors including configuration.
					for _, nonInitNode := range a.implNonInitFuncNodesInDir(i.Dir) {
						queue = append(queue, nonInitNode)
						registerUsage(dirVertex, nonInitNode, fmt.Sprintf(
							"file [%s] imported package [%s] with a dot import name", node.InspectInfo.Filename, pathImportedAsDot,
						))
					}

					seenNonInitFuncDirs.Add(i.Dir)
				}
			}
		}
	}

	return errs
}

// directlyUsedDepNodes returns the GlobalId of each Ops.Dep global directly used by Ops.From.
func (a *Audit) directlyUsedDepNodes() (ids []cage_pkgs.GlobalId, errs []error) {
	if collectErrs := a.collectDirectUsageOfDepGlobals(); len(collectErrs) > 0 {
		for _, collectErr := range collectErrs {
			errs = append(errs, errors.WithStack(collectErr))
		}
		return []cage_pkgs.GlobalId{}, errs
	}

	for _, i := range a.DirectDepImportsIntoLocal.SortedSlice() {
		if exportIds, ok := a.UsedDepExports[i.Path]; ok {
			for _, id := range exportIds {
				vertex := cage_pkgs.NewGlobalId(i.Path, i.DeclName, id.Filename, id.Name)
				ids = append(ids, vertex)
			}
		}
	}

	return ids, []error{}
}

// implInitFuncNodesInDir returns the GlobalId of each init function, in implementation files,
// declared in the directory identified by its absolute path.
func (a *Audit) implInitFuncNodesInDir(dir string) (ids []cage_pkgs.GlobalId) {
	dirNodes := a.inspector.GlobalIdNodes[dir]
	if dirNodes == nil {
		return []cage_pkgs.GlobalId{}
	}

	for _, pkgName := range dirNodes.SortedPkgNames() {
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		pkgNodes := dirNodes[pkgName]
		if pkgNodes == nil {
			continue
		}

		for _, idName := range pkgNodes.SortedIds() {
			node := pkgNodes[idName]
			if node.InspectInfo.InitFuncPos == -1 {
				continue
			}

			ids = append(ids, cage_pkgs.NewGlobalId(node.InspectInfo.PkgPath, pkgName, node.InspectInfo.Filename, idName))
		}
	}

	return ids
}

// implNonInitFuncNodesInDir returns the GlobalId of each non-init global, in implementation files,
// declared in the directory identified by its absolute path.
func (a *Audit) implNonInitFuncNodesInDir(dir string) (ids []cage_pkgs.GlobalId) {
	dirNodes := a.inspector.GlobalIdNodes[dir]
	if dirNodes == nil {
		return []cage_pkgs.GlobalId{}
	}

	for _, pkgName := range dirNodes.SortedPkgNames() {
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		pkgNodes := dirNodes[pkgName]
		if pkgNodes == nil {
			continue
		}

		for _, idName := range pkgNodes.SortedIds() {
			node := pkgNodes[idName]
			if node.InspectInfo.InitFuncPos != -1 {
				continue
			}

			ids = append(ids, cage_pkgs.NewGlobalId(node.InspectInfo.PkgPath, pkgName, node.InspectInfo.Filename, idName))
		}
	}

	return ids
}

// addBlankIdToDepGlobalIdDag adds vertices for regular-form blank identifier declarations whose
// type dependencies on both sides of the assignment are already recorded as used directly/transitively
// by LocalGoFiles.
func (a *Audit) addBlankIdToDepGlobalIdDag(dir, pkgName, idName cage_pkgs.GlobalIdName, node cage_pkgs.Node) (errs []error) {
	// Double-check the global ID name is well-formed.

	_, parseErr := cage_pkgs.NewBlankIdFromString(idName)
	if parseErr != nil {
		errs = append(errs, errors.WithStack(parseErr))
		return
	}

	blankId := cage_pkgs.NewGlobalId(node.InspectInfo.PkgPath, pkgName, node.InspectInfo.Filename, idName)

	// Collect the identifiers used in blank identifier declarations, indexed by GlobalId.String().
	// Sort them by their left-hand-side (LHS) or right-hand-side (RHS) position in the assignment.

	usedOnLhs := make(map[string]cage_pkgs.GlobalId)
	usedOnRhs := make(map[string]cage_pkgs.GlobalId)

	usedMap, idsErrs := a.inspector.GlobalIdsUsedByGlobal(blankId.Dir(), blankId.PkgName, blankId.Name)
	if len(idsErrs) > 0 {
		for _, idsErr := range idsErrs {
			errs = append(errs, errors.Wrapf(idsErr, "failed to load inspection results about global [%s]", blankId))
		}
		return
	}

	for _, used := range usedMap {
		// Ensure that all type dependencies were already recorded as used directly/transitively by LocalGoFiles.
		if !a.IsDepGlobalUsedInLocal(used.GlobalId()) {
			return
		}

		if used.BlankIdAssignPos == cage_pkgs.LhsAssignUsage {
			usedOnLhs[used.GlobalId().String()] = used.GlobalId()
		} else if used.BlankIdAssignPos == cage_pkgs.RhsAssignUsage {
			usedOnRhs[used.GlobalId().String()] = used.GlobalId()
		} else {
			errs = append(errs, errors.Errorf(
				"blank ID [%s] has type dependency [%s] with invalid BlankIdAssignPos [%d]",
				blankId, used.GlobalId(), used.BlankIdAssignPos,
			))
		}
	}

	// Add the blank identifier itself to the DAG, connecting it to its directory's vertex just as
	// non-blank identifiers are.

	dirVertex := cage_pkgs.NewGlobalId("", "", dir, "")
	a.addDepGlobalUsageVertex(blankId)
	if connectErr := a.addDepGlobalUsageEdge(dirVertex, blankId); connectErr != nil {
		errs = append(errs, errors.WithStack(connectErr))
	}

	// Connect the LHS/RHS type dependencies to the DAG.

	for _, lhsTypeDepId := range usedOnLhs {
		if connectErr := a.addDepGlobalUsageEdge(blankId, lhsTypeDepId); connectErr != nil {
			errs = append(errs, errors.WithStack(connectErr))
		}
	}
	for _, rhsTypeDepId := range usedOnRhs {
		if connectErr := a.addDepGlobalUsageEdge(blankId, rhsTypeDepId); connectErr != nil {
			errs = append(errs, errors.WithStack(connectErr))
		}
	}

	return errs
}

func (a *Audit) isTestFilename(p string) bool {
	return strings.HasSuffix(p, "_test.go")
}

func (a *Audit) localDirToImportPath(dir string) string {
	fromLocalFilePath := FromAbs(a.op, a.op.From.LocalFilePath)

	if dir == fromLocalFilePath {
		return a.op.From.LocalImportPath
	}

	sansPrefix := strings.TrimPrefix(dir, fromLocalFilePath+string(filepath.Separator))
	if dir != sansPrefix {
		relImportPathParts := strings.Split(sansPrefix, string(filepath.Separator))
		return path.Join(append([]string{a.op.From.LocalImportPath}, relImportPathParts...)...)
	}
	return ""
}

func (a *Audit) depDirToImportPath(dir string) string {
	// Cover the case where a Op.Dep.From.ImportPath is a prefix of Op.From.LocalImportPath,
	// e.g. "/path/to/deps" and "/path/to/deps/cmd/some_project" respectively, where
	// the former contains first-party dependencies of the latter and its easier to
	// provide the "deps" path instead of individual "deps/*" paths to specific package trees.
	// if strings.HasPrefix(dir, FromAbs(a.op, a.op.From.LocalFilePath)) {
	// return ""
	// }

	for _, d := range a.op.Dep {
		fromAbsPath := FromAbs(a.op, d.From.FilePath)

		if dir == fromAbsPath {
			return d.From.ImportPath
		}

		// Config.ReadFile validates that Ops.Dep.From.{File,Import}Path values must both be unique and
		// not overlap, so any prefix match is the target and there should be one at most.
		sansPrefix := strings.TrimPrefix(dir, fromAbsPath+string(filepath.Separator))
		if dir != sansPrefix {
			relImportPathParts := strings.Split(sansPrefix, string(filepath.Separator))
			return path.Join(append([]string{d.From.ImportPath}, relImportPathParts...)...)
		}
	}

	return ""
}

func (a *Audit) isVendorFilePath(p string) bool {
	return a.op.From.Vendor && strings.HasPrefix(p, FromAbs(a.op, "vendor"))
}

func (a *Audit) isLocalFile(f string) bool {
	return a.LocalGoFiles.Contains(f) ||
		a.LocalGoTestFiles.Contains(f) ||
		a.LocalGoDescendantFiles.Contains(f) ||
		a.LocalCopyOnlyFiles.Contains(f)
}

func (a *Audit) IsDepGlobalUsedInLocal(g cage_pkgs.GlobalId) bool {
	return a.usedDepGlobalIdStr.Contains(g.String())
}

// getDepIotaConstGlobalIds returns all Ops.Dep global identifiers which are iota-valued constants.
func (a *Audit) getDepIotaConstGlobalIds() (ids []cage_pkgs.GlobalId) {
	for _, dir := range a.inspector.GlobalIdNodes.SortedDirs() {
		if !a.AllDepDirs.Contains(dir) {
			continue
		}

		dirNodes := a.inspector.GlobalIdNodes[dir]

		for _, pkgName := range dirNodes.SortedPkgNames() {
			pkgNodes := dirNodes[pkgName]
			for _, idName := range pkgNodes.SortedIds() {
				node := pkgNodes[idName]
				if node.InspectInfo.IotaValuedNames.Contains(idName) {
					ids = append(
						ids,
						cage_pkgs.NewGlobalId(
							node.InspectInfo.PkgPath,
							pkgName,
							node.InspectInfo.Filename,
							idName,
						),
					)
				}
			}
		}
	}

	return ids
}

// getMethodIds returns a GlobalId of every method in the subject identifier if the latter
// is a struct type.
//
// If the subject identifier is not a global, the returned list will be empty.
func (a *Audit) getMethodIds(subjectId cage_pkgs.GlobalId) (methodIds []cage_pkgs.GlobalId) {
	dirIdNodes, ok := a.inspector.GlobalIdNodes[subjectId.Dir()]
	if !ok {
		return []cage_pkgs.GlobalId{}
	}

	pkgIdNodes, ok := dirIdNodes[subjectId.PkgName]
	if !ok {
		return []cage_pkgs.GlobalId{}
	}

	_, ok = pkgIdNodes[subjectId.Name]
	if !ok {
		return []cage_pkgs.GlobalId{}
	}

	for _, idName := range pkgIdNodes.SortedIds() {
		if strings.HasPrefix(idName, subjectId.Name+cage_pkgs.GlobalIdSeparator) {
			methodIds = append(methodIds, cage_pkgs.NewGlobalId(
				pkgIdNodes[idName].InspectInfo.PkgPath,
				subjectId.PkgName,
				pkgIdNodes[idName].InspectInfo.Filename,
				idName,
			))
		}
	}

	return methodIds
}

func (a *Audit) addUnconfiguredDir(importerPath, dir string) {
	if a.UnconfiguredDirs[importerPath] == nil {
		a.UnconfiguredDirs[importerPath] = cage_strings.NewSet()
	}
	a.UnconfiguredDirs[importerPath].Add(dir)
	a.UnconfiguredDirImporters.Add(importerPath)
}

func (a *Audit) logFileActivity(name, msg string) {
	if a.WhyLog != nil {
		a.WhyLog[name] = append(a.WhyLog[name], msg)
	}
}

func (a *Audit) PrintUnconfiguredDirs(w io.Writer) {
	if len(a.UnconfiguredDirs) == 0 {
		return
	}

	heading := "Some Ops.From packages, or their Ops.Dep dependencies, import other packages which are missing from the configuration file.\n" +
		"Below are the packages which perform the imports, followed by the file locations of packages for which there is no configuration:\n"
	fmt.Fprintln(w, heading)

	for _, importerPath := range a.UnconfiguredDirImporters.SortedSlice() {
		fmt.Fprintf(w, "%s\n", importerPath)
		for _, dir := range a.UnconfiguredDirs[importerPath].SortedSlice() {
			fmt.Fprintf(w, "\t%s\n\n", dir)
		}
	}
}
