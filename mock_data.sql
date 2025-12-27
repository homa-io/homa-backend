-- Mock data for Homa backend
-- This script creates sample data for testing

-- Insert Departments
INSERT IGNORE INTO departments (name, description, created_at) VALUES
('Technical Support', 'Handles technical issues and troubleshooting', NOW()),
('Sales', 'Handles sales inquiries and product information', NOW()),
('Billing', 'Handles payment and billing related questions', NOW()),
('Customer Success', 'Handles customer onboarding and success', NOW());

-- Insert Channels (if not exist)
INSERT IGNORE INTO channels (id, name, logo, configuration, enabled, created_at, updated_at) VALUES
('web', 'Web Form', NULL, '{}', 1, NOW(), NOW()),
('whatsapp', 'WhatsApp', NULL, '{}', 1, NOW(), NOW()),
('telegram', 'Telegram', NULL, '{}', 1, NOW(), NOW()),
('slack', 'Slack', NULL, '{}', 1, NOW(), NOW()),
('email', 'Email', NULL, '{}', 1, NOW(), NOW());

-- Insert Tags
INSERT IGNORE INTO tags (name) VALUES
('urgent'),
('bug'),
('feature-request'),
('billing-issue'),
('vip'),
('follow-up');

-- Insert Clients
INSERT INTO clients (id, name, data, language, timezone, created_at, updated_at) VALUES
(UUID(), 'John Smith', '{"email": "john.smith@example.com", "phone": "+1-555-0101", "company": "Tech Corp"}', 'en', 'America/New_York', NOW(), NOW()),
(UUID(), 'Maria Garcia', '{"email": "maria.garcia@example.com", "phone": "+34-600-123456", "company": "Innovate SL"}', 'es', 'Europe/Madrid', NOW(), NOW()),
(UUID(), 'Chen Wei', '{"email": "chen.wei@example.com", "phone": "+86-138-0000-0000", "company": "Dragon Tech"}', 'zh', 'Asia/Shanghai', NOW(), NOW()),
(UUID(), 'Sophie Martin', '{"email": "sophie.martin@example.com", "phone": "+33-6-12-34-56-78", "company": "Paris Digital"}', 'fr', 'Europe/Paris', NOW(), NOW()),
(UUID(), 'Ahmed Hassan', '{"email": "ahmed.hassan@example.com", "phone": "+20-100-000-0000", "company": "Cairo Solutions"}', 'ar', 'Africa/Cairo', NOW(), NOW());

-- Get client IDs for conversations
SET @client1 = (SELECT id FROM clients WHERE name = 'John Smith' ORDER BY created_at DESC LIMIT 1);
SET @client2 = (SELECT id FROM clients WHERE name = 'Maria Garcia' ORDER BY created_at DESC LIMIT 1);
SET @client3 = (SELECT id FROM clients WHERE name = 'Chen Wei' ORDER BY created_at DESC LIMIT 1);
SET @client4 = (SELECT id FROM clients WHERE name = 'Sophie Martin' ORDER BY created_at DESC LIMIT 1);
SET @client5 = (SELECT id FROM clients WHERE name = 'Ahmed Hassan' ORDER BY created_at DESC LIMIT 1);

-- Get department IDs
SET @dept_tech = (SELECT id FROM departments WHERE name = 'Technical Support');
SET @dept_sales = (SELECT id FROM departments WHERE name = 'Sales');
SET @dept_billing = (SELECT id FROM departments WHERE name = 'Billing');
SET @dept_success = (SELECT id FROM departments WHERE name = 'Customer Success');

-- Get user ID for admin
SET @admin_user = (SELECT id FROM users WHERE email = 'admin@getevo.dev' LIMIT 1);

-- Insert Conversations
INSERT INTO conversations (title, client_id, department_id, channel_id, external_id, secret, status, priority, custom_fields, ip, browser, operating_system, created_at, updated_at, closed_at) VALUES
('Unable to login to dashboard', @client1, @dept_tech, 'web', NULL, MD5(CONCAT('secret', RAND())), 'open', 'high', '{}', '192.168.1.100', 'Chrome 120.0', 'Windows 11', DATE_SUB(NOW(), INTERVAL 2 HOUR), NOW(), NULL),
('Pricing inquiry for Enterprise plan', @client2, @dept_sales, 'email', 'msg_1234567890', MD5(CONCAT('secret', RAND())), 'assigned', 'medium', '{}', '85.123.45.67', 'Firefox 121.0', 'macOS 14.0', DATE_SUB(NOW(), INTERVAL 5 HOUR), NOW(), NULL),
('Payment failed - Need assistance', @client3, @dept_billing, 'whatsapp', 'wa_9876543210', MD5(CONCAT('secret', RAND())), 'open', 'urgent', '{"payment_amount": 299.99, "currency": "USD"}', '114.245.67.89', 'Safari 17.0', 'iOS 17.2', DATE_SUB(NOW(), INTERVAL 1 HOUR), NOW(), NULL),
('Feature request: Dark mode', @client4, @dept_tech, 'telegram', 'tg_11223344', MD5(CONCAT('secret', RAND())), 'pending', 'low', '{}', '82.45.123.78', 'Edge 120.0', 'Windows 10', DATE_SUB(NOW(), INTERVAL 1 DAY), DATE_SUB(NOW(), INTERVAL 20 HOUR), NULL),
('Onboarding questions', @client5, @dept_success, 'slack', 'C1234567890', MD5(CONCAT('secret', RAND())), 'assigned', 'medium', '{}', '41.234.56.78', 'Chrome 120.0', 'Ubuntu 22.04', DATE_SUB(NOW(), INTERVAL 3 HOUR), NOW(), NULL),
('Account deletion request', @client1, @dept_success, 'web', NULL, MD5(CONCAT('secret', RAND())), 'closed', 'high', '{}', '192.168.1.100', 'Chrome 120.0', 'Windows 11', DATE_SUB(NOW(), INTERVAL 2 DAY), DATE_SUB(NOW(), INTERVAL 1 DAY), DATE_SUB(NOW(), INTERVAL 1 DAY)),
('Integration setup help', @client3, @dept_tech, 'email', 'msg_tech_001', MD5(CONCAT('secret', RAND())), 'open', 'medium', '{"integration_type": "API", "version": "v2"}', '114.245.67.89', 'Chrome 119.0', 'macOS 13.0', DATE_SUB(NOW(), INTERVAL 4 HOUR), NOW(), NULL);

-- Get conversation IDs
SET @conv1 = (SELECT id FROM conversations WHERE title = 'Unable to login to dashboard' ORDER BY id DESC LIMIT 1);
SET @conv2 = (SELECT id FROM conversations WHERE title = 'Pricing inquiry for Enterprise plan' ORDER BY id DESC LIMIT 1);
SET @conv3 = (SELECT id FROM conversations WHERE title = 'Payment failed - Need assistance' ORDER BY id DESC LIMIT 1);
SET @conv4 = (SELECT id FROM conversations WHERE title = 'Feature request: Dark mode' ORDER BY id DESC LIMIT 1);
SET @conv5 = (SELECT id FROM conversations WHERE title = 'Onboarding questions' ORDER BY id DESC LIMIT 1);
SET @conv6 = (SELECT id FROM conversations WHERE title = 'Account deletion request' ORDER BY id DESC LIMIT 1);
SET @conv7 = (SELECT id FROM conversations WHERE title = 'Integration setup help' ORDER BY id DESC LIMIT 1);

-- Insert Messages for conversations
INSERT INTO messages (conversation_id, user_id, client_id, body, is_system_message, created_at) VALUES
-- Conversation 1 messages (Unable to login)
(@conv1, NULL, @client1, 'Hi, I am unable to login to my dashboard. It keeps showing "Invalid credentials" even though I am sure my password is correct.', 0, DATE_SUB(NOW(), INTERVAL 2 HOUR)),
(@conv1, @admin_user, NULL, 'Hello! I will help you with this. Can you please confirm the email address you are using to login?', 0, DATE_SUB(NOW(), INTERVAL 115 MINUTE)),
(@conv1, NULL, @client1, 'I am using john.smith@example.com', 0, DATE_SUB(NOW(), INTERVAL 110 MINUTE)),
(@conv1, @admin_user, NULL, 'Thank you. I can see your account. Let me send you a password reset link to that email address.', 0, DATE_SUB(NOW(), INTERVAL 105 MINUTE)),
(@conv1, NULL, @client1, 'Got it! Just reset my password and now I can login. Thank you so much!', 0, DATE_SUB(NOW(), INTERVAL 90 MINUTE)),

-- Conversation 2 messages (Pricing inquiry)
(@conv2, NULL, @client2, 'Hello, I am interested in your Enterprise plan. Can you provide more details about pricing and features?', 0, DATE_SUB(NOW(), INTERVAL 5 HOUR)),
(@conv2, @admin_user, NULL, 'Hi Maria! Thank you for your interest. Our Enterprise plan starts at $999/month and includes unlimited users, priority support, and custom integrations. Would you like to schedule a demo?', 0, DATE_SUB(NOW(), INTERVAL 285 MINUTE)),
(@conv2, NULL, @client2, 'Yes, a demo would be great. We have a team of about 50 people. Do you offer volume discounts?', 0, DATE_SUB(NOW(), INTERVAL 270 MINUTE)),

-- Conversation 3 messages (Payment failed)
(@conv3, NULL, @client3, 'My payment just failed. I need to renew my subscription urgently as my team is working on a project.', 0, DATE_SUB(NOW(), INTERVAL 1 HOUR)),
(@conv3, @admin_user, NULL, 'I understand the urgency. Let me check your payment details. It appears your card was declined. Could you try using a different payment method?', 0, DATE_SUB(NOW(), INTERVAL 55 MINUTE)),
(@conv3, NULL, @client3, 'Let me contact my bank first. They might have blocked it for security reasons.', 0, DATE_SUB(NOW(), INTERVAL 50 MINUTE)),

-- Conversation 4 messages (Feature request)
(@conv4, NULL, @client4, 'Would love to see a dark mode option for the dashboard. It would be much easier on the eyes during late-night work sessions.', 0, DATE_SUB(NOW(), INTERVAL 1 DAY)),

-- Conversation 5 messages (Onboarding)
(@conv5, NULL, @client5, 'I just signed up and have some questions about getting started. Where do I begin?', 0, DATE_SUB(NOW(), INTERVAL 3 HOUR)),
(@conv5, @admin_user, NULL, 'Welcome aboard! I recommend starting with our Quick Start Guide. I will send you a link. Do you have any specific features you want to set up first?', 0, DATE_SUB(NOW(), INTERVAL 175 MINUTE)),

-- Conversation 6 messages (Account deletion - closed)
(@conv6, NULL, @client1, 'I would like to delete my account and all associated data.', 0, DATE_SUB(NOW(), INTERVAL 2 DAY)),
(@conv6, @admin_user, NULL, 'I am sorry to see you go. Before we proceed, may I ask if there is anything we could do to improve your experience?', 0, DATE_SUB(NOW(), INTERVAL 47 HOUR)),
(@conv6, NULL, @client1, 'No, I found another solution that better fits my needs. Please proceed with the deletion.', 0, DATE_SUB(NOW(), INTERVAL 46 HOUR)),
(@conv6, @admin_user, NULL, 'Understood. Your account has been scheduled for deletion. All data will be removed within 30 days. Thank you for trying our service.', 0, DATE_SUB(NOW(), INTERVAL 1 DAY)),

-- Conversation 7 messages (Integration setup)
(@conv7, NULL, @client3, 'I need help setting up the API integration. The documentation is a bit unclear on the authentication part.', 0, DATE_SUB(NOW(), INTERVAL 4 HOUR)),
(@conv7, @admin_user, NULL, 'I can help with that. Are you using OAuth 2.0 or API keys for authentication?', 0, DATE_SUB(NOW(), INTERVAL 230 MINUTE)),
(@conv7, NULL, @client3, 'We want to use API keys. How do I generate them?', 0, DATE_SUB(NOW(), INTERVAL 225 MINUTE));

-- Get tag IDs
SET @tag_urgent = (SELECT id FROM tags WHERE name = 'urgent');
SET @tag_bug = (SELECT id FROM tags WHERE name = 'bug');
SET @tag_feature = (SELECT id FROM tags WHERE name = 'feature-request');
SET @tag_billing = (SELECT id FROM tags WHERE name = 'billing-issue');
SET @tag_vip = (SELECT id FROM tags WHERE name = 'vip');
SET @tag_followup = (SELECT id FROM tags WHERE name = 'follow-up');

-- Insert Conversation-Tag relationships
INSERT INTO conversation_tags (conversation_id, tag_id) VALUES
(@conv1, @tag_bug),
(@conv2, @tag_vip),
(@conv3, @tag_urgent),
(@conv3, @tag_billing),
(@conv4, @tag_feature),
(@conv5, @tag_followup),
(@conv6, @tag_vip),
(@conv7, @tag_followup);

-- Insert Conversation Assignments
INSERT INTO conversation_assignments (conversation_id, user_id, department_id) VALUES
(@conv2, @admin_user, NULL),
(@conv5, @admin_user, NULL);

-- Update user-department relationships if needed
INSERT IGNORE INTO user_departments (user_id, department_id) VALUES
(@admin_user, @dept_tech),
(@admin_user, @dept_sales),
(@admin_user, @dept_billing),
(@admin_user, @dept_success);

SELECT 'Mock data created successfully!' AS Status;
SELECT
    'Departments' AS Entity, COUNT(*) AS Count FROM departments
UNION ALL
SELECT 'Channels', COUNT(*) FROM channels
UNION ALL
SELECT 'Tags', COUNT(*) FROM tags
UNION ALL
SELECT 'Clients', COUNT(*) FROM clients
UNION ALL
SELECT 'Conversations (new)', COUNT(*) FROM conversations WHERE id > 1
UNION ALL
SELECT 'Messages', COUNT(*) FROM messages;
