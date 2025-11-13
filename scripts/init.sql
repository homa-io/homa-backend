-- Homa Database Initialization Script
-- This script is run when the MySQL container starts

-- Ensure the database exists
CREATE DATABASE IF NOT EXISTS homa;
USE homa;

-- Set proper charset and collation
ALTER DATABASE homa CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Create settings table if it doesn't exist
-- This prevents the warning about missing settings table
CREATE TABLE IF NOT EXISTS `settings` (
  `key` varchar(255) NOT NULL,
  `value` text,
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default settings
INSERT IGNORE INTO `settings` (`key`, `value`) VALUES
('app.version', '1.0.0'),
('app.initialized', 'true'),
('app.installation_date', NOW());

-- Grant all privileges to the homa user
GRANT ALL PRIVILEGES ON homa.* TO 'homa'@'%';
FLUSH PRIVILEGES;