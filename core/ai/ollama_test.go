package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withOllamaServer replaces the default ollama address with a test server
// for the duration of the test, then restores it.
func withOllamaServer(t *testing.T, handler http.Handler) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := ollamaDefaultAddr
	ollamaDefaultAddr = server.URL
	t.Cleanup(func() { ollamaDefaultAddr = original })
}

func TestCheckOllama_serviceRunningWithModels(t *testing.T) {
	withOllamaServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tags", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		resp := ollamaTagsResponse{
			Models: []ollamaModel{
				{Name: "llama3.2:latest"},
				{Name: "codellama:7b"},
			},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))

	status := CheckOllama()
	assert.True(t, status.Running)
	require.Len(t, status.Models, 2)
	assert.Equal(t, "llama3.2:latest", status.Models[0])
	assert.Equal(t, "codellama:7b", status.Models[1])
}

func TestCheckOllama_serviceRunningNoModels(t *testing.T) {
	withOllamaServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ollamaTagsResponse{Models: []ollamaModel{}}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))

	status := CheckOllama()
	assert.True(t, status.Running)
	assert.Empty(t, status.Models)
}

func TestCheckOllama_serviceRunningInvalidJSON(t *testing.T) {
	withOllamaServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))

	status := CheckOllama()
	assert.True(t, status.Running)
	assert.Empty(t, status.Models)
}

func TestCheckOllama_serviceReturnsError(t *testing.T) {
	withOllamaServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	status := CheckOllama()
	assert.False(t, status.Running)
}

func TestCheckOllama_serviceUnreachable(t *testing.T) {
	// Point to an address that will refuse connections.
	original := ollamaDefaultAddr
	ollamaDefaultAddr = "http://127.0.0.1:1"
	t.Cleanup(func() { ollamaDefaultAddr = original })

	status := CheckOllama()
	assert.False(t, status.Running)
	assert.Empty(t, status.Models)
}

func TestOllamaReady_binaryNotFound(t *testing.T) {
	result := DetectionResult{
		Provider: Provider{ID: "ollama", Name: "Ollama", Binary: "ollama"},
		Found:    false,
	}

	status, err := OllamaReady(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found on PATH")
	assert.False(t, status.Running)
}

func TestOllamaReady_serviceNotRunning(t *testing.T) {
	// Point to an address that will refuse connections.
	original := ollamaDefaultAddr
	ollamaDefaultAddr = "http://127.0.0.1:1"
	t.Cleanup(func() { ollamaDefaultAddr = original })

	result := DetectionResult{
		Provider: Provider{ID: "ollama", Name: "Ollama", Binary: "ollama"},
		Found:    true,
		Path:     "/usr/bin/ollama",
	}

	status, err := OllamaReady(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service is not running")
	assert.False(t, status.Running)
}

func TestOllamaReady_noModels(t *testing.T) {
	withOllamaServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ollamaTagsResponse{Models: []ollamaModel{}}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))

	result := DetectionResult{
		Provider: Provider{ID: "ollama", Name: "Ollama", Binary: "ollama"},
		Found:    true,
		Path:     "/usr/bin/ollama",
	}

	status, err := OllamaReady(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no models available")
	assert.True(t, status.Running)
}

func TestOllamaReady_fullyReady(t *testing.T) {
	withOllamaServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ollamaTagsResponse{
			Models: []ollamaModel{{Name: "llama3.2:latest"}},
		}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))

	result := DetectionResult{
		Provider: Provider{ID: "ollama", Name: "Ollama", Binary: "ollama"},
		Found:    true,
		Path:     "/usr/bin/ollama",
	}

	status, err := OllamaReady(result)
	assert.NoError(t, err)
	assert.True(t, status.Running)
	require.Len(t, status.Models, 1)
	assert.Equal(t, "llama3.2:latest", status.Models[0])
}
