package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Hasras-code/PMT_WEB.git/internal/openapi"
	"github.com/go-chi/chi/v5"
	swaggerFiles "github.com/swaggo/files/v2"
	httpSwagger "github.com/swaggo/http-swagger"
)

// swaggerRoutes serves bundled assets and the live router contract. It does not
// depend on the working directory or a separate documentation generation step.
func (a *API) swaggerRoutes(r chi.Router) {
	a.swaggerRouter = r
	r.Get("/swagger", a.swaggerRedirectHandler)
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/openapi.json")))
	r.Get("/swagger/index.html", httpSwagger.Handler(httpSwagger.URL("/swagger/openapi.json")))
	r.Get("/swagger/openapi.json", a.wrap(a.swaggerOpenAPIHandler))
	assets := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFiles.FS)))
	for _, name := range []string{"swagger-ui.css", "swagger-ui-bundle.js", "favicon-16x16.png", "favicon-32x32.png"} {
		r.Get("/swagger/"+name, assets.ServeHTTP)
	}
}

// swaggerRedirectHandler godoc
//
//	@Summary Open Swagger UI
//	@Description Redirects to the bundled Swagger UI.
//	@Tags swagger
//	@Success 307
//	@Router /swagger [get]
func (a *API) swaggerRedirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger/index.html", http.StatusTemporaryRedirect)
}

// swaggerOpenAPIHandler godoc
//
//	@Summary Get the live OpenAPI document
//	@Tags swagger
//	@Produce json
//	@Success 200 {object} map[string]any
//	@Router /swagger/openapi.json [get]
func (a *API) swaggerOpenAPIHandler(w http.ResponseWriter, r *http.Request) error {
	a.swaggerOnce.Do(func() {
		var spec openapi.M
		spec, a.swaggerDocumentErr = openapi.Generate(a.swaggerRouter)
		if a.swaggerDocumentErr != nil {
			return
		}
		spec["servers"] = []openapi.M{{"url": "/"}}
		a.swaggerDocument, a.swaggerDocumentErr = json.Marshal(spec)
	})
	if a.swaggerDocumentErr != nil {
		return a.swaggerDocumentErr
	}
	return send(w, http.StatusOK, json.RawMessage(a.swaggerDocument))
}
