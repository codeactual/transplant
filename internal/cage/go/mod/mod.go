// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package mod

import (
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	cage_io "github.com/codeactual/transplant/internal/cage/io"
)

var (
	// modGOLine matches the `go <version>` directive.
	modGoLine *regexp.Regexp

	// modPathLine matches the `module <path>` directive.
	modPathLine *regexp.Regexp

	// modReplaceLine matches a go.mod `replace` directive.
	// <old path> => <replacement spec>
	modReplaceLine *regexp.Regexp

	// modReplaceNew matches the replacement spec of a go.mod `replace` directive.
	// <new path> [version]
	modReplaceNew *regexp.Regexp

	// modRequireSection matches the contents before, in, and after a go.mod `require (...)` directive block.
	modRequireSection *regexp.Regexp

	// modRequireLine matches a line of a go.mod `require (...)` directive block.
	// <path> <version> [comment]
	modRequireLine *regexp.Regexp

	// sumLine matches lines with the hash of a module or module's go.mod.
	//
	// <path> <version> <hash>
	sumLine *regexp.Regexp
)

func init() {
	modGoLine = regexp.MustCompile(`^go\s+(.+)`)
	modPathLine = regexp.MustCompile(`^module\s+(.+)`)
	modReplaceLine = regexp.MustCompile(`^replace\s+(\S+)\s+=>\s+(.+)`)
	modReplaceNew = regexp.MustCompile(`(\S+)(\s+(\S+))?`)
	modRequireLine = regexp.MustCompile(`^\t(\S+) (\S+)( \/\/.*)?`)
	modRequireSection = regexp.MustCompile(`(?ms)(.*require \()([^)]+)(.*)`)
	sumLine = regexp.MustCompile(`^(\S+) (\S+) (\S+)`)
}

// ModReplace defines one `replace` directive.
type ModReplace struct {
	// Old is the replaced import path.
	Old string

	// New is the replacement path.
	New string

	// Version holds the optional version selection.
	Version string
}

func (r ModReplace) String() string {
	s := "replace " + r.Old + " => " + r.New
	if r.Version != "" {
		s += " " + r.Version
	}
	return s
}

// ModRequire defines one line of the `require (...)` directive block.
type ModRequire struct {
	// Comment is the substring found after the version. If a comment was present, it contains
	// the leading space.
	Comment string

	// Path is an import path.
	Path string

	// Version holds the version substring.
	Version string
}

func (r ModRequire) String() string {
	return r.Path + " " + r.Version + r.Comment
}

type Mod struct {
	// Path is from the `module <path>` directive.
	Path string

	// Go is the `go` directive value.
	Go string

	beforeRequire string
	afterRequire  string

	// replaces indexes each ModReplace by import path.
	replaces map[string]ModReplace

	// replacesOrder holds the import paths in the order they were read.
	replacesOrder []string

	// requires indexes each ModRequire by import path.
	requires map[string]ModRequire

	// requiresOrder holds the import paths in the order they were read.
	requiresOrder []string
}

func NewModFromFile(name string) (m *Mod, err error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer cage_io.CloseOrStderr(f, name)

	return NewMod(f)
}

func NewMod(r io.Reader) (m *Mod, err error) {
	m = &Mod{}

	readBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	readStr := string(readBytes)

	pathMatches := modPathLine.FindAllStringSubmatch(readStr, 1)
	if pathMatches == nil {
		return nil, errors.New("failed to parse go.mod module path")
	}
	m.Path = pathMatches[0][1]

	sectionMatches := modRequireSection.FindAllStringSubmatch(readStr, 1)
	if sectionMatches == nil {
		m.beforeRequire = readStr
		return m, nil
	}

	m.beforeRequire = sectionMatches[0][1]
	m.afterRequire = sectionMatches[0][3]

	m.requires = make(map[string]ModRequire)
	for _, line := range strings.Split(sectionMatches[0][2], "\n") {
		if line == "" {
			continue
		}

		lineMatches := modRequireLine.FindAllStringSubmatch(line, 1)
		if lineMatches == nil {
			return nil, errors.Errorf("failed to parse go.mod line [%s]", line)
		}

		p := lineMatches[0][1]

		m.requires[p] = ModRequire{
			Path:    p,
			Version: lineMatches[0][2],
			Comment: lineMatches[0][3],
		}
		m.requiresOrder = append(m.requiresOrder, p)
	}

	m.replaces = make(map[string]ModReplace)
	collectReplace := func(raw string) error {
		for _, line := range strings.Split(raw, "\n") {
			if line == "" {
				continue
			}

			lineMatches := modReplaceLine.FindAllStringSubmatch(line, 1)
			if lineMatches == nil {
				continue
			}

			oldPath := lineMatches[0][1]

			lineMatches = modReplaceNew.FindAllStringSubmatch(lineMatches[0][2], 1)
			if lineMatches == nil {
				return errors.Errorf("failed to parse go.mod line [%s]", line)
			}

			m.replaces[oldPath] = ModReplace{
				Old:     oldPath,
				New:     lineMatches[0][1],
				Version: lineMatches[0][3],
			}
			m.replacesOrder = append(m.replacesOrder, oldPath)
		}

		return nil
	}
	if err := collectReplace(m.beforeRequire); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := collectReplace(m.afterRequire); err != nil {
		return nil, errors.WithStack(err)
	}

	collectGo := func(raw string) {
		for _, line := range strings.Split(raw, "\n") {
			if line == "" {
				continue
			}

			lineMatches := modGoLine.FindAllStringSubmatch(line, 1)
			if lineMatches == nil {
				continue
			}

			m.Go = lineMatches[0][1]
		}
	}
	collectGo(m.beforeRequire)
	collectGo(m.afterRequire)

	return m, nil
}

func (m *Mod) String() string {
	var b strings.Builder
	b.WriteString("module ")
	b.WriteString(m.Path)
	b.WriteString("\n")
	if m.Go != "" {
		b.WriteString("\n")
		b.WriteString("go ")
		b.WriteString(m.Go)
		b.WriteString("\n")
	}
	if len(m.requiresOrder) > 0 {
		b.WriteString("\nrequire (")
		for _, p := range m.requiresOrder {
			b.WriteString("\n\t")
			b.WriteString(m.requires[p].String())
		}
		b.WriteString("\n)")
	}
	replacesLen := len(m.replacesOrder)
	if replacesLen > 0 {
		b.WriteString("\n")
		for n, p := range m.replacesOrder {
			b.WriteString("\n")
			b.WriteString(m.replaces[p].String())
			if n < replacesLen-1 {
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")
	return b.String()
}

func (m *Mod) DelRequire(importPath string) (found bool) {
	_, found = m.requires[importPath]
	if found {
		for n, p := range m.requiresOrder {
			if p == importPath {
				m.requiresOrder = append(m.requiresOrder[:n], m.requiresOrder[n+1:]...)
				delete(m.requires, importPath)
				return true
			}
		}
	}
	return false
}

func (m *Mod) GetRequire(importPath string) (require ModRequire, found bool) {
	require, found = m.requires[importPath]
	return require, found
}

// SetRequire overwrites the identified require's fields with new values.
func (m *Mod) SetRequire(importPath string, require ModRequire) {
	require.Path = importPath

	// Retain the original comment because the destination go.mod/go/sum are more
	// specific/accurate to its requirements than the source's.
	require.Comment = m.requires[importPath].Comment

	m.requires[importPath] = require
}

// Require returns all `require (...)` directives in the order they were read.
func (m *Mod) Requires() (r []ModRequire) {
	for _, p := range m.requiresOrder {
		r = append(r, m.requires[p])
	}
	return r
}

// Replace returns all `replace` directives in the order they were read.
func (m *Mod) Replaces() (r []ModReplace) {
	for _, p := range m.replacesOrder {
		r = append(r, m.replaces[p])
	}
	return r
}

type SumLine struct {
	Path    string
	Version string
	Hash    string
}

type Sum struct {
	// deps indexes lines by "<path>+<version>".
	hashes map[string]SumLine

	// order holds lines' "<path>+<version>" values in in the order they were read.
	order []string
}

func NewSum(r io.Reader) (s *Sum, err error) {
	s = &Sum{}

	readBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	s.hashes = make(map[string]SumLine)

	for _, line := range strings.Split(strings.TrimSpace(string(readBytes)), "\n") {
		if line == "" {
			continue
		}

		lineMatches := sumLine.FindAllStringSubmatch(line, 1)

		idx := lineMatches[0][1] + lineMatches[0][2]
		s.hashes[idx] = SumLine{
			Path:    lineMatches[0][1],
			Version: lineMatches[0][2],
			Hash:    lineMatches[0][3],
		}

		s.order = append(s.order, idx)
	}

	return s, nil
}

func (s *Sum) GetHash(importPath, version string) string {
	if d, ok := s.hashes[importPath+version]; ok {
		return d.Hash
	}
	return ""
}

// Lines returns SumLine values in the order they were read during NewSum.
func (s *Sum) GetLines() (lines []SumLine) {
	for _, idx := range s.order {
		lines = append(lines, s.hashes[idx])
	}
	return lines
}

// SyncRequire overwrites each go.mod `require (...)` version in the destination with the source's.
func SyncRequire(src, dst *Mod) (errs []error) {
	for _, r := range dst.Requires() {
		srcReq, found := src.GetRequire(r.Path)

		// This is expected if the destination is a main module and the source is not,
		// or if `go mod tidy` has been used in the destination but not the source.
		if !found {
			continue
		}

		dst.SetRequire(r.Path, srcReq)
	}

	if len(errs) > 0 {
		return errs
	}

	return errs
}

// CompareSum verifies each go.sum hash in the destination matches the source's.
func CompareSum(src, dst *Sum) (errs []error) {
	for _, line := range dst.GetLines() {
		srcHash := src.GetHash(line.Path, line.Version)

		// This is expected if the destination is a main module and the source is not,
		// or if `go mod tidy` has been used in the destination but not the source.
		if srcHash == "" {
			continue
		}

		if line.Hash != srcHash {
			errs = append(errs, errors.Errorf("expected hash for [%s] version [%s] to be [%s], found [%s]", line.Path, line.Version, srcHash, line.Hash))
		}
	}

	return errs
}
