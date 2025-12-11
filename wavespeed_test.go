package wavespeed

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCompletes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"id":"pred-123","model":"wavespeed-ai/z-image/turbo","status":"processing","input":{"prompt":"Cat"},"outputs":[],"urls":{"get":"http://example.com"},"has_nsfw_contents":[],"created_at":"2024-01-01T00:00:00Z"}}`))
	})
	mux.HandleFunc("/api/v3/predictions/pred-123/result", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"id":"pred-123","model":"wavespeed-ai/z-image/turbo","status":"completed","input":{"prompt":"Cat"},"outputs":["https://img"],"urls":{"get":"http://example.com"},"has_nsfw_contents":[false],"created_at":"2024-01-01T00:00:00Z","executionTime":1234}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient("test-key", &ClientOptions{
		BaseURL:             server.URL,
		PollIntervalSeconds: floatPtr(0.01),
		TimeoutSeconds:      floatPtr(5),
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "Cat"}, nil)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if p.Status != "completed" {
		t.Fatalf("expected completed, got %s", p.Status)
	}
	if len(p.Outputs) != 1 || p.Outputs[0] != "https://img" {
		t.Fatalf("unexpected outputs: %+v", p.Outputs)
	}
	t.Logf("[RunCompletes] outputs=%v status=%s", p.Outputs, p.Status)
}

func TestRunTimeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"id":"pred-123","model":"wavespeed-ai/z-image/turbo","status":"processing","input":{"prompt":"Cat"},"outputs":[],"urls":{"get":"http://example.com"},"has_nsfw_contents":[],"created_at":"2024-01-01T00:00:00Z"}}`))
	})
	// always processing to trigger timeout
	mux.HandleFunc("/api/v3/predictions/pred-123/result", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"id":"pred-123","model":"wavespeed-ai/z-image/turbo","status":"processing","input":{"prompt":"Cat"},"outputs":[],"urls":{"get":"http://example.com"},"has_nsfw_contents":[],"created_at":"2024-01-01T00:00:00Z"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient("test-key", &ClientOptions{
		BaseURL:             server.URL,
		PollIntervalSeconds: floatPtr(0.01),
		TimeoutSeconds:      floatPtr(0.05),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "Cat"}, nil)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	t.Logf("[RunTimeout] timed out as expected: %v", err)
}

func TestUpload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/media/upload/binary", func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Authorization"); !strings.HasPrefix(ct, "Bearer ") {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "no file", http.StatusBadRequest)
			return
		}
		defer f.Close()
		content, _ := io.ReadAll(f)
		if string(content) != "hello" {
			http.Error(w, "bad content", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"download_url":"https://cdn/file.png"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient("test-key", &ClientOptions{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	tmp := filepath.Join(os.TempDir(), "wavespeed-go-test.txt")
	if err := os.WriteFile(tmp, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp)

	url, err := client.Upload(tmp)
	if err != nil {
		t.Fatalf("upload error: %v", err)
	}
	if url != "https://cdn/file.png" {
		t.Fatalf("unexpected url: %s", url)
	}
	t.Logf("[Upload] download_url=%s", url)
}

func floatPtr(f float64) *float64 {
	return &f
}

// --- Real API smoke tests (skip if env missing) ---

func TestRealRun(t *testing.T) {
	apiKey := os.Getenv("WAVESPEED_API_KEY")
	if apiKey == "" {
		t.Skip("WAVESPEED_API_KEY not set; skipping real API test")
	}
	base := os.Getenv("WAVESPEED_BASE_URL")
	client, err := NewClient(apiKey, &ClientOptions{
		BaseURL:             base,
		PollIntervalSeconds: floatPtr(1),
		TimeoutSeconds:      floatPtr(120),
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	out, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "Test image from go sdk"}, nil)
	if err != nil {
		t.Fatalf("Real run error: %v", err)
	}
	if len(out.Outputs) == 0 {
		t.Fatalf("Real run returned no outputs")
	}
	t.Logf("[RealRun] outputs[0]=%s status=%s", out.Outputs[0], out.Status)
}

func TestRealUpload(t *testing.T) {
	apiKey := os.Getenv("WAVESPEED_API_KEY")
	if apiKey == "" {
		t.Skip("WAVESPEED_API_KEY not set; skipping real upload test")
	}
	base := os.Getenv("WAVESPEED_BASE_URL")
	client, err := NewClient(apiKey, &ClientOptions{
		BaseURL: base,
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	// minimal PNG bytes (1x1 red pixel)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xFE,
		0xD4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	tmp := filepath.Join(os.TempDir(), "wavespeed-go-upload.png")
	if err := os.WriteFile(tmp, pngData, 0644); err != nil {
		t.Fatalf("write temp png: %v", err)
	}
	defer os.Remove(tmp)

	url, err := client.Upload(tmp)
	if err != nil {
		t.Fatalf("Real upload error: %v", err)
	}
	if url == "" {
		t.Fatalf("Real upload returned empty url")
	}
	t.Logf("[RealUpload] download_url=%s", url)
}