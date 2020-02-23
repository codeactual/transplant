// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cobra

import (
	"fmt"
	"strings"

	tp_viper "github.com/codeactual/transplant/internal/third_party/github.com/config/viper"
	"github.com/pkg/errors"
	std_cobra "github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	cage_viper "github.com/codeactual/transplant/internal/cage/config/viper"
	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// Config provides viper integration and enforces prefixed environment
// variables.
//
// It has a similar method set as cobra.Mixin implementations but is not a mixin.
// It is directly used in cobra.NewCommand to automatically add the functionality.
type Config struct {
	*viper.Viper

	// envPrefix records the value passed to Init for which there is no viper method,
	// e.g. GetEnvPrefix, to retrieve later.
	//
	// If viper.SetEnvPrefix is called again after Init, the value will be stale.
	envPrefix string

	requiredKeys map[string]bool

	cmd *std_cobra.Command
}

// Init creates the config storage instance.
func (c *Config) Init(envPrefix string, cmd *std_cobra.Command) *std_cobra.Command {
	c.Viper = cage_viper.NewEnvSpace(envPrefix)

	c.cmd = cmd
	c.envPrefix = envPrefix
	c.requiredKeys = make(map[string]bool)

	return cmd
}

// BindEnvToAllFlags binds all flags in the command to the viper instance.
func (c *Config) BindEnvToAllFlags(cmd *std_cobra.Command) {
	if err := c.Viper.BindPFlags(cmd.Flags()); err != nil {
		panic(errors.WithStack(errors.Errorf("failed to bind all flags to environment variable aliases")))
	}
}

// SetRequired registers config keys which must be provided as a flag or environment value.
//
// It panics if any key is invalid.
func (c *Config) SetRequired(keys ...string) {
	validKeys := make(map[string]bool)
	c.cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		validKeys[f.Name] = true
	})

	for _, key := range keys {
		if !validKeys[key] {
			panic(errors.WithStack(errors.Errorf("invalid required key selection [%s]", key)))
		}
		c.requiredKeys[key] = true
	}
}

// PreRun checks for missing required flags, completes viper/cobra binding, etc.
//
// It returns an error if any config is missing from both flags and environment.
// The error string contains a list of all missing config keys.
func (c *Config) PreRun() (showUsage bool, _ error) {
	if err := tp_viper.MergeConfig(c.cmd.Flags(), c.Viper); err != nil {
		return false, errors.WithStack(err)
	}

	missing := c.MissingRequiredKeyStrings()

	if len(missing) > 0 {
		errMsg := "Missing:"
		for _, s := range missing {
			errMsg += "\n\t" + s
		}
		return true, errors.New(errMsg)
	}

	return false, nil
}

// MissingRequiredKeyStrings returns a "--<flag key name>/<env key name>" element for each missing key.
func (c *Config) MissingRequiredKeyStrings() (missing []string) {
	for key := range c.requiredKeys {
		if !cage_viper.IsSetInCommand(c.Viper, c.cmd, c.envPrefix, key) {
			missing = append(missing, c.KeyUsageString(key))
		}
	}
	cage_strings.SortStable(missing)
	return missing
}

// RequiredKeyStrings returns a "--<flag key name>/<env key name>" element for each required key.
func (c *Config) RequiredKeyStrings() (required []string) {
	for key := range c.requiredKeys {
		required = append(required, c.KeyUsageString(key))
	}
	cage_strings.SortStable(required)
	return required
}

func (c *Config) KeyUsageString(key string) string {
	return fmt.Sprintf("--%s/%s", key, cage_viper.EnvPrefixedName(c.envPrefix, strings.ToUpper(key)))
}
