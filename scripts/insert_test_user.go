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

	// Insert test user (agent)
	query := `INSERT IGNORE INTO users (id, name, last_name, display_name, email, password_hash, type, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`

	_, err = db.Exec(query,
		"22222222-2222-2222-2222-222222222222",
		"Jane",
		"Smith",
		"Jane Smith",
		"agent1@homa.com",
		"$2a$14$7BDZskTQX9JGCOq8ZDXKdu4O5hAu2/7D5QI9L1DjG6MqgzF1jQZ7K", // password: password123
		"agent",
	)

	if err != nil {
		log.Fatal("Failed to insert user:", err)
	}

	fmt.Println("✓ Test user inserted successfully")
	fmt.Println("  ID: 22222222-2222-2222-2222-222222222222")
	fmt.Println("  Email: agent1@homa.com")
	fmt.Println("  Type: agent")
}
