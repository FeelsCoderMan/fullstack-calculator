// Package api embeds the OpenAPI contract and the Swagger UI page into the
// binary, so the server always documents exactly the API it was built with.
package api

import _ "embed" // for go:embed

// Spec is the OpenAPI 3.0 document (openapi.yaml): the single source of truth
// for the HTTP contract. Handlers are tested against it and the frontend
// generates its types from it.
//
//go:embed openapi.yaml
var Spec []byte

// SwaggerUI is the HTML page that renders Spec with Swagger UI.
//
//go:embed swagger.html
var SwaggerUI []byte
