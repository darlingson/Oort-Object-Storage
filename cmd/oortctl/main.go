package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/darlingson/Oort-Object-Storage/internal/client"
)

var (
	serverURL string
	apiKey    string
)

var rootCmd = &cobra.Command{
	Use:   "oortctl",
	Short: "Oort Object Storage CLI",
}

func execute() {
	rootCmd.PersistentFlags().StringVarP(
		&serverURL, "server", "s", "http://localhost:3333", "server URL",
	)
	rootCmd.PersistentFlags().StringVar(
		&apiKey, "api-key", "", "API key for machine auth",
	)

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	execute()
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login and store JWT token",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" || password == "" {
			fmt.Print("Email: ")
			fmt.Scanln(&email)
			fmt.Print("Password: ")
			fmt.Scanln(&password)
		}

		c := client.New(serverURL)
		result, err := c.Login(email, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := client.WriteToken(result.Token); err != nil {
			return fmt.Errorf("failed to store token: %w", err)
		}

		fmt.Printf("Logged in as %s\n", result.User.Email)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored JWT token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.DeleteToken(); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}
		fmt.Println("Logged out")
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("email", "e", "", "email address")
	loginCmd.Flags().StringP("password", "p", "", "password")
}
