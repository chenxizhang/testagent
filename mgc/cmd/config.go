package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mgc configuration",
	}

	configCmd.AddCommand(newConfigListCmd())
	configCmd.AddCommand(newConfigSetCmd())
	return configCmd
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings := viper.AllSettings()
			if len(settings) == 0 {
				fmt.Println("No configuration values set.")
				fmt.Fprintf(os.Stderr, "Config file: %s\n", viper.ConfigFileUsed())
				return nil
			}
			for k, v := range settings {
				fmt.Printf("%-25s = %v\n", k, v)
			}
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])
			val := args[1]

			validKeys := map[string]bool{
				"default_tenant": true,
				"default_output": true,
				"client_id":      true,
				"debug":          true,
			}
			if !validKeys[key] {
				return fmt.Errorf("unknown config key %q; valid keys: default_tenant, default_output, client_id, debug", key)
			}

			viper.Set(key, val)

			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("cannot determine home directory: %w", err)
				}
				configDir := fmt.Sprintf("%s/.config/mgc", home)
				if err := os.MkdirAll(configDir, 0700); err != nil {
					return fmt.Errorf("cannot create config directory: %w", err)
				}
				configFile = configDir + "/config.json"
			}

			if err := viper.WriteConfigAs(configFile); err != nil {
				return fmt.Errorf("cannot write config: %w", err)
			}

			fmt.Printf("Set %s = %s (saved to %s)\n", key, val, configFile)
			return nil
		},
	}
}
