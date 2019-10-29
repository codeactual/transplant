// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package errors

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/go-stack/stack"
	"github.com/pkg/errors"
)

// pkgErrsFullLocRe matches the '%s' output from Frame.Format in order to capture the
// package/function name and file path/line.
//
// Aligns with:
//   https://github.com/pkg/errors/blob/30136e27e2ac8d167177e8a583aa4c3fea5be833/stack.go#L63
var pkgErrsFullLocRe *regexp.Regexp

func init() {
	pkgErrsFullLocRe = regexp.MustCompile(`(\S+)\n\t(.*)`)
}

// causer is a copy of the pkg/errors interface.
type causer interface {
	Cause() error
}

// stackTracer is a copy of the pkg/errors interface.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// Event aims to provide error detail for inclusion in a structured log.
//
// It omits a time value, assuming the structured logger will include it.
type Event struct {
	// Loc is the event creation site.
	Loc Location

	// Errors holds one or more errors collected at the event creation site.
	Errors []Error
}

type Location struct {
	// File is an absolute path or relative path from GOROOT/GOPATH.
	File string

	Func string
	Line string
	Pkg  string
}

type Error struct {
	// Msg is message from the topmost error.
	Msg string

	// Loc is the error creation site, if available (e.g. from pkg/errors).
	Loc Location

	// Stack includes locations leading to, but excluding, Error.Loc, if available (e.g. from pkg/errors).
	Stack []Location `json:",omitempty"`

	// Cause includes the errors leading to, but excluding, this Error, if available (e.g. from pkg/errors).
	Cause []Error `json:",omitempty"`
}

type Config struct {
	// RuntimeFrames is true if stack traces should retain frames from the runtime package.
	RuntimeFrames bool

	// TestingFrames is true if stack traces should retain frames from the testing package.
	TestingFrames bool

	// EventLoc is true if Event.Loc should be populated.
	EventLoc bool

	// Stack is true if Error.Stack should be populated.
	Stack bool

	// Cause is true if Error.Cause should be populated and all Error.Msg values should be restored
	// to their original values (without messages of prior causes appended).
	Cause bool
}

func DefaultConfig() Config {
	return Config{
		Cause:    true,
		EventLoc: true,
		Stack:    true,
	}
}

func NewConfiguredEvent(config Config, errs ...error) (*Event, error) {
	return newConfiguredEvent(config, errs...)
}

func NewEvent(errs ...error) (*Event, error) {
	return newConfiguredEvent(DefaultConfig(), errs...)
}

func newConfiguredEvent(config Config, errs ...error) (*Event, error) {
	event := &Event{}

	caller := stack.Caller(2) // skip this function and NewEvent/NewConfiguredEvent

	event.Loc = Location{
		File: fmt.Sprintf("%#s", caller), // use full path to support modules
		Func: fmt.Sprintf("%n", caller),
		Line: fmt.Sprintf("%d", caller),
		Pkg:  fmt.Sprintf("%k", caller),
	}

	for _, err := range errs {
		event.Errors = append(event.Errors, newError(config, err))
	}

	return event, nil
}

func newError(config Config, err error) (computed Error) {
	if err == nil {
		return Error{}
	}

	computed.Msg = err.Error()

	if e, ok := err.(stackTracer); ok {
		for n, f := range e.StackTrace() {
			matches := pkgErrsFullLocRe.FindAllStringSubmatch(fmt.Sprintf("%+v", f), 1)
			if matches == nil {
				continue
			}

			loc := Location{}
			funcName := fmt.Sprintf("%n", f)
			funcNameIdx := strings.LastIndex(matches[0][1], funcName)

			loc.Pkg = matches[0][1][:funcNameIdx-1]
			loc.Func = funcName

			if !config.RuntimeFrames && loc.Pkg == "runtime" {
				continue
			}
			if !config.TestingFrames && loc.Pkg == "testing" {
				continue
			}

			if fileAndLine := strings.Split(matches[0][2], ":"); len(fileAndLine) == 2 {
				loc.File = fileAndLine[0]
				loc.Line = fileAndLine[1]
			}

			// Don't redundantly add the error site as the first Stack element.
			if n == 0 {
				computed.Loc = loc
				if !config.Stack {
					break
				}
			} else {
				computed.Stack = append(computed.Stack, loc)
			}
		}
	}

	if !config.Cause {
		return computed
	}

	var causeMsgs []string

	for {
		c, ok := err.(causer)
		if !ok {
			break
		}
		cause := c.Cause()
		if cause == err {
			break
		}

		// It's unclear why these links appear in the cause chain, but this check filters them out. Links:
		// - The topmost error.
		// - Duplicates of errors in the link.
		config.Cause = false
		config.Stack = false
		computedCause := newError(config, cause)
		if computedCause.Loc.File == "" {
			err = cause
			continue
		}

		computed.Cause = append(computed.Cause, computedCause)
		causeMsgs = append(causeMsgs, computedCause.Msg)

		err = cause
	}

	// Restore all error messages to their original values, trimming away messages from errors
	// deeper in the casual chain. The intent is to make Error.[]Cause.Msg, and especially
	// Error.Msg, more readable.
	var prevMsg string
	for n := len(causeMsgs) - 1; n >= 0; n-- {
		if prevMsg == "" {
			prevMsg = computed.Cause[n].Msg
			computed.Msg = strings.TrimSuffix(computed.Msg, ": "+prevMsg)
		} else {
			computed.Cause[n].Msg = strings.TrimSuffix(computed.Cause[n].Msg, ": "+prevMsg)
			prevMsg = computed.Cause[n].Msg
			computed.Msg = strings.TrimSuffix(computed.Msg, ": "+prevMsg)
		}
	}

	return computed
}

func WriteErrList(w io.Writer, errs ...error) {
	errsLen := len(errs)
	if errsLen > 0 {
		fmt.Fprintf(w, "\n") // in case the cursor at the end of a "starting X ..." line
		for n, err := range errs {
			fmt.Fprintf(w, "Error (%d/%d): %s\n", n+1, errsLen, err)
		}
	}
}

func Append(errs *[]error, err error) bool {
	if err != nil {
		*errs = append(*errs, err)
		return true
	}
	return false
}
