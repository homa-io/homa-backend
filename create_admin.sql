-- Create admin user: admin@getevo.dev
-- This script creates an administrator user for the Homa backend system
-- Execute this on your production database where api.getevo.dev is running

INSERT INTO users (id, name, last_name, display_name, email, password_hash, type, created_at, updated_at)
VALUES ('eb7a14e7-c23e-4a8c-9ce6-1d7938ca7ef2', 'Admin', 'User', 'Admin User', 'admin@getevo.dev', '$s2$16384$8$1$qEZR6Rn9AOE1aKNrsMN7ZHfk$mc34wZPzmYZoYQt60k+PkABn/dHkG+3QidqgUnlQBKU=', 'administrator', '2025-11-14 17:59:23', '2025-11-14 17:59:23');

-- User Details:
-- User ID: eb7a14e7-c23e-4a8c-9ce6-1d7938ca7ef2
-- Email: admin@getevo.dev
-- Password: lightbear11
-- Type: administrator
-- Name: Admin User

-- After running this script, you can log in at:
-- POST https://api.getevo.dev/api/auth/login
-- Body: {"email": "admin@getevo.dev", "password": "lightbear11"}
