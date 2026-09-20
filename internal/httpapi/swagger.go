package httpapi

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Hasras-code/PMT_WEB.git/internal/openapi"
	"github.com/go-chi/chi/v5"
	swaggerFiles "github.com/swaggo/files/v2"
)

// swaggerRoutes serves bundled assets and the live router contract. It does not
// depend on the working directory or a separate documentation generation step.
func (a *API) swaggerRoutes(r chi.Router) {
	r.Get("/swagger", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/", http.StatusTemporaryRedirect)
	})
	index := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerIndex))
	}
	r.Get("/swagger/", index)
	r.Get("/swagger/index.html", index)
	r.Get("/swagger/swagger-initializer.js", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write([]byte(swaggerInitializer))
	})
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

const swaggerIndex = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>PMT LMS API — Swagger UI</title>
  <link rel="stylesheet" href="/swagger/swagger-ui.css">
  <link rel="icon" type="image/png" href="/swagger/favicon-32x32.png">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="/swagger/swagger-ui-bundle.js"></script>
  <script src="/swagger/swagger-initializer.js"></script>
</body>
</html>`

const swaggerInitializer = `window.onload = function () {
  window.ui = SwaggerUIBundle({
    url: "/swagger/openapi.json",
    dom_id: "#swagger-ui",
    deepLinking: true,
    persistAuthorization: false,
    validatorUrl: null,
    presets: [SwaggerUIBundle.presets.apis],
    layout: "BaseLayout"
  });
};`
