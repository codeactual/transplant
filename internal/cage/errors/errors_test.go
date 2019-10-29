// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package errors_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	cage_errors "github.com/codeactual/transplant/internal/cage/errors"
	cage_trace "github.com/codeactual/transplant/internal/cage/trace"
)

var thisFile string

func init() {
	thisFile = cage_trace.ThisFile()
}

func TestSingleErrEvent(t *testing.T) {
	testSingleErrEvent(t) // to add a non-runtime/testing stack frame
}

func testSingleErrEvent(t *testing.T) {
	digitRe := regexp.MustCompile(`^\d+$`)
	f2 := func() error {
		return errors.New("f2 root")
	}
	f1 := func() error {
		return errors.Wrap(f2(), "f1 wrap")
	}
	err := errors.Wrap(f1(), "final wrap")

	event, eventErr := cage_errors.NewEvent(err)
	require.NoError(t, eventErr)

	require.Exactly(
		t,
		cage_errors.Event{
			Loc: cage_errors.Location{
				File: thisFile,
				Func: "testSingleErrEvent",
				Line: "41",
				Pkg:  "errors_test",
			},
			Errors: []cage_errors.Error{
				{
					Msg: "final wrap",
					Loc: cage_errors.Location{
						File: thisFile,
						Func: "testSingleErrEvent",
						Line: "39",
						Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
					},
					Stack: []cage_errors.Location{
						{
							File: thisFile,
							Func: "TestSingleErrEvent",
							Line: "28",
							Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
						},
					},
					Cause: []cage_errors.Error{
						{
							Msg: "f1 wrap",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "testSingleErrEvent.func2",
								Line: "37",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
						{
							Msg: "f2 root",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "testSingleErrEvent.func1",
								Line: "34",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
					},
				},
			},
		},
		*event,
	)

	config := cage_errors.Config{ // inverse of defaults except for Config.Stack
		Cause:         false,
		EventLoc:      false,
		RuntimeFrames: true,
		Stack:         true,
		TestingFrames: true,
	}
	event, eventErr = cage_errors.NewConfiguredEvent(config, err)
	require.NoError(t, eventErr)

	require.Exactly(
		t,
		cage_errors.Location{
			File: thisFile,
			Func: "testSingleErrEvent",
			Line: "107",
			Pkg:  "errors_test",
		},
		event.Loc,
	)
	require.Len(t, event.Errors, 1)
	require.Len(t, event.Errors[0].Stack, 3)
	require.Exactly(t, "final wrap: f1 wrap: f2 root", event.Errors[0].Msg)
	require.Exactly(
		t,
		cage_errors.Location{
			File: thisFile,
			Func: "testSingleErrEvent",
			Line: "39",
			Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
		},
		event.Errors[0].Loc,
	)
	require.Exactly(
		t,
		cage_errors.Location{
			File: thisFile,
			Func: "TestSingleErrEvent",
			Line: "28",
			Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
		},
		event.Errors[0].Stack[0],
	)

	require.True(t, strings.HasSuffix(event.Errors[0].Stack[1].File, "testing/testing.go"))
	require.Exactly(t, "tRunner", event.Errors[0].Stack[1].Func)
	require.Regexp(t, digitRe, event.Errors[0].Stack[1].Line)
	require.Exactly(t, "testing", event.Errors[0].Stack[1].Pkg)

	require.True(t, strings.Contains(event.Errors[0].Stack[2].File, "runtime/"))
	require.Exactly(t, "goexit", event.Errors[0].Stack[2].Func)
	require.Regexp(t, digitRe, event.Errors[0].Stack[2].Line)
	require.Exactly(t, "runtime", event.Errors[0].Stack[2].Pkg)

	require.Nil(t, event.Errors[0].Cause)

	config = cage_errors.DefaultConfig()
	config.Stack = false
	event, eventErr = cage_errors.NewConfiguredEvent(config, err)
	require.NoError(t, eventErr)

	require.Exactly(
		t,
		cage_errors.Event{
			Loc: cage_errors.Location{
				File: thisFile,
				Func: "testSingleErrEvent",
				Line: "158",
				Pkg:  "errors_test",
			},
			Errors: []cage_errors.Error{
				{
					Msg: "final wrap",
					Loc: cage_errors.Location{
						File: thisFile,
						Func: "testSingleErrEvent",
						Line: "39",
						Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
					},
					Stack: nil,
					Cause: []cage_errors.Error{
						{
							Msg: "f1 wrap",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "testSingleErrEvent.func2",
								Line: "37",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
						{
							Msg: "f2 root",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "testSingleErrEvent.func1",
								Line: "34",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
					},
				},
			},
		},
		*event,
	)
}

func TestMultiErrEvent(t *testing.T) {
	testMultiErrEvent(t)
}

func testMultiErrEvent(t *testing.T) {
	var errs []error

	f2 := func() error {
		return errors.New("f2 root")
	}
	f1 := func() error {
		return errors.Wrap(f2(), "f1 wrap")
	}
	errs = append(errs, errors.Wrap(f1(), "first"))

	f4 := func() error {
		return errors.New("f4 root")
	}
	f3 := func() error {
		return errors.Wrap(f4(), "f3 wrap")
	}
	errs = append(errs, errors.Wrap(f3(), "second"))

	event, eventErr := cage_errors.NewEvent(errs...)
	require.NoError(t, eventErr)

	require.Exactly(
		t,
		cage_errors.Event{
			Loc: cage_errors.Location{
				File: thisFile,
				Func: "testMultiErrEvent",
				Line: "234",
				Pkg:  "errors_test",
			},
			Errors: []cage_errors.Error{
				{
					Msg: "first",
					Loc: cage_errors.Location{
						File: thisFile,
						Func: "testMultiErrEvent",
						Line: "224",
						Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
					},
					Stack: []cage_errors.Location{
						{
							File: thisFile,
							Func: "TestMultiErrEvent",
							Line: "212",
							Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
						},
					},
					Cause: []cage_errors.Error{
						{
							Msg: "f1 wrap",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "testMultiErrEvent.func2",
								Line: "222",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
						{
							Msg: "f2 root",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "testMultiErrEvent.func1",
								Line: "219",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
					},
				},
				{
					Msg: "second",
					Loc: cage_errors.Location{
						File: thisFile,
						Func: "testMultiErrEvent",
						Line: "232",
						Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
					},
					Stack: []cage_errors.Location{
						{
							File: thisFile,
							Func: "TestMultiErrEvent",
							Line: "212",
							Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
						},
					},
					Cause: []cage_errors.Error{
						{
							Msg: "f3 wrap",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "testMultiErrEvent.func4",
								Line: "230",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
						{
							Msg: "f4 root",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "testMultiErrEvent.func3",
								Line: "227",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
					},
				},
			},
		},
		*event,
	)
}

type S struct{}

func (s S) Method1() error {
	return errors.Wrap(s.Method2(), "Method1 wrap")
}
func (s S) Method2() error {
	return errors.New("Method2 root")
}

func TestMethodFrameInEvent(t *testing.T) {
	event, eventErr := cage_errors.NewEvent(errors.Wrap(S{}.Method1(), "final wrap"))
	require.NoError(t, eventErr)

	require.Exactly(
		t,
		cage_errors.Event{
			Loc: cage_errors.Location{
				File: thisFile,
				Func: "TestMethodFrameInEvent",
				Line: "345",
				Pkg:  "errors_test",
			},
			Errors: []cage_errors.Error{
				{
					Msg: "final wrap",
					Loc: cage_errors.Location{
						File: thisFile,
						Func: "TestMethodFrameInEvent",
						Line: "345",
						Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
					},
					Stack: nil,
					Cause: []cage_errors.Error{
						{
							Msg: "Method1 wrap",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "S.Method1",
								Line: "338",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
						{
							Msg: "Method2 root",
							Loc: cage_errors.Location{
								File: thisFile,
								Func: "S.Method2",
								Line: "341",
								Pkg:  "github.com/codeactual/transplant/internal/cage/errors_test",
							},
							Stack: nil,
							Cause: nil,
						},
					},
				},
			},
		},
		*event,
	)
}
