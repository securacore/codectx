package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ollamaDefaultAddr is the default Ollama API endpoint. It is a var
// (not const) so tests can swap it for a test server.
var ollamaDefaultAddr = "http://localhost:11434"

const ollamaTimeout = 3 * time.Second

// OllamaStatus holds the result of checking the Ollama service.
type OllamaStatus struct {
	// Running reports whether the Ollama HTTP API is reachable.
	Running bool

	// Models lists the names of locally available models.
	// Empty when the service is not running or has no models pulled.
	Models []string
}

// ollamaTagsResponse mirrors the JSON shape returned by GET /api/tags.
type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name string `json:"name"`
}

// CheckOllama probes the Ollama HTTP API to determine whether the service
// is running and which models are available locally. It uses a short
// timeout to avoid blocking the CLI when the service is unreachable.
func CheckOllama() OllamaStatus {
	client := &http.Client{Timeout: ollamaTimeout}

	resp, err := client.Get(ollamaDefaultAddr + "/api/tags")
	if err != nil {
		return OllamaStatus{Running: false}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return OllamaStatus{Running: false}
	}

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		// Service is running but returned unexpected data.
		return OllamaStatus{Running: true}
	}

	models := make([]string, len(tags.Models))
	for i, m := range tags.Models {
		models[i] = m.Name
	}

	return OllamaStatus{Running: true, Models: models}
}

// OllamaReady reports whether Ollama is both installed and has at least
// one model available for use.
func OllamaReady(result DetectionResult) (OllamaStatus, error) {
	if !result.Found {
		return OllamaStatus{}, fmt.Errorf("ollama binary not found on PATH")
	}

	status := CheckOllama()
	if !status.Running {
		return status, fmt.Errorf("ollama binary found but service is not running")
	}

	if len(status.Models) == 0 {
		return status, fmt.Errorf("ollama service running but no models available — run 'ollama pull <model>'")
	}

	return status, nil
}
