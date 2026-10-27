package api

import _ "embed"

// OpenAPI is the Phase 1 HTTP contract served at /openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPI []byte

// SwaggerUI is the Swagger UI page served at /docs.
//
//go:embed swagger.html
var SwaggerUI []byte
