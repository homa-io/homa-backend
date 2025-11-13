package swagger

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/args"
)

type App struct{}

func (a App) Register() error {
	// No models to register for swagger app
	return nil
}

func (a App) Router() error {
	// Only register routes if --swagger flag is provided
	if !args.Exists("--swagger") {
		return nil
	}
	// Print information about Swagger UI availability
	println("📖 Swagger UI available at: http://localhost:8000/swagger")
	println("📄 OpenAPI spec available at: http://localhost:8000/swagger/openapi.json")
	var controller Controller

	// OpenAPI specification endpoint
	evo.Get("/swagger/openapi.json", controller.OpenAPISpecHandler)

	// Swagger UI routes
	evo.Get("/swagger", controller.SwaggerUIHandler)
	evo.Get("/swagger/", controller.SwaggerUIHandler)

	return nil
}

func (a App) WhenReady() error {
	return nil
}

func (a App) Name() string {
	return "swagger"
}
