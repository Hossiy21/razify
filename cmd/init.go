package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive wizard to bootstrap your .env.example file",
	Run: func(cmd *cobra.Command, args []string) {
		bold := color.New(color.Bold)
		cyan := color.New(color.FgCyan)
		green := color.New(color.FgGreen)

		bold.Println("\n🚀 Welcome to the Razify Setup Wizard!")
		fmt.Println("This will help you create a professional .env.example with validation tags.")

		filename := ".env.example"
		if _, err := os.Stat(filename); err == nil {
			fmt.Printf("⚠️  %s already exists. Overwrite? (y/N): ", filename)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" {
				fmt.Println("Aborted.")
				return
			}
		}

		reader := bufio.NewReader(os.Stdin)
		var entries []string

		for {
			cyan.Print("\nEnter variable name (or leave blank to finish): ")
			key, _ := reader.ReadString('\n')
			key = strings.TrimSpace(key)
			if key == "" {
				break
			}

			fmt.Print("Description / Comment: ")
			comment, _ := reader.ReadString('\n')
			comment = strings.TrimSpace(comment)

			fmt.Print("Default value (optional): ")
			val, _ := reader.ReadString('\n')
			val = strings.TrimSpace(val)

			fmt.Print("Is this required? (y/N): ")
			req, _ := reader.ReadString('\n')
			reqStr := ""
			if strings.ToLower(strings.TrimSpace(req)) == "y" {
				reqStr = " @required=true"
			}

			fmt.Print("Type (int, bool, url, or leave blank): ")
			vType, _ := reader.ReadString('\n')
			vType = strings.TrimSpace(vType)
			typeStr := ""
			if vType != "" {
				typeStr = fmt.Sprintf(" @type=%s", vType)
			}

			// Format the entry
			entry := ""
			if comment != "" || reqStr != "" || typeStr != "" {
				entry += fmt.Sprintf("# %s%s%s\n", comment, reqStr, typeStr)
			}
			entry += fmt.Sprintf("%s=%s\n", key, val)
			entries = append(entries, entry)
		}

		if len(entries) == 0 {
			fmt.Println("\nNo variables added. Nothing to save.")
			return
		}

		content := strings.Join(entries, "\n")
		err := os.WriteFile(filename, []byte(content), 0644)
		if err != nil {
			fmt.Printf("\n❌ Error writing file: %v\n", err)
			os.Exit(1)
		}

		green.Printf("\n✔ Successfully created %s with %d variables!\n", filename, len(entries))
		fmt.Println("Next steps: Run 'razify fix .env .env.example' to sync your local environment.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
