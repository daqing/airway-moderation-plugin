package moderationplugin

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPluginIdentity(t *testing.T) {
	p := Plugin{}
	if p.Name() != "moderation" {
		t.Fatalf("unexpected plugin name %q", p.Name())
	}
	if p.MountPath() != "/api/v1/moderation" {
		t.Fatalf("unexpected mount path %q", p.MountPath())
	}
}

func TestPluginRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	p := Plugin{}
	p.Routes(router.Group(p.MountPath()))

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/moderation/health"},
		{http.MethodGet, "/api/v1/moderation/words"},
		{http.MethodPost, "/api/v1/moderation/screen"},
		{http.MethodPost, "/api/v1/moderation/reload"},
	} {
		request := httptest.NewRequest(route.method, route.path, strings.NewReader(`{"content":"hi"}`))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code == http.StatusNotFound {
			t.Fatalf("%s %s is not registered", route.method, route.path)
		}
	}
}

func TestBootLoadsTheConfiguredWordList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.json")
	if err := os.WriteFile(path, []byte(`{"categories":{"custom":["服务词表"]}}`), 0o644); err != nil {
		t.Fatalf("write word list: %v", err)
	}
	t.Setenv("MODERATION_WORDS_FILE", path)

	if err := (Plugin{}).Boot(); err != nil {
		t.Fatalf("Boot: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	(Plugin{}).Routes(router.Group(Plugin{}.MountPath()))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/moderation/screen", strings.NewReader(`{"content":"服务词表"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), `"flagged":true`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

// A word list that is present but unusable fails the boot: screening against
// a list nobody chose is worse than not starting.
func TestBootFailsOnABrokenWordList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.json")
	if err := os.WriteFile(path, []byte(`{"categories":`), 0o644); err != nil {
		t.Fatalf("write word list: %v", err)
	}
	t.Setenv("MODERATION_WORDS_FILE", path)

	if err := (Plugin{}).Boot(); err == nil {
		t.Fatal("Boot on a broken word list: got nil error, want failure")
	}
}

// A word list that is not there is not an error: a checkout that never ran
// plugin:install still screens, against the built-in list.
func TestBootFallsBackToTheBuiltInWordList(t *testing.T) {
	t.Setenv("MODERATION_WORDS_FILE", filepath.Join(t.TempDir(), "missing.json"))

	if err := (Plugin{}).Boot(); err != nil {
		t.Fatalf("Boot: %v", err)
	}
}
