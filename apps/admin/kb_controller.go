package admin

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/pagination"
	"github.com/google/uuid"
	"github.com/iesreza/homa-backend/apps/auth"
	"github.com/iesreza/homa-backend/apps/models"
	"github.com/iesreza/homa-backend/lib/response"
	"gorm.io/gorm"
)

// ========================
// KNOWLEDGE BASE ARTICLE APIs
// ========================

// ListKBArticles returns paginated list of knowledge base articles
func (c Controller) ListKBArticles(request *evo.Request) any {
	var articles []models.KnowledgeBaseArticle
	query := db.Model(&models.KnowledgeBaseArticle{}).
		Preload("Category").
		Preload("Tags").
		Preload("Media")

	// Search functionality
	if search := request.Query("search").String(); search != "" {
		query = query.Where(
			"title LIKE ? OR summary LIKE ? OR content LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%",
		)
	}

	// Filter by status
	if status := request.Query("status").String(); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by category
	if categoryID := request.Query("category_id").String(); categoryID != "" {
		if catUUID, err := uuid.Parse(categoryID); err == nil {
			query = query.Where("category_id = ?", catUUID)
		}
	}

	// Filter by featured
	if featured := request.Query("featured").String(); featured == "true" {
		query = query.Where("featured = ?", true)
	}

	// Order by updated_at desc by default
	query = query.Order("updated_at DESC")

	p, err := pagination.New(query, request, &articles, pagination.Options{MaxSize: 100})
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OKWithMeta(articles, &response.Meta{
		Page:       p.CurrentPage,
		Limit:      p.Size,
		Total:      int64(p.Records),
		TotalPages: p.Pages,
	})
}

// GetKBArticle returns a single knowledge base article by ID
func (c Controller) GetKBArticle(request *evo.Request) any {
	id := request.Param("id").String()
	articleID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var article models.KnowledgeBaseArticle
	err = db.Model(&models.KnowledgeBaseArticle{}).
		Preload("Category").
		Preload("Tags").
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("id = ?", articleID).
		First(&article).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(article)
}

// CreateKBArticle creates a new knowledge base article
func (c Controller) CreateKBArticle(request *evo.Request) any {
	user := request.User().(*auth.User)

	var req struct {
		Title         string   `json:"title" validate:"required"`
		Content       string   `json:"content" validate:"required"`
		Summary       string   `json:"summary"`
		FeaturedImage string   `json:"featured_image"`
		CategoryID    string   `json:"category_id"`
		Status        string   `json:"status"`
		Featured      bool     `json:"featured"`
		TagIDs        []string `json:"tag_ids"`
		Media         []struct {
			Type        string `json:"type"`
			URL         string `json:"url"`
			Title       string `json:"title"`
			Description string `json:"description"`
			SortOrder   int    `json:"sort_order"`
			IsPrimary   bool   `json:"is_primary"`
		} `json:"media"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Generate slug from title
	slug := generateSlug(req.Title)

	// Ensure slug is unique
	var count int64
	db.Model(&models.KnowledgeBaseArticle{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		slug = slug + "-" + time.Now().Format("20060102150405")
	}

	article := models.KnowledgeBaseArticle{
		Title:         req.Title,
		Slug:          slug,
		Content:       req.Content,
		Summary:       req.Summary,
		FeaturedImage: req.FeaturedImage,
		AuthorID:      &user.UserID,
		Status:        "draft",
		Featured:      req.Featured,
	}

	if req.Status != "" {
		article.Status = req.Status
	}

	if req.Status == "published" {
		now := time.Now()
		article.PublishedAt = &now
	}

	// Parse category ID
	if req.CategoryID != "" {
		if catUUID, err := uuid.Parse(req.CategoryID); err == nil {
			article.CategoryID = &catUUID
		}
	}

	// Create article
	if err := db.Create(&article).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Handle tags
	if len(req.TagIDs) > 0 {
		for _, tagIDStr := range req.TagIDs {
			if tagID, err := uuid.Parse(tagIDStr); err == nil {
				articleTag := models.KnowledgeBaseArticleTag{
					ArticleID: article.ID,
					TagID:     tagID,
				}
				db.Create(&articleTag)
			}
		}
	}

	// Handle media (ensure only one is primary)
	if len(req.Media) > 0 {
		hasPrimary := false
		for _, m := range req.Media {
			isPrimary := m.IsPrimary && !hasPrimary // Only first primary wins
			if isPrimary {
				hasPrimary = true
			}
			media := models.KnowledgeBaseMedia{
				ArticleID:   article.ID,
				Type:        m.Type,
				URL:         m.URL,
				Title:       m.Title,
				Description: m.Description,
				SortOrder:   m.SortOrder,
				IsPrimary:   isPrimary,
			}
			db.Create(&media)
		}
	}

	// Reload article with relationships
	db.Model(&models.KnowledgeBaseArticle{}).
		Preload("Category").
		Preload("Tags").
		Preload("Media").
		Where("id = ?", article.ID).
		First(&article)

	return response.Created(article)
}

// UpdateKBArticle updates an existing knowledge base article
func (c Controller) UpdateKBArticle(request *evo.Request) any {
	id := request.Param("id").String()
	articleID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var article models.KnowledgeBaseArticle
	if err := db.Where("id = ?", articleID).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	var req struct {
		Title         string   `json:"title"`
		Content       string   `json:"content"`
		Summary       string   `json:"summary"`
		FeaturedImage *string  `json:"featured_image"`
		CategoryID    *string  `json:"category_id"`
		Status        string   `json:"status"`
		Featured      *bool    `json:"featured"`
		TagIDs        []string `json:"tag_ids"`
		Media         []struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			URL         string `json:"url"`
			Title       string `json:"title"`
			Description string `json:"description"`
			SortOrder   int    `json:"sort_order"`
			IsPrimary   bool   `json:"is_primary"`
		} `json:"media"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Update fields
	if req.Title != "" {
		article.Title = req.Title
		// Update slug if title changed
		newSlug := generateSlug(req.Title)
		if newSlug != article.Slug {
			var count int64
			db.Model(&models.KnowledgeBaseArticle{}).Where("slug = ? AND id != ?", newSlug, article.ID).Count(&count)
			if count > 0 {
				newSlug = newSlug + "-" + time.Now().Format("20060102150405")
			}
			article.Slug = newSlug
		}
	}

	if req.Content != "" {
		article.Content = req.Content
	}

	if req.Summary != "" {
		article.Summary = req.Summary
	}

	// Only update FeaturedImage if explicitly provided in request
	if req.FeaturedImage != nil {
		article.FeaturedImage = *req.FeaturedImage
	}

	if req.CategoryID != nil {
		if *req.CategoryID == "" {
			article.CategoryID = nil
		} else if catUUID, err := uuid.Parse(*req.CategoryID); err == nil {
			article.CategoryID = &catUUID
		}
	}

	if req.Status != "" {
		// Set published_at when status changes to published
		if req.Status == "published" && article.Status != "published" {
			now := time.Now()
			article.PublishedAt = &now
		}
		article.Status = req.Status
	}

	if req.Featured != nil {
		article.Featured = *req.Featured
	}

	// Save article
	if err := db.Save(&article).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Update tags if provided
	if req.TagIDs != nil {
		// Delete existing tags
		db.Where("article_id = ?", article.ID).Delete(&models.KnowledgeBaseArticleTag{})

		// Add new tags
		for _, tagIDStr := range req.TagIDs {
			if tagID, err := uuid.Parse(tagIDStr); err == nil {
				articleTag := models.KnowledgeBaseArticleTag{
					ArticleID: article.ID,
					TagID:     tagID,
				}
				db.Create(&articleTag)
			}
		}
	}

	// Update media if provided (ensure only one is primary)
	if req.Media != nil {
		// Delete existing media
		db.Where("article_id = ?", article.ID).Delete(&models.KnowledgeBaseMedia{})

		// Add new media
		hasPrimary := false
		for _, m := range req.Media {
			isPrimary := m.IsPrimary && !hasPrimary // Only first primary wins
			if isPrimary {
				hasPrimary = true
			}
			media := models.KnowledgeBaseMedia{
				ArticleID:   article.ID,
				Type:        m.Type,
				URL:         m.URL,
				Title:       m.Title,
				Description: m.Description,
				SortOrder:   m.SortOrder,
				IsPrimary:   isPrimary,
			}
			db.Create(&media)
		}
	}

	// Reload article with relationships
	db.Model(&models.KnowledgeBaseArticle{}).
		Preload("Category").
		Preload("Tags").
		Preload("Media").
		Where("id = ?", article.ID).
		First(&article)

	return response.OK(article)
}

// DeleteKBArticle deletes a knowledge base article
func (c Controller) DeleteKBArticle(request *evo.Request) any {
	id := request.Param("id").String()
	articleID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var article models.KnowledgeBaseArticle
	if err := db.Where("id = ?", articleID).First(&article).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Delete related records
	db.Where("article_id = ?", articleID).Delete(&models.KnowledgeBaseArticleTag{})
	db.Where("article_id = ?", articleID).Delete(&models.KnowledgeBaseMedia{})
	db.Where("article_id = ?", articleID).Delete(&models.KnowledgeBaseChunk{})

	// Delete article
	if err := db.Delete(&article).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Article deleted successfully"})
}

// ========================
// KNOWLEDGE BASE CATEGORY APIs
// ========================

// ListKBCategories returns all knowledge base categories
func (c Controller) ListKBCategories(request *evo.Request) any {
	var categories []models.KnowledgeBaseCategory
	query := db.Model(&models.KnowledgeBaseCategory{}).
		Preload("Parent").
		Preload("Children").
		Order("sort_order ASC, name ASC")

	if err := query.Find(&categories).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Calculate article counts
	type ArticleCount struct {
		CategoryID uuid.UUID
		Count      int
	}
	var counts []ArticleCount
	db.Model(&models.KnowledgeBaseArticle{}).
		Select("category_id, COUNT(*) as count").
		Where("category_id IS NOT NULL").
		Group("category_id").
		Scan(&counts)

	countMap := make(map[uuid.UUID]int)
	for _, c := range counts {
		countMap[c.CategoryID] = c.Count
	}

	// Create response with article counts
	type CategoryWithCount struct {
		models.KnowledgeBaseCategory
		ArticleCount int `json:"article_count"`
	}

	result := make([]CategoryWithCount, len(categories))
	for i, cat := range categories {
		result[i] = CategoryWithCount{
			KnowledgeBaseCategory: cat,
			ArticleCount:          countMap[cat.ID],
		}
	}

	return response.OK(result)
}

// GetKBCategory returns a single category by ID
func (c Controller) GetKBCategory(request *evo.Request) any {
	id := request.Param("id").String()
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var category models.KnowledgeBaseCategory
	if err := db.Where("id = ?", categoryID).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(category)
}

// CreateKBCategory creates a new knowledge base category
func (c Controller) CreateKBCategory(request *evo.Request) any {
	var req struct {
		Name        string  `json:"name" validate:"required"`
		Description string  `json:"description"`
		Icon        string  `json:"icon"`
		Color       string  `json:"color"`
		ParentID    *string `json:"parent_id"`
		SortOrder   int     `json:"sort_order"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Generate slug
	slug := generateSlug(req.Name)
	var count int64
	db.Model(&models.KnowledgeBaseCategory{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		slug = slug + "-" + time.Now().Format("20060102150405")
	}

	category := models.KnowledgeBaseCategory{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		SortOrder:   req.SortOrder,
	}

	if req.ParentID != nil && *req.ParentID != "" {
		if parentID, err := uuid.Parse(*req.ParentID); err == nil {
			category.ParentID = &parentID
		}
	}

	if err := db.Create(&category).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.Created(category)
}

// UpdateKBCategory updates an existing knowledge base category
func (c Controller) UpdateKBCategory(request *evo.Request) any {
	id := request.Param("id").String()
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var category models.KnowledgeBaseCategory
	if err := db.Where("id = ?", categoryID).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Icon        string  `json:"icon"`
		Color       string  `json:"color"`
		ParentID    *string `json:"parent_id"`
		SortOrder   *int    `json:"sort_order"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.Name != "" {
		category.Name = req.Name
		// Update slug if name changed
		newSlug := generateSlug(req.Name)
		if newSlug != category.Slug {
			var count int64
			db.Model(&models.KnowledgeBaseCategory{}).Where("slug = ? AND id != ?", newSlug, category.ID).Count(&count)
			if count > 0 {
				newSlug = newSlug + "-" + time.Now().Format("20060102150405")
			}
			category.Slug = newSlug
		}
	}

	category.Description = req.Description
	category.Icon = req.Icon
	category.Color = req.Color

	if req.ParentID != nil {
		if *req.ParentID == "" {
			category.ParentID = nil
		} else if parentID, err := uuid.Parse(*req.ParentID); err == nil {
			category.ParentID = &parentID
		}
	}

	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}

	if err := db.Save(&category).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(category)
}

// DeleteKBCategory deletes a knowledge base category
func (c Controller) DeleteKBCategory(request *evo.Request) any {
	id := request.Param("id").String()
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var category models.KnowledgeBaseCategory
	if err := db.Where("id = ?", categoryID).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Check if there are articles using this category
	var articleCount int64
	db.Model(&models.KnowledgeBaseArticle{}).Where("category_id = ?", categoryID).Count(&articleCount)
	if articleCount > 0 {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Cannot delete category with articles", 400))
	}

	// Check if there are child categories
	var childCount int64
	db.Model(&models.KnowledgeBaseCategory{}).Where("parent_id = ?", categoryID).Count(&childCount)
	if childCount > 0 {
		return response.Error(response.NewError(response.ErrorCodeInvalidInput, "Cannot delete category with child categories", 400))
	}

	if err := db.Delete(&category).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Category deleted successfully"})
}

// ========================
// KNOWLEDGE BASE TAG APIs
// ========================

// ListKBTags returns all knowledge base tags
func (c Controller) ListKBTags(request *evo.Request) any {
	var tags []models.KnowledgeBaseTag
	query := db.Model(&models.KnowledgeBaseTag{}).Order("name ASC")

	if err := query.Find(&tags).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	// Calculate usage counts
	type TagCount struct {
		TagID uuid.UUID
		Count int
	}
	var counts []TagCount
	db.Model(&models.KnowledgeBaseArticleTag{}).
		Select("tag_id, COUNT(*) as count").
		Group("tag_id").
		Scan(&counts)

	countMap := make(map[uuid.UUID]int)
	for _, c := range counts {
		countMap[c.TagID] = c.Count
	}

	// Create response with usage counts
	type TagWithCount struct {
		models.KnowledgeBaseTag
		UsageCount int `json:"usage_count"`
	}

	result := make([]TagWithCount, len(tags))
	for i, tag := range tags {
		result[i] = TagWithCount{
			KnowledgeBaseTag: tag,
			UsageCount:       countMap[tag.ID],
		}
	}

	return response.OK(result)
}

// GetKBTag returns a single tag by ID
func (c Controller) GetKBTag(request *evo.Request) any {
	id := request.Param("id").String()
	tagID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var tag models.KnowledgeBaseTag
	if err := db.Where("id = ?", tagID).First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tag)
}

// CreateKBTag creates a new knowledge base tag
func (c Controller) CreateKBTag(request *evo.Request) any {
	var req struct {
		Name  string `json:"name" validate:"required"`
		Color string `json:"color"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	// Generate slug
	slug := generateSlug(req.Name)
	var count int64
	db.Model(&models.KnowledgeBaseTag{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		slug = slug + "-" + time.Now().Format("20060102150405")
	}

	tag := models.KnowledgeBaseTag{
		Name:  req.Name,
		Slug:  slug,
		Color: req.Color,
	}

	if err := db.Create(&tag).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.Created(tag)
}

// UpdateKBTag updates an existing knowledge base tag
func (c Controller) UpdateKBTag(request *evo.Request) any {
	id := request.Param("id").String()
	tagID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var tag models.KnowledgeBaseTag
	if err := db.Where("id = ?", tagID).First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.Name != "" {
		tag.Name = req.Name
		// Update slug if name changed
		newSlug := generateSlug(req.Name)
		if newSlug != tag.Slug {
			var count int64
			db.Model(&models.KnowledgeBaseTag{}).Where("slug = ? AND id != ?", newSlug, tag.ID).Count(&count)
			if count > 0 {
				newSlug = newSlug + "-" + time.Now().Format("20060102150405")
			}
			tag.Slug = newSlug
		}
	}

	tag.Color = req.Color

	if err := db.Save(&tag).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(tag)
}

// DeleteKBTag deletes a knowledge base tag
func (c Controller) DeleteKBTag(request *evo.Request) any {
	id := request.Param("id").String()
	tagID, err := uuid.Parse(id)
	if err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	var tag models.KnowledgeBaseTag
	if err := db.Where("id = ?", tagID).First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Error(response.ErrNotFound)
		}
		return response.Error(response.ErrInternalError)
	}

	// Delete article-tag associations
	db.Where("tag_id = ?", tagID).Delete(&models.KnowledgeBaseArticleTag{})

	// Delete tag
	if err := db.Delete(&tag).Error; err != nil {
		return response.Error(response.ErrInternalError)
	}

	return response.OK(map[string]string{"message": "Tag deleted successfully"})
}

// ========================
// KNOWLEDGE BASE MEDIA APIs
// ========================

// UploadKBMedia handles media upload for knowledge base articles
func (c Controller) UploadKBMedia(request *evo.Request) any {
	var req struct {
		Data string `json:"data" validate:"required"` // Base64 encoded image/video
		Type string `json:"type"`                     // image, video
	}

	if err := request.BodyParser(&req); err != nil {
		return response.Error(response.ErrInvalidInput)
	}

	if req.Type == "" {
		req.Type = "image"
	}

	// Determine the media type and extension
	var ext string
	var contentType string

	if req.Type == "video" {
		// For videos, we'll use MP4 as default
		ext = ".mp4"
		contentType = "video/mp4"
	} else {
		// For images, detect from base64 header
		if strings.HasPrefix(req.Data, "data:image/png") {
			ext = ".png"
			contentType = "image/png"
		} else if strings.HasPrefix(req.Data, "data:image/gif") {
			ext = ".gif"
			contentType = "image/gif"
		} else if strings.HasPrefix(req.Data, "data:image/webp") {
			ext = ".webp"
			contentType = "image/webp"
		} else {
			ext = ".jpg"
			contentType = "image/jpeg"
		}
	}

	// Save the file
	filename := uuid.New().String() + ext
	savePath := "uploads/kb/" + filename

	// Use imageutil for saving
	err := saveBase64File(req.Data, savePath, contentType)
	if err != nil {
		return response.Error(response.NewError(response.ErrorCodeInternalError, "Failed to save media: "+err.Error(), 500))
	}

	return response.OK(map[string]string{
		"url":  "/" + savePath,
		"type": req.Type,
	})
}

// ========================
// HELPER FUNCTIONS
// ========================

// generateSlug creates a URL-friendly slug from a string
func generateSlug(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]+")
	s = reg.ReplaceAllString(s, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile("-+")
	s = reg.ReplaceAllString(s, "-")

	// Trim hyphens from start and end
	s = strings.Trim(s, "-")

	return s
}

// saveBase64File saves a base64 encoded file to disk
func saveBase64File(data, path, contentType string) error {
	// Remove data URL prefix if present
	if idx := strings.Index(data, ","); idx != -1 {
		data = data[idx+1:]
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write file
	return os.WriteFile(path, decoded, 0644)
}
