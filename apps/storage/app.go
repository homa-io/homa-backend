package storage

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
)

// App represents the storage application
type App struct{}

// Register registers the storage routes
func (app App) Register() error {
	// Initialize S3 connection
	if err := Initialize(); err != nil {
		log.Warning("Failed to initialize S3 storage: %v", err)
	}

	// Register routes
	router := evo.GetFiber()

	// Admin routes (protected)
	admin := router.Group("/api/admin/storage")
	admin.Post("/multipart/create", CreateMultipartUploadHandler)
	admin.Post("/multipart/sign-part", SignPartHandler)
	admin.Post("/multipart/complete", CompleteMultipartUploadHandler)
	admin.Post("/multipart/abort", AbortMultipartUploadHandler)
	admin.Get("/multipart/parts", ListPartsHandler)
	admin.Post("/presign/upload", PresignUploadHandler)

	// Register media proxy
	RegisterMediaProxy(router)

	return nil
}

// Router returns the router interface
func (app App) Router() error {
	return nil
}

// WhenReady is called when application is ready
func (app App) WhenReady() error {
	return nil
}

// Name returns the application name
func (app App) Name() string {
	return "storage"
}
