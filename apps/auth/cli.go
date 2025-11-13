package auth

import (
	"fmt"
	"log"
	"os"

	"github.com/getevo/evo/v2/lib/args"
	"github.com/getevo/evo/v2/lib/db"
)

// CreateAdminUser creates an admin user via CLI
func CreateAdminUser() {
	email := args.Get("-email")
	password := args.Get("-password")
	name := args.Get("-name")
	lastName := args.Get("-lastname")

	if email == "" || password == "" || name == "" {
		fmt.Println("Usage: ./homa -create-admin -email admin@example.com -password secret123 -name Admin -lastname User")
		os.Exit(1)
	}

	// Check if user already exists
	var existingUser User
	if err := db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		// User exists, reset password
		if err := existingUser.SetPassword(password); err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}

		// Update user information if provided
		if lastName != "" {
			existingUser.LastName = lastName
		}
		existingUser.Name = name
		existingUser.DisplayName = name + " " + existingUser.LastName
		existingUser.Type = UserTypeAdministrator

		// Save updated user
		if err := db.Save(&existingUser).Error; err != nil {
			log.Fatalf("Failed to update admin user: %v", err)
		}

		fmt.Printf("Admin user already existed - password has been reset:\n")
		fmt.Printf("Email: %s\n", existingUser.Email)
		fmt.Printf("Name: %s %s\n", existingUser.Name, existingUser.LastName)
		fmt.Printf("Type: %s\n", existingUser.Type)
		fmt.Printf("ID: %s\n", existingUser.UserID.String())
		return
	}

	// Create new admin user
	user := User{
		Name:        name,
		LastName:    lastName,
		DisplayName: name + " " + lastName,
		Email:       email,
		Type:        UserTypeAdministrator,
	}

	// Set password
	if err := user.SetPassword(password); err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Save user to database
	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Printf("Admin user created successfully:\n")
	fmt.Printf("Email: %s\n", user.Email)
	fmt.Printf("Name: %s %s\n", user.Name, user.LastName)
	fmt.Printf("Type: %s\n", user.Type)
	fmt.Printf("ID: %s\n", user.UserID.String())
}
