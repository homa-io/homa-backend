package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Connect to MySQL
	dsn := "root:root@tcp(127.0.0.1:3306)/homa?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Dropping old ticket-related tables...")
	fmt.Println("=====================================")

	// Tables to drop (old ticket system replaced by conversations)
	tables := []string{
		"ticket_tags",
		"ticket_assignments",
		"tickets",
	}

	for _, table := range tables {
		fmt.Printf("\nDropping table: %s...", table)
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
		_, err := db.Exec(query)
		if err != nil {
			log.Printf(" ✗ Failed: %v\n", err)
		} else {
			fmt.Printf(" ✓ Dropped successfully\n")
		}
	}

	fmt.Println("\n✓ Cleanup completed!")
}
