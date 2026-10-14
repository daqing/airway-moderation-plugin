package moderation_api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daqing/airway-moderation-plugin/install/lib/moderation"
	"github.com/gin-gonic/gin"
)

const testSecret = "test-moderation-secret"

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Routes(router.Group("/api/v1/moderation"))
	return router
}

// loadWordList installs a word list every test screens against, so a test
// never depends on what the checkout happens to have on disk.
func loadWordList(t *testing.T, body string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "words.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write word list: %v", err)
	}
	if err := moderation.Setup(path); err != nil {
		t.Fatalf("setup word list: %v", err)
	}
}

func do(t *testing.T, router *gin.Engine, method, path, body string) (int, string) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if secret := os.Getenv("MODERATION_SECRET"); secret != "" {
		request.Header.Set(SecretHeader, secret)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response.Code, response.Body.String()
}

func decode(t *testing.T, body string) (int, map[string]any) {
	t.Helper()
	var envelope struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return envelope.Code, envelope.Data
}

func TestScreenReportsTheWordsThatHit(t *testing.T) {
	loadWordList(t, `{"categories":{"illegal":["网络赌博"],"porn":["裸聊"]}}`)
	router := setupRouter(t)

	status, body := do(t, router, http.MethodPost, "/api/v1/moderation/screen",
		`{"content":"有人在群里发广告，还提到网络赌博，大家小心"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	code, data := decode(t, body)
	if code != 0 || data["flagged"] != true {
		t.Fatalf("body = %s", body)
	}
	words, _ := data["words"].([]any)
	categories, _ := data["categories"].([]any)
	if len(words) != 1 || words[0] != "网络赌博" || len(categories) != 1 || categories[0] != "illegal" {
		t.Fatalf("body = %s", body)
	}
}

func TestScreenPassesOrdinaryContent(t *testing.T) {
	loadWordList(t, `{"categories":{"illegal":["网络赌博"]}}`)
	router := setupRouter(t)

	_, body := do(t, router, http.MethodPost, "/api/v1/moderation/screen", `{"content":"新版本什么时候发？"}`)
	code, data := decode(t, body)
	if code != 0 || data["flagged"] != false {
		t.Fatalf("body = %s", body)
	}
	// The arrays are always there, so a caller needs no nil checks.
	if !strings.Contains(body, `"words":[]`) || !strings.Contains(body, `"categories":[]`) {
		t.Fatalf("body = %s", body)
	}
}

func TestScreenRejectsUnusableRequests(t *testing.T) {
	loadWordList(t, `{"categories":{"illegal":["网络赌博"]}}`)
	router := setupRouter(t)

	for name, body := range map[string]string{
		"empty body":     "",
		"not json":       `content=网络赌博`,
		"too long":       `{"content":"` + strings.Repeat("a", maxContentBytes+1) + `"}`,
		"wrong shape":    `{"content":42}`,
		"unknown fields": `{"content":"hi","extra":"ignored"}`,
	} {
		status, responseBody := do(t, router, http.MethodPost, "/api/v1/moderation/screen", body)
		code, _ := decode(t, responseBody)
		switch name {
		case "unknown fields":
			if status != http.StatusOK || code != 0 {
				t.Fatalf("%s: status = %d, body = %s", name, status, responseBody)
			}
		default:
			if status != http.StatusOK || code != codeInvalidRequest {
				t.Fatalf("%s: status = %d, body = %s", name, status, responseBody)
			}
		}
	}
}

func TestGuardedEndpointsRequireTheSecret(t *testing.T) {
	loadWordList(t, `{"categories":{"illegal":["网络赌博"]}}`)
	router := setupRouter(t)
	t.Setenv("MODERATION_SECRET", testSecret)

	for _, path := range []string{"/api/v1/moderation/screen", "/api/v1/moderation/words", "/api/v1/moderation/reload"} {
		method := http.MethodGet
		if path != "/api/v1/moderation/words" {
			method = http.MethodPost
		}
		request := httptest.NewRequest(method, path, strings.NewReader(`{"content":"hi"}`))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		code, _ := decode(t, response.Body.String())
		if code != codeUnauthorized {
			t.Fatalf("%s without a secret: body = %s", path, response.Body.String())
		}
	}

	if _, body := do(t, router, http.MethodPost, "/api/v1/moderation/screen", `{"content":"hi"}`); !strings.Contains(body, `"code":0`) {
		t.Fatalf("body = %s", body)
	}
}

func TestHealthStaysOpen(t *testing.T) {
	loadWordList(t, `{"categories":{"illegal":["网络赌博"]}}`)
	router := setupRouter(t)
	t.Setenv("MODERATION_SECRET", testSecret)

	status, body := do(t, router, http.MethodGet, "/api/v1/moderation/health", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, body)
	}
	code, data := decode(t, body)
	if code != 0 || data["status"] != "ok" || data["source"] != "file" || data["words"] != float64(1) {
		t.Fatalf("body = %s", body)
	}
	// The path and the words stay behind the secret.
	if strings.Contains(body, "words.json") {
		t.Fatalf("health leaked the word list path: %s", body)
	}
}

func TestWordsReportsTheListInUse(t *testing.T) {
	loadWordList(t, `{"categories":{"illegal":["网络赌博"],"porn":["裸聊"]}}`)
	router := setupRouter(t)

	_, body := do(t, router, http.MethodGet, "/api/v1/moderation/words", "")
	code, data := decode(t, body)
	if code != 0 {
		t.Fatalf("body = %s", body)
	}
	categories, _ := data["categories"].([]any)
	if len(categories) != 2 || !strings.Contains(body, "网络赌博") || !strings.Contains(body, "裸聊") {
		t.Fatalf("body = %s", body)
	}
	source, _ := data["source"].(map[string]any)
	if source["embedded"] != false || source["words"] != float64(2) {
		t.Fatalf("body = %s", body)
	}
}

func TestReloadPicksUpAnEditedList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.json")
	if err := os.WriteFile(path, []byte(`{"categories":{"illegal":["旧词"]}}`), 0o644); err != nil {
		t.Fatalf("write word list: %v", err)
	}
	if err := moderation.Setup(path); err != nil {
		t.Fatalf("setup: %v", err)
	}
	router := setupRouter(t)

	if _, body := do(t, router, http.MethodPost, "/api/v1/moderation/screen", `{"content":"新词"}`); !strings.Contains(body, `"flagged":false`) {
		t.Fatalf("before reload: body = %s", body)
	}
	if err := os.WriteFile(path, []byte(`{"categories":{"illegal":["新词"]}}`), 0o644); err != nil {
		t.Fatalf("rewrite word list: %v", err)
	}
	if _, body := do(t, router, http.MethodPost, "/api/v1/moderation/reload", ""); !strings.Contains(body, `"code":0`) {
		t.Fatalf("reload: body = %s", body)
	}
	if _, body := do(t, router, http.MethodPost, "/api/v1/moderation/screen", `{"content":"新词"}`); !strings.Contains(body, `"flagged":true`) {
		t.Fatalf("after reload: body = %s", body)
	}
}

// A reload that fails leaves the list in use alone: an operator's typo must
// not stop screening.
func TestReloadRefusesABrokenList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.json")
	if err := os.WriteFile(path, []byte(`{"categories":{"illegal":["旧词"]}}`), 0o644); err != nil {
		t.Fatalf("write word list: %v", err)
	}
	if err := moderation.Setup(path); err != nil {
		t.Fatalf("setup: %v", err)
	}
	router := setupRouter(t)

	if err := os.WriteFile(path, []byte(`{"categories":`), 0o644); err != nil {
		t.Fatalf("rewrite word list: %v", err)
	}
	_, body := do(t, router, http.MethodPost, "/api/v1/moderation/reload", "")
	if code, _ := decode(t, body); code != codeWordList {
		t.Fatalf("body = %s", body)
	}
	if _, body := do(t, router, http.MethodPost, "/api/v1/moderation/screen", `{"content":"旧词"}`); !strings.Contains(body, `"flagged":true`) {
		t.Fatalf("the list in use was lost: body = %s", body)
	}
}
