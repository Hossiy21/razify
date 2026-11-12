/*
Copyright © 2026 hossiy21
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/Hossiy21/razify/internal/reporter"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var asciiLogo = `
  ██████╗  █████╗ ███████╗██╗███████╗██╗   ██╗
  ██╔══██╗██╔══██╗╚══███╔╝██║██╔════╝╚██╗ ██╔╝
  ██████╔╝███████║  ███╔╝ ██║█████╗   ╚████╔╝ 
  ██╔══██╗██╔══██║ ███╔╝  ██║██╔══╝    ╚██╔╝  
  ██║  ██║██║  ██║███████╗██║██║        ██║   
  ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝╚═╝        ╚═╝   `

var overrideExitCode int

func PrintBanner() {
	c := color.New(color.FgMagenta, color.Bold)
	sub := color.New(color.FgCyan)
	c.Println(asciiLogo)
	sub.Printf("  ⚡ Configuration Integrity Engine %s\n\n", CurrentVersion)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "razify",
	Short: "The missing CLI tool for .env file management",
	Long: `Razify is a powerful CLI tool designed to solve environment variable hell.
It allows you to diff, scan, validate, document, and audit your .env files
to ensure your application configuration is secure, documented, and consistent
across your entire team.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if overrideExitCode == 0 || overrideExitCode == 2 {
			return fmt.Errorf("invalid --exit-code value %d: code 0 is reserved for clean passes, and 2 is reserved for fatal errors", overrideExitCode)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		PrintBanner()
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(reporter.ExitError)
	}
}

// getViolationExitCode returns the configured exit code when violations are detected.
func getViolationExitCode() int {
	if overrideExitCode >= 0 {
		return overrideExitCode
	}
	return reporter.ExitViolation
}

func init() {
	rootCmd.PersistentFlags().IntVar(&overrideExitCode, "exit-code", -1, "Override the exit code returned when violations are found (default 1, values 0 and 2 reserved)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
