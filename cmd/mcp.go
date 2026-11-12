package cmd

import (
	"errors"
	"os"

	"github.com/Hossiy21/razify/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol (MCP) server for agent-safe environment access",
	Long: `Starts a JSON-RPC 2.0 Model Context Protocol (MCP) server over standard I/O (stdio).
Exposes safe environment schema inspection, candidate configuration validation, key diffing,
and health auditing to AI coding agents (Cursor, Claude Desktop, Antigravity, Gemini CLI)
without ever exposing raw production secret values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		if workspace == "" {
			return errors.New("workspace directory must be explicitly specified via --workspace")
		}

		validWs, err := mcp.ValidateWorkspaceDir(workspace)
		if err != nil {
			return err
		}

		server := mcp.NewServer(validWs, os.Stdin, os.Stdout, os.Stderr, CurrentVersion)
		return server.Serve()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().StringP("workspace", "w", "", "Workspace directory for environment and schema inspection (required)")
}
