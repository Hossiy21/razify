package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var fixCmd = &cobra.Command{
	Use:   "fix [env-file] [example-file]",
	Short: "Sync missing keys from your .env.example to your .env file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		envFile := args[0]
		exampleFile := args[1]
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		fmt.Printf("\nFixing %s using template %s...\n\n", envFile, exampleFile)

		added, err := RunFix(envFile, exampleFile, dryRun)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		success := color.New(color.FgGreen, color.Bold)
		if added > 0 {
			if dryRun {
				fmt.Printf("\nDry run: %d keys would be added.\n", added)
			} else {
				success.Printf("\n✔ Successfully added %d missing keys to %s!\n", added, envFile)
			}
		} else {
			success.Println("\n✔ No missing keys found. Your .env is already in sync!")
		}
	},
}

func RunFix(envFile, exampleFile string, dryRun bool) (int, error) {
	// 1. Get existing keys in .env
	envVars, err := parseEnvFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	if os.IsNotExist(err) {
		envVars = make(map[string]string)
	}

	// 2. Get keys from example
	exampleVars, err := ParseEnvWithMetadata(exampleFile)
	if err != nil {
		return 0, err
	}

	// 3. Find missing keys
	var toAdd []EnvVar
	for _, ev := range exampleVars {
		if _, exists := envVars[ev.Key]; !exists {
			toAdd = append(toAdd, ev)
		}
	}

	if len(toAdd) == 0 {
		return 0, nil
	}

	if dryRun {
		for _, ev := range toAdd {
			fmt.Printf("  + Would add: %s (default: %s)\n", ev.Key, ev.Value)
		}
		return len(toAdd), nil
	}

	// 4. Append missing keys to .env
	f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Add a separator if the file is not empty
	info, err := f.Stat()
	if err == nil && info.Size() > 0 {
		f.WriteString("\n# --- Added by Razify Fix ---\n")
	}

	for _, ev := range toAdd {
		line := fmt.Sprintf("%s=%s\n", ev.Key, ev.Value)
		if _, err := f.WriteString(line); err != nil {
			return 0, err
		}
		fmt.Printf("  + Added: %s\n", ev.Key)
	}

	return len(toAdd), nil
}

func init() {
	rootCmd.AddCommand(fixCmd)
	fixCmd.Flags().BoolP("dry-run", "d", false, "Show what would be added without modifying the file")
}
