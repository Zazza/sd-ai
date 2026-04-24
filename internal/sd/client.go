package sd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

func (c *Client) HealthCheck() error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(c.baseURL + "/sdapi/v1/sd-models")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetOptions() (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/sdapi/v1/options")
	if err != nil {
		return nil, fmt.Errorf("get options: %w", err)
	}
	defer resp.Body.Close()
	var opts map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&opts); err != nil {
		return nil, fmt.Errorf("decode options: %w", err)
	}
	return opts, nil
}

func (c *Client) SetURL(baseURL string) {
	c.baseURL = baseURL
}

type Txt2ImgRequest struct {
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	SamplerName    string  `json:"sampler_name"`
	Steps          int     `json:"steps"`
	CfgScale       float64 `json:"cfg_scale"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Seed           *int64  `json:"seed,omitempty"`
}

type Txt2ImgResponse struct {
	Images     []string        `json:"images"`
	Parameters json.RawMessage `json:"parameters"`
	Info       json.RawMessage `json:"info"`
}

type SDModel struct {
	Title  string `json:"title"`
	Name   string `json:"model_name"`
	Hash   string `json:"hash"`
	Config string `json:"config"`
}

type Sampler struct {
	Name      string `json:"name"`
	Aliases   []string `json:"aliases"`
}

func (c *Client) SetModel(modelName string) error {
	body, _ := json.Marshal(map[string]string{"sd_model_checkpoint": modelName})
	resp, err := c.httpClient.Post(c.baseURL+"/sdapi/v1/options", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("set model: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) Txt2Img(req Txt2ImgRequest) (*Txt2ImgResponse, error) {
	if req.Seed != nil {
		_ = c.SetModel("")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/sdapi/v1/txt2img", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result Txt2ImgResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) GetModels() ([]SDModel, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/sdapi/v1/sd-models")
	if err != nil {
		return nil, fmt.Errorf("get models: %w", err)
	}
	defer resp.Body.Close()

	var models []SDModel
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}
	return models, nil
}

func (c *Client) GetSamplers() ([]Sampler, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/sdapi/v1/samplers")
	if err != nil {
		return nil, fmt.Errorf("get samplers: %w", err)
	}
	defer resp.Body.Close()

	var samplers []Sampler
	if err := json.NewDecoder(resp.Body).Decode(&samplers); err != nil {
		return nil, fmt.Errorf("decode samplers: %w", err)
	}
	return samplers, nil
}
