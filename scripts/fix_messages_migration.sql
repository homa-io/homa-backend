-- Migration script to clean up messages table after ticket->conversation rename
-- This script removes old ticket_id column and its constraints

USE homa;

-- Step 1: Find and drop the foreign key constraint on ticket_id
SET @constraint_name = (
    SELECT CONSTRAINT_NAME
    FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
    WHERE TABLE_SCHEMA = 'homa'
    AND TABLE_NAME = 'messages'
    AND COLUMN_NAME = 'ticket_id'
    AND REFERENCED_TABLE_NAME IS NOT NULL
    LIMIT 1
);

SET @sql = IF(@constraint_name IS NOT NULL,
    CONCAT('ALTER TABLE messages DROP FOREIGN KEY ', @constraint_name),
    'SELECT "No foreign key constraint found on ticket_id" AS info'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 2: Drop the index on ticket_id (if it exists)
SET @index_exists = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = 'homa'
    AND TABLE_NAME = 'messages'
    AND INDEX_NAME = 'idx_messages_ticket_id'
);

SET @sql = IF(@index_exists > 0,
    'ALTER TABLE messages DROP INDEX idx_messages_ticket_id',
    'SELECT "Index idx_messages_ticket_id does not exist" AS info'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 3: Drop the ticket_id column (if it exists)
SET @column_exists = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = 'homa'
    AND TABLE_NAME = 'messages'
    AND COLUMN_NAME = 'ticket_id'
);

SET @sql = IF(@column_exists > 0,
    'ALTER TABLE messages DROP COLUMN ticket_id',
    'SELECT "Column ticket_id does not exist" AS info'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 4: Verify the final structure
SHOW CREATE TABLE messages;

SELECT "Migration completed successfully!" AS status;
