// Package swaggerui отдаёт встроенную страницу Swagger UI и OpenAPI-спеку.
//
// Ассеты Swagger UI берутся из github.com/swaggo/files/v2 (они вшиты в бинарь),
// спека - из файла, который генерирует protoc-gen-openapiv2. Интернет не нужен.
package swaggerui

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"

	swaggerFiles "github.com/swaggo/files/v2"
)

//go:embed openapi.swagger.json
var spec []byte

// SpecHandler отдаёт сгенерированную OpenAPI-спеку.
func SpecHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(spec)
	})
}

// Handler отдаёт Swagger UI. Единственное отличие от стокового dist - это
// swagger-initializer.js: в нём подменяется адрес спеки (по умолчанию там petstore).
//
// specURL - путь, по которому доступна спека, например "/openapi.json".
func Handler(specURL string) http.Handler {
	initializer := fmt.Appendf(nil, `window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: %q,
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: "StandaloneLayout"
  });
};
`, specURL)

	files := http.FileServer(http.FS(swaggerFiles.FS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.URL.Path, "/") == "swagger-initializer.js" {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			_, _ = w.Write(initializer)
			return
		}
		files.ServeHTTP(w, r)
	})
}
