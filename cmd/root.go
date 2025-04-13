/*
Copyright © 2025 Dyam Marcano dyam.marcano@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"github.com/inovacc/goinstall/internal/installer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
)

var rootCmd = &cobra.Command{
	Use:   "goinstall",
	Short: "Install, update or remove Go modules with ease",
	Long: `goinstall is a CLI tool that helps manage Go module installations.

You can use it to fetch, install, update, or remove Go packages
from your environment with a clean and idiomatic approach.`,
	Args: cobra.ArbitraryArgs, // <- allows module path as an argument
	RunE: func(cmd *cobra.Command, args []string) error {
		remove, _ := cmd.Flags().GetBool("remove")
		update, _ := cmd.Flags().GetBool("update")

		if remove && update {
			return fmt.Errorf("flags --remove and --update cannot be used together")
		}

		if len(args) == 0 {
			return fmt.Errorf("module name is required. Usage: goinstall [flags] <module>")
		}

		return installer.Installer(cmd, args)
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.Flags().BoolP("remove", "r", false, "Remove go install module")
	rootCmd.Flags().BoolP("update", "u", false, "Update go install module")

	cobra.CheckErr(viper.BindPFlag("remove", rootCmd.Flags().Lookup("remove")))
	cobra.CheckErr(viper.BindPFlag("update", rootCmd.Flags().Lookup("update")))

	viper.Set("installPath", dbPath())
}

func dbPath() string {
	if custom := os.Getenv("GOINSTALL_DB_PATH"); custom != "" {
		return custom
	}

	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Local", "goinstall", "modules.db")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "goinstall", "modules.db")
	default: // Linux/Unix with XDG
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			xdgData = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(xdgData, "goinstall", "modules.db")
	}
}
