// +build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getevo/evo/v2/lib/settings"
	"github.com/iesreza/homa-backend/apps/storage"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MigrateAvatarsToS3 migrates existing avatars from local storage to S3
// Run with: go run scripts/migrate_avatars_to_s3.go -c /path/to/config.yml

func main() {
	// Load config
	configPath := "/home/evo/config/homa-backend/config.yml"
	for i, arg := range os.Args {
		if arg == "-c" && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	// Load settings
	if err := settings.Init(configPath); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize S3
	if err := storage.Initialize(); err != nil {
		fmt.Printf("Failed to initialize S3: %v\n", err)
		os.Exit(1)
	}

	if !storage.IsEnabled() {
		fmt.Println("S3 is not enabled. Please enable S3 in config first.")
		os.Exit(1)
	}

	// Connect to database
	dbConfig := settings.Get("Database")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?%s",
		dbConfig.Get("Username").String(),
		dbConfig.Get("Password").String(),
		dbConfig.Get("Server").String(),
		dbConfig.Get("Database").String(),
		dbConfig.Get("Params").String(),
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	storagePath := settings.Get("STORAGE.PATH").String()
	if storagePath == "" {
		storagePath = "uploads"
	}

	ctx := context.Background()
	var migratedCount int
	var errorCount int

	// Migrate user avatars
	fmt.Println("Migrating user avatars...")
	rows, err := db.Raw("SELECT id, avatar FROM users WHERE avatar IS NOT NULL AND avatar != '' AND avatar LIKE '/uploads/%'").Rows()
	if err != nil {
		fmt.Printf("Failed to query users: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id, avatar string
			if err := rows.Scan(&id, &avatar); err != nil {
				continue
			}

			// Convert /uploads/avatars/users/xxx.jpg to file path
			relativePath := strings.TrimPrefix(avatar, "/uploads/")
			filePath := filepath.Join(storagePath, relativePath)

			// Read file
			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("  Failed to read %s: %v\n", filePath, err)
				errorCount++
				continue
			}

			// Upload to S3
			s3Key := "avatars/users/" + filepath.Base(filePath)
			if err := storage.Upload(ctx, s3Key, data, "image/jpeg"); err != nil {
				fmt.Printf("  Failed to upload %s: %v\n", s3Key, err)
				errorCount++
				continue
			}

			// Update database
			if err := db.Exec("UPDATE users SET avatar = ? WHERE id = ?", s3Key, id).Error; err != nil {
				fmt.Printf("  Failed to update user %s: %v\n", id, err)
				errorCount++
				continue
			}

			fmt.Printf("  Migrated user %s: %s -> %s\n", id, avatar, s3Key)
			migratedCount++
		}
	}

	// Migrate client avatars
	fmt.Println("\nMigrating client avatars...")
	rows, err = db.Raw("SELECT id, avatar FROM clients WHERE avatar IS NOT NULL AND avatar != '' AND avatar LIKE '/uploads/%'").Rows()
	if err != nil {
		fmt.Printf("Failed to query clients: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id, avatar string
			if err := rows.Scan(&id, &avatar); err != nil {
				continue
			}

			// Convert /uploads/avatars/clients/xxx.jpg to file path
			relativePath := strings.TrimPrefix(avatar, "/uploads/")
			filePath := filepath.Join(storagePath, relativePath)

			// Read file
			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("  Failed to read %s: %v\n", filePath, err)
				errorCount++
				continue
			}

			// Upload to S3
			s3Key := "avatars/clients/" + filepath.Base(filePath)
			if err := storage.Upload(ctx, s3Key, data, "image/jpeg"); err != nil {
				fmt.Printf("  Failed to upload %s: %v\n", s3Key, err)
				errorCount++
				continue
			}

			// Update database
			if err := db.Exec("UPDATE clients SET avatar = ? WHERE id = ?", s3Key, id).Error; err != nil {
				fmt.Printf("  Failed to update client %s: %v\n", id, err)
				errorCount++
				continue
			}

			fmt.Printf("  Migrated client %s: %s -> %s\n", id, avatar, s3Key)
			migratedCount++
		}
	}

	fmt.Printf("\n=== Migration Complete ===\n")
	fmt.Printf("Migrated: %d\n", migratedCount)
	fmt.Printf("Errors: %d\n", errorCount)

	if migratedCount > 0 {
		fmt.Println("\nNote: Old local files were NOT deleted. You can manually remove them after verifying the migration.")
		fmt.Printf("Local avatar directory: %s/avatars/\n", storagePath)
	}
}
