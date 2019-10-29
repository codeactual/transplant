// Copyright (C) 2019 The transplant Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package transplant_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	testkit_require "github.com/codeactual/transplant/internal/cage/testkit/testify/require"
)

// Per-case comments may refer to configuration file sections such as Ops.From and Ops.Dep.
// The file is located in ./fixture/egress/transplant.yml.

type EgressCopySuite struct {
	Suite
}

func TestEgressCopySuite(t *testing.T) {
	suite.Run(t, new(EgressCopySuite))
}

// TestCopyPlanPruned simulates an copy operation in which there are files which are expected to be
// added, overwritten, pruned, and removed. It asserts that the CopyPlan returned by Copy reflects
// those expected operations based on the fixture files. The "golden" directory represents the expected
// result of the copy operation based on the preexisting destination directory "to".
func (s *EgressCopySuite) TestCopyPlanPruned() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "copy_plan_pruned")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, "copy_plan1.go"),
			filepath.Join(fixture.OutputPath, "go.mod"),
			filepath.Join(fixture.OutputPath, "internal", "dep3", "dep3.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep_four", "dep4a", "dep4a.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep_four", "dep_four.go"),
		},
		fixture.Plan.Add,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, "copy_plan2.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep2", "dep2.go"),
			filepath.Join(fixture.OutputPath, "local1", "local1.go"),
		},
		fixture.Plan.Overwrite,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep2", "dep2.go") + ".dep2.ExportedFunc2",
			filepath.Join(fixture.Path, "origin", "dep4", "dep4.go") + ".dep4.ExportedFunc2",
		},
		fixture.Plan.PruneGlobalIds,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, "internal", "dep1"),
			filepath.Join(fixture.OutputPath, "internal", "dep1", "dep1.go"),
		},
		fixture.Plan.Remove,
	)

	s.DirsMatchExceptGomod(filepath.Join(fixture.GoldenPath+"_stage"), filepath.Join(fixture.Plan.StagePath))
	s.DirsMatchExceptGomod(filepath.Join(fixture.GoldenPath+"_output"), filepath.Join(fixture.OutputPath))
}

// TestCopyPlanDepCompletelyPruned asserts that the Ops.Dep package "dep_completely_pruned" is not included
// in CopyPlan.{Add,Overwrite} because the entire package was pruned due to lack of direct/transitive use
// by Ops.From files.
func (s *EgressCopySuite) TestCopyPlanDepCompletelyPruned() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "copy_plan_dep_completely_pruned")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	// Does not include:
	//   - internal/dep1/file_completely_pruned.go (file was completely pruned)
	//   - internal/dep_completely_pruned (only file was completely pruned)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, "go.mod"),
			filepath.Join(fixture.OutputPath, "internal", "dep1", "dep1.go"),
			filepath.Join(fixture.OutputPath, "proj.go"),
		},
		fixture.Plan.Add,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Overwrite,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep1", "file_completely_pruned.go"),
			filepath.Join(fixture.Path, "origin", "dep_completely_pruned", "dep_completely_pruned.go"),
		},
		fixture.Plan.PruneGoFiles,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Remove,
	)

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestRemoveBaseline asserts that files which are absent in the origin are removed in the destination.
func (s *EgressCopySuite) TestRemoveBaseline() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "remove_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestCommentsBaseline asserts that leading/inline/trailing comments retain the positions in the copy.
func (s *EgressCopySuite) TestCommentsBaseline() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "comments_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestCommentsBaseline asserts that leading/inline/trailing comments retain the positions in the copy,
// even when globals are pruned.
func (s *EgressCopySuite) TestCommentsPruned() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "comments_pruned")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestGenDeclPruned asserts that leading/inline/trailing comments retain the positions in the copy,
// even when globals are pruned from var/const/type/import declaration groups (ast.GenDecl).
func (s *EgressCopySuite) TestGenDeclPruned() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "gendecl_pruned")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestNonGoFiles asserts that non-Go files are included in the copy as long as they are
// identified in CopyOnlyFilePath config sections. It also asserts that CopyOnlyFilePath
// patterns can select non-Go files from directories of pruned packages like "unused1".
func (s *EgressCopySuite) TestNonGoFiles() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "copy_non_go")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, ".local_hidden_file"),
			filepath.Join(fixture.OutputPath, "go.mod"),
			filepath.Join(fixture.OutputPath, "internal", "dep1", ".dep1_hidden_file"),
			filepath.Join(fixture.OutputPath, "internal", "dep1", "dep1.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep1", "dep1.md"),
			filepath.Join(fixture.OutputPath, "internal", "unused1", ".unused1_hidden_file"),
			filepath.Join(fixture.OutputPath, "internal", "unused1", "unused1.md"),
			filepath.Join(fixture.OutputPath, "local.md"),
			filepath.Join(fixture.OutputPath, "proj.go"),
		},
		fixture.Plan.Add,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Overwrite,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "unused1", "unused1.go"),
		},
		fixture.Plan.PruneGoFiles,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Remove,
	)
}

// TestLocalInitAffectsPruning asserts that Ops.Dep globals used directly/transitively by init functions
// in Ops.From files will not be pruned.
//
// See the "Pruning" section in README.md for more details about the expectation/rationale.
func (s *EgressCopySuite) TestLocalInitAffectsPruning() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "local_init_affects_pruning")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(filepath.Join(fixture.GoldenPath, "internal"), filepath.Join(fixture.Plan.StagePath, "internal"))
	s.DirsMatchExceptGomod(filepath.Join(fixture.GoldenPath, "internal"), filepath.Join(fixture.OutputPath, "internal"))
}

// TestInitBaseline asserts that the copy will include init functions as long as their packages are
// directly/transitively used by Ops.From files.
func (s *EgressCopySuite) TestInitBaseline() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "init_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, "go.mod"),
			filepath.Join(fixture.OutputPath, "internal", "dep1", "dep1.go"),
			filepath.Join(fixture.OutputPath, "internal", "only_used_by_dep1_init", "only_used_by_dep1_init.go"),
			filepath.Join(fixture.OutputPath, "proj.go"),
		},
		fixture.Plan.Add,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Overwrite,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "only_used_by_unused1", "only_used_by_unused1.go"),
			filepath.Join(fixture.Path, "origin", "unused1", "unused1.go"),
		},
		fixture.Plan.PruneGoFiles,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Remove,
	)

	s.DirsMatchExceptGomod(filepath.Join(fixture.GoldenPath, "internal"), filepath.Join(fixture.Plan.StagePath, "internal"))
	s.DirsMatchExceptGomod(filepath.Join(fixture.GoldenPath, "internal"), filepath.Join(fixture.OutputPath, "internal"))
}

// TestInitChain asserts that init functions, in both Ops.From and Ops.Dep impl/test packages,
// are included in copies and their transitive dependencies are satisfied.
//
// The "init_chain0" begins in local/local.go and "init_chain1" in local/local_test.go.
// Each of these init functions use a dependency which defines its own init function, and so on.
// At the ends of the chains are init_chain*_init_dep packages which include tests in order to assert
// that the tests of transitive dependencies are also included (as long as tests are enabled in the config.
func (s *EgressCopySuite) TestInitChain() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "init_chain")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestInitMultiFunc asserts that multiple init functions in a Ops.Dep package (dep1) will
// all impact pruning, which in this case means retaining dep2/dep3.
func (s *EgressCopySuite) TestInitMultiFunc() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "init_multi_func")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestOneMethodUsedPreventsStructPruning asserts that if any method of type T is used:
// all methods of type T are retained and used to inform further pruning.
func (s *EgressCopySuite) TestOneMethodUsedPreventsStructPruning() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "one_method_used_struct_pruning")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestTypeUsedAloneMethodPruning asserts that if a type is used alone,
// e.g. a function that returns an initialized T but does not call any method of T,
// all methods of type T are retained and used to inform further pruning.
func (s *EgressCopySuite) TestTypeUsedAloneMethodPruning() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "type_used_alone_method_pruning")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestImportPruningBaseline asserts that if the pruning of globals leads to imports which are
// no longer required, then those imports are also pruned.
func (s *EgressCopySuite) TestImportPruningBaseline() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "import_pruning_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(filepath.Join(fixture.GoldenPath, "internal"), filepath.Join(fixture.Plan.StagePath, "internal"))
	s.DirsMatchExceptGomod(filepath.Join(fixture.GoldenPath, "internal"), filepath.Join(fixture.OutputPath, "internal"))
}

// TestBlankIdentifierAffectsPruning asserts that blank identifier declarations are pruned
// unless globals on both sides of the assignment are otherwise directly/transitively
// used by Ops.From files. For example, if a blank identifier is used to assert that
// an implementation satisfies an interface, but the implementation or interface is not
// actually used, then the blank identifier is considered safe to omit from the copy.
//
// See the "Pruning" section in README.md for more details about the expectation/rationale.
func (s *EgressCopySuite) TestBlankIdentifierAffectsPruning() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "blank_identifier_affects_pruning")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestStringsRewrittenBaseline asserts that Ops.From.ImportPath config values
// are rewritten to their respective Ops.To.ImportPath values in string literals.
func (s *EgressCopySuite) TestStringsRewrittenBaseline() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "strings_rewritten_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestTestConstIotaPruning asserts that if any const in an iota group is used, then the group
// as a whole is retained even if some members are not used.
//
// See the "Pruning" section in README.md for more details about the expectation/rationale.
func (s *EgressCopySuite) TestTestConstIotaPruning() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "const_iota_pruning")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestRenameBaseline asserts that RenameFilePath config sections can determine filenames of copied files
// in the destination.
func (s *EgressCopySuite) TestRenameBaseline() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "rename_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, "Makefile"),
			filepath.Join(fixture.OutputPath, "another.go"),
			filepath.Join(fixture.OutputPath, "go.mod"),
			filepath.Join(fixture.OutputPath, "proj.go"),
		},
		fixture.Plan.Add,
	)

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestGoDescendantFilePathBaseline asserts that from/local3 and dep2 are omitted because they
// do not have ancestor directories which contain Go files which where included in the copy
// via other config sections, e.g. GoFilePath.
func (s *EgressCopySuite) TestGoDescendantFilePathBaseline() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "godescendantfilepath_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestInitFromOnlyUsedPkgs asserts that init function sources must be packages which are
// directly/transitively based on used globals, not simply imported packages.
//
// dep3's init function is omitted because no dep3 global is used by the from package.
// In at dep2/dep2.go, only ExportedFunc1 is used and dep3 is imported for ExportedFunc2.
// But since ExportedFunc2 is not used, dep3's init function should be pruned.
func (s *EgressCopySuite) TestInitFromOnlyUsedPkgs() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "init_local_only_used_pkgs")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestCopyFilePerm asserts that the copy operation retains normal/regular file permission modes.
func (s *EgressCopySuite) TestCopyFilePerm() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "copy_file_perm")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)

	stat, err := os.Stat(filepath.Join(fixture.OutputPath, "bin", "tool"))
	require.NoError(t, err)
	require.Exactly(t, os.FileMode(0755), stat.Mode())
}

// TestInitViaDepTests asserts that if an a Ops.From transitive dependency under Ops.Dep.From
// contains a test, then an init function from that test package is retained and its own
// dependencies are also retained.
func (s *EgressCopySuite) TestInitViaDepTests() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "init_via_dep_tests")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestRenameFileAtFilePathRoot asserts that files which share base filenames with top-level
// Ops.From.FilePath/Ops.Dep.From.FilePath directories will be renamed to match associated "To" names.
func (s *EgressCopySuite) TestRenameFileAtFilePathRoot() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "rename_file_at_filepath_root")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestTestSupportBaseline asserts that globals in test files are not pruned, that their
// transitive dependencies are satisfied, and those dependencies in turn affect pruning decisions.
//
// See the "Pruning" section in README.md for more details about the expectation/rationale.
func (s *EgressCopySuite) TestTestSupportBaseline() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "test_support_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, "go.mod"),
			filepath.Join(fixture.OutputPath, "internal", "dep2", "dep2.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep2", "dep2_test.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep3", "dep3.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep3", "dep3_test.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep5", "dep5.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep5", "dep5_test.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep7", "dep7.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep7", "dep7_test.go"),
			filepath.Join(fixture.OutputPath, "proj.go"),
			filepath.Join(fixture.OutputPath, "proj_test.go"),
		},
		fixture.Plan.Add,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.OutputPath, "internal", "dep1", "dep1.go"),
			filepath.Join(fixture.OutputPath, "internal", "dep1", "dep1_test.go"),
		},
		fixture.Plan.Overwrite,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep4_unused", "dep4_unused.go"),
			filepath.Join(fixture.Path, "origin", "dep4_unused", "dep4_unused_test.go"),
			filepath.Join(fixture.Path, "origin", "dep6_only_used_by_dep4_test_file", "dep6_only_used_by_dep4_test_file.go"),
			filepath.Join(fixture.Path, "origin", "dep6_only_used_by_dep4_test_file", "dep6_only_used_by_dep4_test_file_test.go"),
		},
		fixture.Plan.PruneGoFiles,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Remove,
	)

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestFilePathConfigBaseline asserts that {Go,CopyOnly}FilePath config sections are applied.
//
// See the "Configuration" section in README.md for more details about their expected behaviors.
func (s *EgressCopySuite) TestFilePathConfigBaseline() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "filepath_config_baseline")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	// Go files/dirs (discovered)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "local", "local.go"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "force_test1.go"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "force_test1a", "force_test1a.go"),
		},
		fixture.Audit.LocalGoFiles.Slice(),
	)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep1", "dep1.go"),
		},
		fixture.Audit.UsedDepGoFiles.Slice(),
	)

	// test Go files (discovered)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "local", "local_test.go"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "force_test1_test.go"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "force_test1a", "force_test1a_test.go"),
		},
		fixture.Audit.LocalGoTestFiles.Slice(),
	)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep1", "dep1_test.go"),
		},
		fixture.Audit.DepGoTestFiles.Slice(),
	)

	// Go/non-Go files configured w/ CopyOnlyFilePath

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "local", "fixture1", "copy_only.go"),
			filepath.Join(fixture.Path, "origin", "local", "fixture1", "copy_only.txt"),
			filepath.Join(fixture.Path, "origin", "local", "fixture1", "fixture1a", "copy_only.go"),
			filepath.Join(fixture.Path, "origin", "local", "fixture1", "fixture1a", "copy_only.txt"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "fixture1", "copy_only.go"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "fixture1", "copy_only.txt"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "force_test1a", "fixture", "copy_only.go"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "force_test1a", "fixture", "copy_only.txt"),
		},
		fixture.Audit.LocalCopyOnlyFiles.Slice(),
	)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep1", "fixture1", "copy_only.go"),
			filepath.Join(fixture.Path, "origin", "dep1", "fixture1", "copy_only.txt"),
			filepath.Join(fixture.Path, "origin", "dep1", "fixture1", "fixture1a", "copy_only.go"),
			filepath.Join(fixture.Path, "origin", "dep1", "fixture1", "fixture1a", "copy_only.txt"),
		},
		fixture.Audit.DepCopyOnlyFiles.Slice(),
	)

	// dirs processed by golang.org/x/tools/go/packages in Generate

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "local"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1"),
			filepath.Join(fixture.Path, "origin", "local", "force_test1", "force_test1a"),
		},
		fixture.Audit.LocalInspectDirs.Slice(),
	)

	testkit_require.StringSortedSliceExactly(
		t,
		[]string{
			filepath.Join(fixture.Path, "origin", "dep1"),
		},
		fixture.Audit.DepInspectDirs.Slice(),
	)

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestOverwriteMinimum asserts that the stage/output dirs and CopyPlan only receive regular file
// overwrites where the destination contains different content.
//
// Fixtures contain the string "(edit)" to represent a content change.
func (s *EgressCopySuite) TestOverwriteMinimum() {
	t := s.T()

	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "copy_overwrite_minimal")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(t, cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	changedPaths := []string{
		filepath.Join(fixture.OutputPath, "auto_detect", "changed.go"),
		filepath.Join(fixture.OutputPath, "copy_only", "changed.go"),
		filepath.Join(fixture.OutputPath, "go_descendant", "descendant", "changed.md"),
		filepath.Join(fixture.OutputPath, "internal", "dep1", "auto_detect", "changed.go"),
		filepath.Join(fixture.OutputPath, "internal", "dep1", "copy_only", "changed.go"),
		filepath.Join(fixture.OutputPath, "internal", "dep1", "go_descendant", "descendant", "changed.md"),
	}

	skipPaths := []string{
		filepath.Join(fixture.OutputPath, "auto_detect", "skip.go"),
		filepath.Join(fixture.OutputPath, "copy_only", "skip.go"),
		filepath.Join(fixture.OutputPath, "go_descendant", "descendant", "skip.md"),
		filepath.Join(fixture.OutputPath, "go_descendant", "go_descendant.go"),
		filepath.Join(fixture.OutputPath, "internal", "dep1", "auto_detect", "skip.go"),
		filepath.Join(fixture.OutputPath, "internal", "dep1", "copy_only", "skip.go"),
		filepath.Join(fixture.OutputPath, "internal", "dep1", "go_descendant", "descendant", "skip.md"),
		filepath.Join(fixture.OutputPath, "internal", "dep1", "go_descendant", "go_descendant.go"),
	}

	testkit_require.StringSliceExactly(
		t,
		[]string{filepath.Join(fixture.OutputPath, "go.mod")},
		fixture.Plan.Add,
	)
	testkit_require.StringSliceExactly(
		t,
		skipPaths,
		fixture.Plan.OverwriteSkip,
	)
	testkit_require.StringSliceExactly(
		t,
		changedPaths,
		fixture.Plan.Overwrite,
	)
	testkit_require.StringSliceExactly(
		t,
		[]string{},
		fixture.Plan.Remove,
	)

	for _, p := range changedPaths {
		fi, err := os.Stat(p)
		require.NoError(t, err)
		require.Greater(t, fi.ModTime().UnixNano(), fixture.OutputStats[p].ModTime().UnixNano()) // above plan to overwrite was enacted
	}
	for _, p := range skipPaths {
		fi, err := os.Stat(p)
		require.NoError(t, err)
		require.Exactly(t, fixture.OutputStats[p].ModTime().UnixNano(), fi.ModTime().UnixNano()) // above plan to skip was enacted
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath+"_output", fixture.OutputPath)
}

// TestAllowedShadowsPruningUnaffected asserts that "allowed" global shadowing cases
// do not count as usage of the global, and therefore allow the global to be pruned
// in these cases where the shadowing is the only uses of the common name.
//
// Allowed cases include the names of: method/function parameters, method receivers.
func (s *EgressCopySuite) TestAllowedShadowsPruningUnaffected() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "allowed_shadows_pruning_unaffected")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestBlankImportSupport asserts that "_"-named imports in Ops.Dep packages are not pruned
// and that the direct/transitive dependencies of "_"-imported Ops.Dep packages are included
// in the copy.
//
// Similar to TestInitChain, "blank_chain0" begins in local/local.go and "blank_chain1" in local/local_test.go.
// Each of these imports leads to a package which itself performs a blank import, and so on.
//
// All packages contain tests in order to assert that tests in packages which are blank-imported are
// not included when there is no usage of any other implementation globals (i.e. the tests would
// likely fail because only init functions were copied).
//
// At the ends of the chains are blank_chain*_init_dep packages which include tests that are expected
// in the copy because they're imported from the final init functions in the chain, rather than
// blank-imported.
func (s *EgressCopySuite) TestBlankImportSupport() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "blank_import_support")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}

// TestDotImportSupport asserts that "."-named imports in Ops.Dep packages are not pruned
// and that the direct/transitive dependencies of "."-imported Ops.Dep packages are included
// in the copy.
//
// Similar to TestInitChain, "dot_chain0" begins in local/local.go and "dot_chain1" in local/local_test.go.
// Each of these imports leads to a package which itself performs a dot import, and so on.
//
// All packages contain tests in order to assert that tests in packages which are dot-imported are
// included (because at least one exported global is used).
//
// At the ends of the chains are dot_chain*_init_dep packages which include tests that are expected
// in the copy because they're imported from the final init functions in the chain, rather than
// dot-imported.
func (s *EgressCopySuite) TestDotImportSupport() {
	fixture := s.MustCopyFixtureWithGomod("egress", "egress", "EgressCopySuite", "yml", "dot_import_support")
	if fixture.Plan.StagePath != "" {
		defer func() {
			require.NoError(s.T(), cage_file.RemoveAllSafer(fixture.Plan.StagePath))
		}()
	}

	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.Plan.StagePath)
	s.DirsMatchExceptGomod(fixture.GoldenPath, fixture.OutputPath)
}
