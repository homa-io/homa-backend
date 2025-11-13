-- Mock data for testing Homa agent APIs
USE homa;

-- Insert channels
INSERT INTO channels (id, name, type, configuration, created_at, updated_at) VALUES
('web', 'Web Portal', 'web', '{}', NOW(), NOW()),
('email', 'Email Support', 'email', '{}', NOW(), NOW()),
('whatsapp', 'WhatsApp', 'whatsapp', '{}', NOW(), NOW());

-- Insert departments
INSERT INTO departments (id, name, description, created_at, updated_at) VALUES
(1, 'Technical Support', 'Handle technical issues and bugs', NOW(), NOW()),
(2, 'Sales', 'Handle sales inquiries and quotes', NOW(), NOW()),
(3, 'Billing', 'Handle payment and billing issues', NOW(), NOW());

-- Insert tags
INSERT INTO tags (id, name) VALUES
(1, 'urgent'),
(2, 'bug'),
(3, 'feature-request'),
(4, 'billing-issue'),
(5, 'resolved');

-- Insert users (agents and administrators)
INSERT INTO users (id, name, last_name, display_name, email, password_hash, type, created_at, updated_at) VALUES
('11111111-1111-1111-1111-111111111111', 'John', 'Doe', 'John Doe', 'admin@nexa.com', '$2a$14$7BDZskTQX9JGCOq8ZDXKdu4O5hAu2/7D5QI9L1DjG6MqgzF1jQZ7K', 'administrator', NOW(), NOW()),
('22222222-2222-2222-2222-222222222222', 'Jane', 'Smith', 'Jane Smith', 'agent1@nexa.com', '$2a$14$7BDZskTQX9JGCOq8ZDXKdu4O5hAu2/7D5QI9L1DjG6MqgzF1jQZ7K', 'agent', NOW(), NOW()),
('33333333-3333-3333-3333-333333333333', 'Bob', 'Johnson', 'Bob Johnson', 'agent2@nexa.com', '$2a$14$7BDZskTQX9JGCOq8ZDXKdu4O5hAu2/7D5QI9L1DjG6MqgzF1jQZ7K', 'agent', NOW(), NOW());

-- Assign agents to departments
INSERT INTO user_departments (user_id, department_id) VALUES
('22222222-2222-2222-2222-222222222222', 1),  -- Jane in Technical Support
('22222222-2222-2222-2222-222222222222', 2),  -- Jane in Sales
('33333333-3333-3333-3333-333333333333', 3);  -- Bob in Billing

-- Insert clients
INSERT INTO clients (id, data, created_at, updated_at) VALUES
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '{"name": "Client One", "email": "client1@example.com", "type": "email"}', NOW(), NOW()),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '{"name": "Client Two", "email": "client2@example.com", "type": "email"}', NOW(), NOW()),
('cccccccc-cccc-cccc-cccc-cccccccccccc', '{"name": "Client Three", "phone": "+1234567890", "type": "phone"}', NOW(), NOW());

-- Insert conversations
INSERT INTO conversations (id, title, client_id, department_id, channel_id, secret, status, priority, custom_fields, created_at, updated_at) VALUES
(1, 'Login Issue - Cannot access account', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 1, 'web', 'a1b2c3d4e5f6789012345678901234ef', 'new', 'high', '{}', NOW(), NOW()),
(2, 'Billing inquiry about monthly charges', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 3, 'email', 'b2c3d4e5f67890123456789012345678', 'wait_for_agent', 'medium', '{}', NOW(), NOW()),
(3, 'Feature request for mobile app', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 1, 'whatsapp', 'c3d4e5f678901234567890123456789a', 'in_progress', 'low', '{}', NOW(), NOW()),
(4, 'Password reset not working', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 1, 'web', 'd4e5f6789012345678901234567890ab', 'resolved', 'medium', '{}', NOW(), NOW()),
(5, 'Account suspension appeal', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 1, 'email', 'e5f6789012345678901234567890abcd', 'closed', 'urgent', '{}', NOW(), NOW());

-- Assign conversations to agents
INSERT INTO conversation_assignments (conversation_id, user_id, department_id) VALUES
(1, '22222222-2222-2222-2222-222222222222', 1),  -- Jane assigned to conversation 1
(3, '22222222-2222-2222-2222-222222222222', 1),  -- Jane assigned to conversation 3
(2, '33333333-3333-3333-3333-333333333333', 3);  -- Bob assigned to conversation 2

-- Add some tags to conversations
INSERT INTO conversation_tags (conversation_id, tag_id) VALUES
(1, 2),  -- Login issue tagged as 'bug'
(1, 1),  -- Login issue tagged as 'urgent'
(2, 4),  -- Billing inquiry tagged as 'billing-issue'
(3, 3),  -- Feature request tagged as 'feature-request'
(4, 5),  -- Password reset tagged as 'resolved'
(5, 1);  -- Account suspension tagged as 'urgent'

-- Insert some messages
INSERT INTO messages (id, conversation_id, user_id, client_id, body, is_system_message, created_at) VALUES
(1, 1, NULL, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'I cannot log into my account. Getting error message "Invalid credentials".', 0, NOW()),
(2, 1, '22222222-2222-2222-2222-222222222222', NULL, 'Hello! I see you are having trouble logging in. Can you please try resetting your password first?', 0, NOW()),
(3, 2, NULL, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'I need clarification on my monthly charges. There are some items I do not recognize.', 0, NOW()),
(4, 3, NULL, 'cccccccc-cccc-cccc-cccc-cccccccccccc', 'Would it be possible to add dark mode to the mobile application?', 0, NOW()),
(5, 3, '22222222-2222-2222-2222-222222222222', NULL, 'Thank you for the suggestion! I will forward this to our development team for consideration.', 0, NOW());