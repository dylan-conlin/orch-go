// Package main provides the account command for managing Claude Max accounts.
// Extracted from main.go as part of the main.go refactoring.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage Claude Max accounts",
	Long: `Manage multiple Claude Max accounts for usage tracking and rate limit arbitrage.

Accounts are stored in ~/.orch/accounts.yaml with refresh tokens for switching.

Examples:
  orch-go account list              # List all saved accounts
  orch-go account switch personal   # Switch to 'personal' account
  orch-go account remove work       # Remove 'work' account`,
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved accounts",
	Long:  "List all saved accounts with their email and default status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountList()
	},
}

var accountSwitchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch to a saved account",
	Long: `Switch to a saved account by refreshing its OAuth tokens.

This updates the OpenCode auth file with new tokens from the saved refresh token.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountSwitch(args[0])
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a saved account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountRemove(args[0])
	},
}

var (
	// Account add command flags
	accountAddSetDefault bool
)

var accountAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new account via OAuth login",
	Long: `Add a new Claude Max account by initiating an OAuth login flow.

This command opens your browser for authentication with Anthropic. After
successful login, the refresh token is saved to ~/.orch/accounts.yaml.

The account can then be switched to using 'orch account switch <name>'.

Examples:
  orch-go account add personal           # Add account named 'personal'
  orch-go account add work --default     # Add as default account`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountAdd(args[0], accountAddSetDefault)
	},
}

func init() {
	accountAddCmd.Flags().BoolVar(&accountAddSetDefault, "default", false, "Set as default account")

	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountSwitchCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountAddCmd)
}

func runAccountList() error {
	accounts, err := account.ListAccountInfo()
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("No saved accounts")
		fmt.Println("\nTo add an account:")
		fmt.Println("  orch account add <name>")
		return nil
	}

	// Determine recommended account (without capacity fetcher — uses role-based default)
	recommended := account.RecommendAccount(accounts, nil)

	fmt.Printf("%-12s %-35s %-6s %-12s %-15s\n", "NAME", "EMAIL", "TIER", "ROLE", "STATUS")
	fmt.Printf("%s\n", strings.Repeat("-", 82))

	for _, acc := range accounts {
		tier := acc.Tier
		if tier == "" {
			tier = "-"
		}
		role := acc.Role
		if role == "" {
			role = "-"
		}
		status := ""
		if acc.Name == recommended {
			status = "[RECOMMENDED]"
		}
		if acc.IsDefault {
			if status != "" {
				status += " default"
			} else {
				status = "default"
			}
		}
		fmt.Printf("%-12s %-35s %-6s %-12s %-15s\n", acc.Name, acc.Email, tier, role, status)
	}

	return nil
}

func runAccountSwitch(name string) error {
	email, err := account.SwitchAccount(name)
	if err != nil {
		// Check if it's a token refresh error to provide actionable guidance
		if tokenErr, ok := err.(*account.TokenRefreshError); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", tokenErr.Error())
			fmt.Fprintf(os.Stderr, "\n%s\n", tokenErr.ActionableGuidance())
			return fmt.Errorf("token refresh failed for account '%s'", name)
		}
		return err
	}

	fmt.Printf("Switched to account: %s (%s)\n", name, email)
	return nil
}

func runAccountRemove(name string) error {
	cfg, err := account.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}

	if err := cfg.Remove(name); err != nil {
		return fmt.Errorf("failed to remove account: %w", err)
	}

	if err := account.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed account: %s\n", name)
	return nil
}

func runAccountAdd(name string, setDefault bool) error {
	// Check if account already exists
	cfg, err := account.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load accounts: %w", err)
	}

	if _, err := cfg.Get(name); err == nil {
		return fmt.Errorf("account '%s' already exists. Use 'orch account remove %s' first, or choose a different name", name, name)
	}

	fmt.Printf("Adding account '%s'...\n", name)
	fmt.Println()

	email, err := account.AddAccount(name, setDefault, nil)
	if err != nil {
		return fmt.Errorf("failed to add account: %w", err)
	}

	fmt.Println()
	fmt.Printf("Successfully added account '%s' (%s)\n", name, email)
	if setDefault {
		fmt.Println("Set as default account")
	}
	fmt.Println("\nThe account is now active. Use 'orch account switch <name>' to change accounts later.")

	return nil
}
