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
		log.Fatal(err)
	}
	defer db.Close()

	// Insert web channel
	_, err = db.Exec(`
		INSERT INTO channels (id, name, logo, configuration, enabled, created_at, updated_at)
		VALUES ('web', 'Web Form', NULL, '{}', 1, NOW(), NOW())
		ON DUPLICATE KEY UPDATE enabled = 1
	`)
	if err != nil {
		log.Printf("Channel insert warning: %v", err)
	} else {
		fmt.Println("✓ Web channel created/updated")
	}

	// Insert webhook
	_, err = db.Exec(`
		INSERT INTO webhooks (name, url, secret, enabled, event_all, event_ticket_created, event_ticket_updated, event_ticket_status_change, event_ticket_closed, event_message_created, created_at, updated_at)
		VALUES ('E2E Test Webhook', 'http://localhost:9000/webhook', 'test-secret-key-123', 1, 0, 1, 1, 1, 1, 1, NOW(), NOW())
		ON DUPLICATE KEY UPDATE enabled = 1, url = 'http://localhost:9000/webhook'
	`)
	if err != nil {
		log.Printf("Webhook insert warning: %v", err)
	} else {
		fmt.Println("✓ Webhook created/updated")
	}

	fmt.Println("✓ Test data setup complete!")
}
