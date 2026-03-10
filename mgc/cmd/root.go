// Package cmd provides the CLI command structure for mgc.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version string

	// Global flags
	outputFormat string
	queryFilter  string
	tenantID     string
	debugMode    bool
)

// rootCmd is the base command for mgc.
var rootCmd = &cobra.Command{
	Use:   "mgc",
	Short: "Microsoft Graph CLI — a cross-platform tool for the Microsoft Graph API",
	Long: `mgc is a cross-platform, agent-friendly command-line tool for the Microsoft Graph API.

It lets you manage users, groups, mail, calendar, files, and more from your terminal
or from scripts and AI agents.

Get started:
  mgc auth login        Authenticate with Microsoft Graph
  mgc users list        List users in your tenant
  mgc --help            Show this help

Documentation: https://github.com/chenxizhang/testagent/blob/main/docs/ARCHITECTURE.md`,
}

// Execute runs the root command. Called from main().
func Execute(v string) {
	version = v
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/mgc/config.json)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", "output format: json, table, yaml, tsv (default: table)")
	rootCmd.PersistentFlags().StringVarP(&queryFilter, "query", "q", "", "JMESPath query to filter output")
	rootCmd.PersistentFlags().StringVar(&tenantID, "tenant", "", "Azure tenant ID (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug logging (shows HTTP requests)")

	// Bind flags to viper
	_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("tenant", rootCmd.PersistentFlags().Lookup("tenant"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	// Register subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newConfigCmd())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Look for config in ~/.config/mgc/
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(fmt.Sprintf("%s/.config/mgc", home))
		}
		viper.SetConfigName("config")
		viper.SetConfigType("json")
	}

	// Environment variables with MGC_ prefix
	viper.SetEnvPrefix("MGC")
	viper.AutomaticEnv()

	// Read config file (ignore error if not found)
	_ = viper.ReadInConfig()
}

// GetOutputFormat returns the effective output format (flag > config > default).
func GetOutputFormat() string {
	if outputFormat != "" {
		return outputFormat
	}
	configured := viper.GetString("default_output")
	if configured != "" {
		return configured
	}
	return "table"
}

// GetQueryFilter returns the JMESPath query string.
func GetQueryFilter() string {
	return queryFilter
}

// GetTenantID returns the effective tenant ID (flag > config).
func GetTenantID() string {
	if tenantID != "" {
		return tenantID
	}
	return viper.GetString("default_tenant")
}

// IsDebug returns true if debug mode is enabled.
func IsDebug() bool {
	return debugMode || viper.GetBool("debug")
}
