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

	// Assign user to department 1 (should match the test conversations)
	query := `INSERT IGNORE INTO user_departments (user_id, department_id) VALUES (?, ?)`

	_, err = db.Exec(query,
		"22222222-2222-2222-2222-222222222222",
		1, // Department ID 1
	)

	if err != nil {
		log.Fatal("Failed to assign user to department:", err)
	}

	fmt.Println("✓ Test user assigned to department 1")
}
