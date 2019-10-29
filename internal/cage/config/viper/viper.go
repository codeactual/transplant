// Copyright (C) 2019 The CodeActual Go Environment Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package viper

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	std_viper "github.com/spf13/viper"
)

// ReadInConfig wraps viper.ReadInConfig to resolve relative paths,
// auto-detect the file type, etc.
func ReadInConfig(v *std_viper.Viper, configPath string) (err error) {
	v.SetConfigFile(configPath)
	return errors.WithStack(v.ReadInConfig())
}

// NewEnvSpace returns an initialized Viper instance configured to read all keys from an
// environment variable prefix.
func NewEnvSpace(prefix string) *std_viper.Viper {
	v := std_viper.New()

	// When trying to access a config named 'allow-x', look for env named '<prefix>_allow_x'.
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	v.SetEnvPrefix(prefix)
	v.AutomaticEnv()
	return v
}

// IsSetInCommand provides a viper.IsSet alternative that works around a bug which causes
// IsSet to always return true if the config key is bound to cobra: https://github.com/spf13/viper/issues/276.
//
// The workaround logic doesn't support the case where a flag both has a default value and
// was also erroneously marked as required.
func IsSetInCommand(v *std_viper.Viper, c *cobra.Command, prefix, key string) bool {
	return os.Getenv(EnvPrefixedName(prefix, key)) != "" || c.Flag(key).Changed
}

// EnvPrefixedName replicates the private viper.mergeWithEnvPrefix.
func EnvPrefixedName(prefix, key string) string {
	return strings.ToUpper(prefix + "_" + key)
}
