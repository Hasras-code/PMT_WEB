package httpapi

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Hasras-code/PMT_WEB.git/internal/openapi"
	"github.com/go-chi/chi/v5"
	swaggerFiles "github.com/swaggo/files/v2"
	httpSwagger "github.com/swaggo/http-swagger"
)

// swaggerRoutes serves bundled assets and the live router contract. It does not
// depend on the working directory or a separate documentation generation step.
func (a *API) swaggerRoutes(r chi.Router) {
	r.Get("/swagger", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/index.html", http.StatusTemporaryRedirect)
	})
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/openapi.json")))
	r.Get("/swagger/index.html", httpSwagger.Handler(httpSwagger.URL("/swagger/openapi.json")))
	// Generate lazily, after the entire router has been registered, then cache
	// the immutable JSON. Air restarts the process whenever routes change.
	var once sync.Once
	var document []byte
	var documentErr error
	r.Get("/swagger/openapi.json", a.wrap(func(w http.ResponseWriter, req *http.Request) error {
		once.Do(func() {
			var spec openapi.M
			spec, documentErr = openapi.Generate(r)
			if documentErr != nil {
				return
			}
			// Try-it-out follows the serving origin, including custom Air ports.
			spec["servers"] = []openapi.M{{"url": "/"}}
			document, documentErr = json.Marshal(spec)
		})
		if documentErr != nil {
			return documentErr
		}
		return send(w, http.StatusOK, json.RawMessage(document))
	}))
	assets := http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFiles.FS)))
	for _, name := range []string{"swagger-ui.css", "swagger-ui-bundle.js", "favicon-16x16.png", "favicon-32x32.png"} {
		r.Get("/swagger/"+name, assets.ServeHTTP)
	}
}
