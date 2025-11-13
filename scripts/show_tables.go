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

	// Query all tables
	query := `SHOW TABLES`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Failed to query tables:", err)
	}
	defer rows.Close()

	fmt.Println("\nAll Tables in Database:")
	fmt.Println("=======================")

	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			log.Fatal("Failed to scan row:", err)
		}
		fmt.Println(tableName)
	}
}
