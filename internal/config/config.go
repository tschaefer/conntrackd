/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// InitConfig initializes the configuration using Viper.
// It reads from the specified config file or defaults to
// /etc/conntrackd/conntrackd.{yaml,json,toml}.
// Environment variables with the prefix CONNTRACKD_ can override config values.
func InitConfig(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("conntrackd")
		viper.AddConfigPath("/etc/conntrackd/")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("CONNTRACKD")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if cfgFile != "" {
				return fmt.Errorf("config file not found: %w", err)
			}
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	return nil
}
