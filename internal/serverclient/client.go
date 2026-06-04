package serverclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxServerResponseBodySize = 10 * 1024 * 1024

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type ProcessStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	PID      int    `json:"pid,omitempty"`
	Uptime   string `json:"uptime,omitempty"`
	Restarts int    `json:"restarts"`
	Category string `json:"category,omitempty"`
}

type HealthResult struct {
	Healthy   bool   `json:"healthy"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

type GPUInfo struct {
	Name        string `json:"name,omitempty"`
	MemoryTotal int    `json:"memory_total_mb,omitempty"`
	MemoryUsed  int    `json:"memory_used_mb,omitempty"`
	MemoryFree  int    `json:"memory_free_mb,omitempty"`
	Utilization int    `json:"utilization_percent,omitempty"`
	Available   bool   `json:"available"`
}

type InstallStatus struct {
	Key        string `json:"key"`
	Installed  bool   `json:"installed"`
	Installing bool   `json:"installing"`
	Progress   string `json:"progress"`
	Error      string `json:"error,omitempty"`
	Version    string `json:"version,omitempty"`
}

type ServerModels struct {
	SDCheckpoint string   `json:"sd_checkpoint,omitempty"`
	LLMRunning   []string `json:"llm_running,omitempty"`
}

type ServerStatus struct {
	Processes map[string]ProcessStatus `json:"processes"`
	Health    map[string]HealthResult  `json:"health"`
	GPU       GPUInfo                  `json:"gpu"`
	Installs  map[string]InstallStatus `json:"installs"`
	Models    ServerModels             `json:"models"`
}

type ModelInfo struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Extension string `json:"extension,omitempty"`
}

type LLMModelInfo struct {
	Name string `json:"name"`
	Size string `json:"size,omitempty"`
}

type BackendInfo struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func sanitizePathSegment(s string) error {
	if strings.Contains(s, "/") || strings.Contains(s, "\\") || strings.Contains(s, "..") {
		return fmt.Errorf("invalid path segment: %q", s)
	}
	return nil
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) SetBaseURL(url string) {
	c.baseURL = strings.TrimRight(url, "/")
}

func (c *Client) GetBaseURL() string {
	return c.baseURL
}

func (c *Client) ProxyURLs() (sdURL, llmURL, rembgURL string) {
	base := c.baseURL
	sdURL = base + "/api/sd"
	llmURL = base + "/api/llm"
	rembgURL = base + "/api/rembg"
	return
}

func (c *Client) HealthCheck() error {
	resp, err := c.httpClient.Get(c.baseURL + "/")
	if err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetStatus() (*ServerStatus, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/server/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status ServerStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *Client) GetModels(modelType string) ([]ModelInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/server/models/" + modelType)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var models []ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, err
	}
	return models, nil
}

func (c *Client) GetLLMModels() ([]LLMModelInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/server/models/llm")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var models []LLMModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, err
	}
	return models, nil
}

func (c *Client) DownloadModel(modelType, url, filename string) error {
	if err := sanitizePathSegment(filename); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"url": url, "filename": filename})
	resp, err := c.httpClient.Post(c.baseURL+"/api/server/models/"+modelType, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("download failed: %s", errResp["error"])
		}
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) DeleteModel(modelType, filename string) error {
	if err := sanitizePathSegment(filename); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/server/models/delete/"+modelType+"/"+filename, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxServerResponseBodySize))
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("delete failed: %s", errResp["error"])
		}
		return fmt.Errorf("delete failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) PullLLMModel(name string) error {
	body, _ := json.Marshal(map[string]string{"name": name})
	resp, err := c.httpClient.Post(c.baseURL+"/api/server/models/llm", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("pull failed: %s", errResp["error"])
		}
		return fmt.Errorf("pull failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) DeleteLLMModel(name string) error {
	if err := sanitizePathSegment(name); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/server/models/delete/llm/"+name, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxServerResponseBodySize))
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("delete failed: %s", errResp["error"])
		}
		return fmt.Errorf("delete failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) StartProcess(name string) error {
	if err := sanitizePathSegment(name); err != nil {
		return err
	}
	resp, err := c.httpClient.Post(c.baseURL+"/api/server/start/"+name, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("start failed: %s", errResp["error"])
		}
		return fmt.Errorf("start failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) StopProcess(name string) error {
	if err := sanitizePathSegment(name); err != nil {
		return err
	}
	resp, err := c.httpClient.Post(c.baseURL+"/api/server/stop/"+name, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("stop failed: %s", errResp["error"])
		}
		return fmt.Errorf("stop failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) RestartProcess(name string) error {
	resp, err := c.httpClient.Post(c.baseURL+"/api/server/restart/"+name, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("restart failed: %s", errResp["error"])
		}
		return fmt.Errorf("restart failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetBackends() ([]BackendInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/server/backends")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var backends []BackendInfo
	if err := json.NewDecoder(resp.Body).Decode(&backends); err != nil {
		return nil, err
	}
	return backends, nil
}

func (c *Client) SwitchBackend(backend string) error {
	body, _ := json.Marshal(map[string]string{"backend": backend})
	resp, err := c.httpClient.Post(c.baseURL+"/api/server/backends/switch", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf("switch failed: %s", errResp["error"])
		}
		return fmt.Errorf("switch failed: status %d", resp.StatusCode)
	}
	return nil
}
