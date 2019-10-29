// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package zap

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
	"github.com/spf13/cobra"
	std_zap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/codeactual/transplant/internal/cage/cli/handler"
	cage_errors "github.com/codeactual/transplant/internal/cage/errors"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
	cage_reflect "github.com/codeactual/transplant/internal/cage/reflect"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
	"github.com/codeactual/transplant/internal/ldflags"
)

const (
	newDirPerm = 0755
)

type Mixin struct {
	handler.IO

	*std_zap.Logger

	LogAppend bool   `usage:"Append events to preexisting file instead of truncating it"`
	LogFile   string `usage:"File to receive JSON log events"`
	LogLevel  string
}

// Implements cage/cli/handler.Mixin
func (m *Mixin) BindCobraFlags(cmd *cobra.Command) []string {
	cmd.Flags().StringVarP(&m.LogFile, "log-file", "", "", cage_reflect.GetFieldTag(*m, "LogFile", "usage"))
	cmd.Flags().StringVarP(&m.LogLevel, "log-level", "", zapcore.WarnLevel.String(), "Minimum level included in file: "+strings.Join(m.logLevels().Slice(), ", "))
	cmd.Flags().BoolVarP(&m.LogAppend, "log-append", "", true, cage_reflect.GetFieldTag(*m, "LogAppend", "usage"))
	return []string{}
}

// Implements cage/cli/handler/Mixin
func (m *Mixin) Name() string {
	return "cage/cli/handler/mixin/log/zap"
}

// Implements cage/cli/handler.PreRun
func (m *Mixin) PreRun(ctx context.Context, args []string) error {
	if m.LogFile == "" {
		m.Logger = std_zap.NewNop()
		return nil
	}

	var err error
	var runId string

	randId, err := ksuid.NewRandom()
	if err == nil {
		runId = randId.String()
	} else {
		return errors.Wrap(err, "failed to generate run ID for logger")
	}

	logCfg := std_zap.NewProductionConfig()

	// These will redundantly appear in every log event to make it easier for inclusion in bug reports,
	// rather than logging these fields once at a startup and asking the user for that entry or equivalent info.
	logCfg.InitialFields = map[string]interface{}{
		"version": ldflags.Version,
		"go": map[string]interface{}{
			"arch":    runtime.GOARCH,
			"os":      runtime.GOOS,
			"srcDirs": build.Default.SrcDirs(),
			"version": runtime.Version(),
		},
		"runId": runId,
		"args":  os.Args,
	}

	if err = os.MkdirAll(filepath.Dir(m.LogFile), newDirPerm); err != nil {
		return errors.Wrapf(err, "failed to create log dir [%s]", filepath.Dir(m.LogFile))
	}

	if !m.LogAppend {
		exists, _, existsErr := cage_file.Exists(m.LogFile)
		if existsErr != nil {
			return errors.Wrapf(existsErr, "failed to check if log file eixsts [%s]", m.LogFile)
		}

		if exists {
			if truncErr := os.Truncate(m.LogFile, 0); truncErr != nil {
				return errors.Wrapf(truncErr, "failed to truncate log file [%s]", m.LogFile)
			}
		}
	}

	logCfg.OutputPaths = []string{m.LogFile}
	logCfg.ErrorOutputPaths = []string{m.LogFile}
	logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if m.logLevels().Contains(m.LogLevel) {
		level := std_zap.NewAtomicLevel()
		if levelErr := level.UnmarshalText([]byte(m.LogLevel)); levelErr != nil {
			return errors.Wrapf(levelErr, "failed to apply selected log level [%s]", m.LogLevel)
		}
		logCfg.Level = level
	} else {
		return errors.Errorf("log level [%s] not found in available levels %v", m.LogLevel, m.logLevels().Slice())
	}

	logCfg.EncoderConfig.EncodeCaller = zapcore.FullCallerEncoder

	m.Logger, err = logCfg.Build()
	if err != nil {
		return errors.Wrap(err, "failed to configure logger")
	}

	return nil
}

func (m *Mixin) ErrToFile(errs ...error) {
	if m.Logger == nil {
		return
	}

	var fields []zapcore.Field

	event, eventErr := cage_errors.NewEvent(errs...)
	if eventErr == nil {
		fields = append(fields, std_zap.Any("errs", event))
	} else {
		eventErr = errors.Wrap(eventErr, "failed to parse error(s) for log")
		fields = append(fields, std_zap.NamedError("errParseErr", eventErr))
		fields = append(fields, std_zap.Errors("errs", errs))
	}

	m.Logger.Error(fmt.Sprintf("%d errors logged, see 'err#' and 'err#Verbose' keys", len(errs)), fields...)
}

// Implements cage/cli/handler.PostRun
func (m *Mixin) PostRun(ctx context.Context) {
	if err := m.Logger.Sync(); err != nil {
		fmt.Fprintf(m.Err(), "failed to flush events to log file [%s]: %s", m.LogFile, err)
	}
}

func (m *Mixin) ExitOnErr(code int, errs ...error) {
	errsLen := len(errs)
	if errsLen == 0 || (errsLen == 1 && errs[0] == nil) {
		return
	}

	cage_errors.WriteErrList(m.Err(), errs...)
	m.ErrToFile(errs...)

	if m.LogFile == "" {
		fmt.Fprintln(m.Err(), "To save a more detailed error list, see --log-* flags.")
	}

	os.Exit(code)
}

func (m *Mixin) logLevels() *cage_strings.Set {
	return cage_strings.NewSet().AddSlice([]string{
		zapcore.DebugLevel.String(),
		zapcore.InfoLevel.String(),
		zapcore.WarnLevel.String(),
		zapcore.ErrorLevel.String(),
	})
}

var _ handler.Mixin = (*Mixin)(nil)
var _ handler.PreRun = (*Mixin)(nil)
var _ handler.PostRun = (*Mixin)(nil)
