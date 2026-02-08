package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  `Commands for managing users in WeatherMaestro.`,
}

var createUserCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new admin user",
	Long:  `Create a new admin user.`,
	RunE:  runCreateUser,
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(createUserCmd)
}

func runCreateUser(cmd *cobra.Command, args []string) error {
	dbManager := cmd.Context().Value("dbManager").(*database.DatabaseManager)

	reader := bufio.NewReader(os.Stdin)

	// Get username
	fmt.Print("Enter username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	// Get password
	fmt.Print("Enter password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // New line after password input

	password := string(passwordBytes)
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}
	fmt.Println() // New line after password input

	if password != string(confirmBytes) {
		return fmt.Errorf("passwords do not match")
	}

	// Create user
	user, err := dbManager.CreateUser(cmd.Context(), username, password)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("User created successfully!\n")
	fmt.Printf("ID: %s\n", user.ID)
	fmt.Printf("Username: %s\n", user.Username)
	fmt.Printf("Created: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))

	return nil
}
