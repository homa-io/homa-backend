package models

import (
	"time"

	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/restify"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KnowledgeBaseArticle represents an article in the knowledge base
type KnowledgeBaseArticle struct {
	ID            uuid.UUID  `gorm:"column:id;type:char(36);primaryKey" json:"id"`
	Title         string     `gorm:"column:title;size:500;not null;index" json:"title"`
	Slug          string     `gorm:"column:slug;size:500;uniqueIndex;not null" json:"slug"`
	Content       string     `gorm:"column:content;type:longtext;not null" json:"content"`
	Summary       string     `gorm:"column:summary;type:text" json:"summary"`
	FeaturedImage string     `gorm:"column:featured_image;size:500" json:"featured_image"`
	CategoryID    *uuid.UUID `gorm:"column:category_id;type:char(36);index" json:"category_id"`
	AuthorID      *uuid.UUID `gorm:"column:author_id;type:char(36);index" json:"author_id"`
	Status        string     `gorm:"column:status;size:50;default:'draft';check:status IN ('draft','published','archived')" json:"status"`
	Featured      bool       `gorm:"column:featured;default:0" json:"featured"`
	ViewCount     int        `gorm:"column:view_count;default:0" json:"view_count"`
	HelpfulYes    int        `gorm:"column:helpful_yes;default:0" json:"helpful_yes"`
	HelpfulNo     int        `gorm:"column:helpful_no;default:0" json:"helpful_no"`
	PublishedAt   *time.Time `gorm:"column:published_at" json:"published_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Category *KnowledgeBaseCategory `gorm:"foreignKey:CategoryID;references:ID" json:"category,omitempty"`
	Tags     []KnowledgeBaseTag     `gorm:"many2many:knowledge_base_article_tags;foreignKey:ID;joinForeignKey:ArticleID;references:ID;joinReferences:TagID" json:"tags,omitempty"`
	Media    []KnowledgeBaseMedia   `gorm:"foreignKey:ArticleID;references:ID" json:"media,omitempty"`
	Chunks   []KnowledgeBaseChunk   `gorm:"foreignKey:ArticleID;references:ID" json:"-"`

	restify.API
}

// KnowledgeBaseCategory represents a category for organizing articles
type KnowledgeBaseCategory struct {
	ID          uuid.UUID  `gorm:"column:id;type:char(36);primaryKey" json:"id"`
	Name        string     `gorm:"column:name;size:255;not null" json:"name"`
	Slug        string     `gorm:"column:slug;size:255;uniqueIndex;not null" json:"slug"`
	Description string     `gorm:"column:description;type:text" json:"description"`
	ParentID    *uuid.UUID `gorm:"column:parent_id;type:char(36);index" json:"parent_id"`
	Icon        string     `gorm:"column:icon;size:100" json:"icon"`
	Color       string     `gorm:"column:color;size:20" json:"color"`
	SortOrder   int        `gorm:"column:sort_order;default:0" json:"sort_order"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Parent   *KnowledgeBaseCategory  `gorm:"foreignKey:ParentID;references:ID" json:"parent,omitempty"`
	Children []KnowledgeBaseCategory `gorm:"foreignKey:ParentID;references:ID" json:"children,omitempty"`
	Articles []KnowledgeBaseArticle  `gorm:"foreignKey:CategoryID;references:ID" json:"-"`

	restify.API
}

// KnowledgeBaseMedia represents media attachments for articles
type KnowledgeBaseMedia struct {
	ID          uuid.UUID `gorm:"column:id;type:char(36);primaryKey" json:"id"`
	ArticleID   uuid.UUID `gorm:"column:article_id;type:char(36);not null;index" json:"article_id"`
	Type        string    `gorm:"column:type;size:50;not null" json:"type"` // image, video, document
	URL         string    `gorm:"column:url;size:500;not null" json:"url"`
	Title       string    `gorm:"column:title;size:255" json:"title"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	SortOrder   int       `gorm:"column:sort_order;default:0" json:"sort_order"`
	IsPrimary   bool      `gorm:"column:is_primary;default:0" json:"is_primary"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relationships
	Article *KnowledgeBaseArticle `gorm:"foreignKey:ArticleID;references:ID" json:"-"`

	restify.API
}

// KnowledgeBaseTag represents a tag for articles
type KnowledgeBaseTag struct {
	ID        uuid.UUID `gorm:"column:id;type:char(36);primaryKey" json:"id"`
	Name      string    `gorm:"column:name;size:100;uniqueIndex;not null" json:"name"`
	Slug      string    `gorm:"column:slug;size:100;uniqueIndex;not null" json:"slug"`
	Color     string    `gorm:"column:color;size:20" json:"color"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relationships
	Articles []KnowledgeBaseArticle `gorm:"many2many:knowledge_base_article_tags;foreignKey:ID;joinForeignKey:TagID;references:ID;joinReferences:ArticleID" json:"-"`

	restify.API
}

// KnowledgeBaseArticleTag represents the many-to-many relationship between articles and tags
type KnowledgeBaseArticleTag struct {
	ArticleID uuid.UUID `gorm:"column:article_id;type:char(36);primaryKey" json:"article_id"`
	TagID     uuid.UUID `gorm:"column:tag_id;type:char(36);primaryKey" json:"tag_id"`

	restify.API
}

// KnowledgeBaseChunk represents a text chunk for RAG (Retrieval-Augmented Generation)
type KnowledgeBaseChunk struct {
	ID         uuid.UUID `gorm:"column:id;type:char(36);primaryKey" json:"id"`
	ArticleID  uuid.UUID `gorm:"column:article_id;type:char(36);not null;index" json:"article_id"`
	Content    string    `gorm:"column:content;type:text;not null" json:"content"`
	ChunkIndex int       `gorm:"column:chunk_index;not null" json:"chunk_index"`
	TokenCount int       `gorm:"column:token_count;default:0" json:"token_count"`
	Embedding  []byte    `gorm:"column:embedding;type:blob" json:"-"` // Store embedding as binary
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Article *KnowledgeBaseArticle `gorm:"foreignKey:ArticleID;references:ID" json:"article,omitempty"`

	restify.API
}

// TableName sets the table name for KnowledgeBaseArticle
func (KnowledgeBaseArticle) TableName() string {
	return "knowledge_base_articles"
}

// TableName sets the table name for KnowledgeBaseCategory
func (KnowledgeBaseCategory) TableName() string {
	return "knowledge_base_categories"
}

// TableName sets the table name for KnowledgeBaseTag
func (KnowledgeBaseTag) TableName() string {
	return "knowledge_base_tags"
}

// TableName sets the table name for KnowledgeBaseArticleTag
func (KnowledgeBaseArticleTag) TableName() string {
	return "knowledge_base_article_tags"
}

// TableName sets the table name for KnowledgeBaseChunk
func (KnowledgeBaseChunk) TableName() string {
	return "knowledge_base_chunks"
}

// TableName sets the table name for KnowledgeBaseMedia
func (KnowledgeBaseMedia) TableName() string {
	return "knowledge_base_media"
}

// KnowledgeBaseIndexer is an interface for indexing articles
// This allows the models package to trigger indexing without importing the ai package
type KnowledgeBaseIndexer interface {
	IndexArticle(articleID uuid.UUID) error
	DeleteArticleIndex(articleID uuid.UUID) error
}

// Global indexer - set by the ai package during initialization
var knowledgeBaseIndexer KnowledgeBaseIndexer

// SetKnowledgeBaseIndexer sets the indexer implementation
func SetKnowledgeBaseIndexer(indexer KnowledgeBaseIndexer) {
	knowledgeBaseIndexer = indexer
}

// BeforeCreate hook - generate UUID if not set
func (a *KnowledgeBaseArticle) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// AfterCreate hook - index the article in Qdrant
func (a *KnowledgeBaseArticle) AfterCreate(tx *gorm.DB) error {
	if knowledgeBaseIndexer != nil && a.Status == "published" {
		go func() {
			if err := knowledgeBaseIndexer.IndexArticle(a.ID); err != nil {
				// Log error but don't fail the transaction
				log.Error("Failed to index article %s: %v", a.ID, err)
			}
		}()
	}
	return nil
}

// AfterUpdate hook - re-index the article in Qdrant
func (a *KnowledgeBaseArticle) AfterUpdate(tx *gorm.DB) error {
	if knowledgeBaseIndexer != nil {
		go func() {
			if a.Status == "published" {
				// Re-index published articles
				if err := knowledgeBaseIndexer.IndexArticle(a.ID); err != nil {
					log.Error("Failed to re-index article %s: %v", a.ID, err)
				}
			} else {
				// Remove from index if not published
				if err := knowledgeBaseIndexer.DeleteArticleIndex(a.ID); err != nil {
					log.Error("Failed to delete article index %s: %v", a.ID, err)
				}
			}
		}()
	}
	return nil
}

// AfterDelete hook - remove the article from Qdrant
func (a *KnowledgeBaseArticle) AfterDelete(tx *gorm.DB) error {
	if knowledgeBaseIndexer != nil {
		go func() {
			if err := knowledgeBaseIndexer.DeleteArticleIndex(a.ID); err != nil {
				log.Error("Failed to delete article index %s: %v", a.ID, err)
			}
		}()
	}
	return nil
}

// BeforeCreate hook for Category - generate UUID if not set
func (c *KnowledgeBaseCategory) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for Tag - generate UUID if not set
func (t *KnowledgeBaseTag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for Chunk - generate UUID if not set
func (c *KnowledgeBaseChunk) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for Media - generate UUID if not set
func (m *KnowledgeBaseMedia) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
