package swagger

import (
	"github.com/getevo/evo/v2/lib/outcome"
	"path/filepath"

	"github.com/getevo/evo/v2"
	"github.com/iesreza/homa-backend/lib/response"
)

type Controller struct{}

// SwaggerUIHandler serves the Swagger UI interface
func (c Controller) SwaggerUIHandler(request *evo.Request) any {
	// Serve the index.html file from static/swagger
	return request.SendFile(filepath.Join("static", "swagger", "index.html"))
}

// OpenAPISpecHandler serves the OpenAPI JSON specification
func (c Controller) OpenAPISpecHandler(request *evo.Request) any {
	// Generate the OpenAPI specification
	spec := GenerateOpenAPI()

	// Convert to JSON
	jsonData, err := spec.ToJSON()
	if err != nil {
		return response.Error(response.ErrInternalError)
	}

	return outcome.Response{
		ContentType: "application/json",
		Data:        jsonData,
	}
}
