package api

import (
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

// TestOpenAPICoversAllRoutes walks the registered routes and fails when a
// route is missing from openapi.yaml. The served spec is embedded from that
// file, so the docs can never quietly drift from the router.
func TestOpenAPICoversAllRoutes(t *testing.T) {
	raw, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}

	var spec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("openapi.yaml declares no paths — is it valid?")
	}

	documented := map[string]bool{}
	for path, ops := range spec.Paths {
		for method := range ops {
			if strings.EqualFold(method, "parameters") {
				continue
			}
			documented[strings.ToUpper(method)+" "+path] = true
		}
	}

	router := NewRouter(&Server{}).(*chi.Mux)
	registered := map[string]bool{}
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("walk router: %v", err)
	}
	// The catch-all handlers chi registers for unmatched requests.
	delete(registered, "/*")

	var missing []string
	for route := range registered {
		if !documented[route] {
			missing = append(missing, route)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("routes registered but not documented in openapi.yaml:\n  %s",
			strings.Join(missing, "\n  "))
	}
}
