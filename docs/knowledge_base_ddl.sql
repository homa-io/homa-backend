-- =============================================
-- KNOWLEDGE BASE DATABASE SCHEMA
-- Generated: 2025-12-27
-- =============================================

-- =============================================
-- KNOWLEDGE BASE ARTICLES
-- Main table for storing KB articles
-- =============================================
CREATE TABLE `knowledge_base_articles` (
  `id` char(36) NOT NULL,                              -- UUID primary key
  `title` varchar(500) NOT NULL,                       -- Article title
  `slug` varchar(500) NOT NULL,                        -- URL-friendly slug (unique)
  `content` longtext NOT NULL,                         -- Full article content (HTML)
  `summary` text NOT NULL,                             -- Short summary/description for listings
  `featured_image` varchar(500) NOT NULL,              -- URL to featured/hero image
  `category_id` char(36) DEFAULT NULL,                 -- Foreign key to categories
  `author_id` char(36) DEFAULT NULL,                   -- Foreign key to users (author)
  `status` varchar(50) NOT NULL DEFAULT 'draft',       -- Article status: draft, published, archived
  `featured` tinyint(1) NOT NULL DEFAULT 0,            -- Whether article is featured/pinned
  `view_count` bigint(20) NOT NULL DEFAULT 0,          -- Number of times article was viewed
  `helpful_yes` bigint(20) NOT NULL DEFAULT 0,         -- Count of "helpful" votes
  `helpful_no` bigint(20) NOT NULL DEFAULT 0,          -- Count of "not helpful" votes
  `published_at` timestamp NULL DEFAULT NULL,          -- When article was first published
  `created_at` timestamp NOT NULL,                     -- Record creation timestamp
  `updated_at` timestamp NOT NULL,                     -- Record last update timestamp
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_knowledge_base_articles_slug` (`slug`),
  KEY `idx_knowledge_base_articles_title` (`title`),
  KEY `idx_knowledge_base_articles_category_id` (`category_id`),
  KEY `idx_knowledge_base_articles_author_id` (`author_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================
-- KNOWLEDGE BASE CATEGORIES
-- Hierarchical categories for organizing articles
-- =============================================
CREATE TABLE `knowledge_base_categories` (
  `id` char(36) NOT NULL,                              -- UUID primary key
  `name` varchar(255) NOT NULL,                        -- Category display name
  `slug` varchar(255) NOT NULL,                        -- URL-friendly slug (unique)
  `description` text NOT NULL,                         -- Category description
  `parent_id` char(36) DEFAULT NULL,                   -- Parent category ID (for hierarchy)
  `icon` varchar(100) NOT NULL,                        -- Category icon (emoji or icon class)
  `color` varchar(20) NOT NULL,                        -- Category color (hex code)
  `sort_order` bigint(20) NOT NULL DEFAULT 0,          -- Display order (lower = first)
  `created_at` timestamp NOT NULL,                     -- Record creation timestamp
  `updated_at` timestamp NOT NULL,                     -- Record last update timestamp
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_knowledge_base_categories_slug` (`slug`),
  KEY `idx_knowledge_base_categories_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================
-- KNOWLEDGE BASE TAGS
-- Tags for labeling and filtering articles
-- =============================================
CREATE TABLE `knowledge_base_tags` (
  `id` char(36) NOT NULL,                              -- UUID primary key
  `name` varchar(100) NOT NULL,                        -- Tag display name (unique)
  `slug` varchar(100) NOT NULL,                        -- URL-friendly slug (unique)
  `color` varchar(20) NOT NULL,                        -- Tag color (hex code)
  `created_at` timestamp NOT NULL,                     -- Record creation timestamp
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_knowledge_base_tags_name` (`name`),
  UNIQUE KEY `idx_knowledge_base_tags_slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================
-- KNOWLEDGE BASE ARTICLE-TAGS (Junction Table)
-- Many-to-Many relationship between articles and tags
-- =============================================
CREATE TABLE `knowledge_base_article_tags` (
  `article_id` char(36) NOT NULL,                      -- Foreign key to articles
  `tag_id` char(36) NOT NULL,                          -- Foreign key to tags
  PRIMARY KEY (`article_id`,`tag_id`)                  -- Composite primary key
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================
-- KNOWLEDGE BASE MEDIA
-- Media attachments (images, videos, documents) for articles
-- =============================================
CREATE TABLE `knowledge_base_media` (
  `id` char(36) NOT NULL,                              -- UUID primary key
  `article_id` char(36) NOT NULL,                      -- Foreign key to articles
  `type` varchar(50) NOT NULL,                         -- Media type: image, video, document
  `url` varchar(500) NOT NULL,                         -- URL/path to the media file
  `title` varchar(255) NOT NULL,                       -- Media title/alt text
  `description` text NOT NULL,                         -- Media description/caption
  `sort_order` bigint(20) NOT NULL DEFAULT 0,          -- Display order (lower = first)
  `is_primary` tinyint(1) NOT NULL DEFAULT 0,          -- Whether this is the primary media (one per article)
  `created_at` timestamp NOT NULL,                     -- Record creation timestamp
  PRIMARY KEY (`id`),
  KEY `idx_knowledge_base_media_article_id` (`article_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
