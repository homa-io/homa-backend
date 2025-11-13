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

	// Query recent messages
	query := `SELECT id, conversation_id, user_id, client_id, SUBSTRING(body, 1, 50) as body, created_at FROM messages ORDER BY id DESC LIMIT 10`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Failed to query messages:", err)
	}
	defer rows.Close()

	fmt.Println("\nRecent Messages:")
	fmt.Println("ID | Conv_ID | User_ID | Client_ID | Body | Created_At")
	fmt.Println("---+--------+---------+-----------+------+-----------")

	for rows.Next() {
		var id, conversationID int
		var userID, clientID, body sql.NullString
		var createdAt string

		err := rows.Scan(&id, &conversationID, &userID, &clientID, &body, &createdAt)
		if err != nil {
			log.Fatal("Failed to scan row:", err)
		}

		userIDStr := "NULL"
		if userID.Valid {
			userIDStr = userID.String
		}

		clientIDStr := "NULL"
		if clientID.Valid {
			clientIDStr = clientID.String
		}

		bodyStr := "NULL"
		if body.Valid {
			bodyStr = body.String
		}

		fmt.Printf("%d | %d | %s | %s | %s | %s\n",
			id, conversationID, userIDStr, clientIDStr, bodyStr, createdAt)
	}
}
