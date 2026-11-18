package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSwaggerSpec(t *testing.T) {
	server := httptest.NewServer((&Server{}).Handler())
	defer server.Close()

	page := getText(t, server.URL+"/swagger/index.html")
	if !strings.Contains(page, "swagger-ui") {
		t.Fatalf("index is not the swagger page: %.200s", page)
	}

	specText := getText(t, server.URL+"/swagger/doc.json")
	var spec map[string]any
	if err := json.Unmarshal([]byte(specText), &spec); err != nil {
		t.Fatalf("doc.json: %v", err)
	}
	raw := specText
	for _, needle := range []string{
		"/studio",
		"/admin/weeks",
		"/admin/clients.csv",
		"ClassView",
		"price_min",
		"NO_CREDITS",
		"Authorization",
		"postpay",
	} {
		if !strings.Contains(raw, needle) {
			t.Fatalf("spec has no %q", needle)
		}
	}
	paths, _ := spec["paths"].(map[string]any)
	studio, _ := paths["/studio"].(map[string]any)
	getStudio, _ := studio["get"].(map[string]any)
	security, _ := getStudio["security"].([]any)
	if len(security) == 0 {
		t.Fatal("GET /studio has no security")
	}
}

func getText(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: %d %s", url, resp.StatusCode, body)
	}
	return string(body)
}
