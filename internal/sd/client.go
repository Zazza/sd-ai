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
	Prompt                 string   `json:"prompt"`
	NegativePrompt         string   `json:"negative_prompt"`
	SamplerName            string   `json:"sampler_name"`
	Scheduler              string   `json:"scheduler,omitempty"`
	Steps                  int      `json:"steps"`
	CfgScale               float64  `json:"cfg_scale"`
	Width                  int      `json:"width"`
	Height                 int      `json:"height"`
	Seed                   *int64   `json:"seed,omitempty"`
	DenoisingStrength      *float64 `json:"denoising_strength,omitempty"`
	ClipSkip               *int     `json:"clip_skip,omitempty"`
	BatchSize              *int     `json:"batch_size,omitempty"`
	BatchCount             *int     `json:"n_iter,omitempty"`
	HiresFix               *bool    `json:"enable_hr,omitempty"`
	HiresUpscale           *float64 `json:"hr_scale,omitempty"`
	HiresDenoisingStrength *float64 `json:"hr_denoising_strength,omitempty"`
	HiresUpscaler          string   `json:"hr_upscaler,omitempty"`
	HiresResizeX           int      `json:"hr_resize_x"`
	HiresResizeY           int      `json:"hr_resize_y"`
	DoNotSaveImages        bool     `json:"do_not_save_images"`
	DoNotSaveGrid          bool     `json:"do_not_save_grid"`
}

type Txt2ImgResponse struct {
	Images     []string        `json:"images"`
	Parameters json.RawMessage `json:"parameters"`
	Info       json.RawMessage `json:"info"`
	Error      string          `json:"error"`
}

type SDModel struct {
	Title  string `json:"title"`
	Name   string `json:"model_name"`
	Hash   string `json:"hash"`
	Config string `json:"config"`
}

type Sampler struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
}

type Scheduler struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

type Upscaler struct {
	Name  string `json:"name"`
	Model string `json:"model_name"`
}

type VAE struct {
	ModelName string `json:"model_name"`
	Path      string `json:"path"`
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
		return nil, fmt.Errorf("API error %d: %s\nRequest: %s", resp.StatusCode, string(respBody), string(body))
	}

	var result Txt2ImgResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != "" {
		return &result, fmt.Errorf("SD error: %s", result.Error)
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

func (c *Client) GetSchedulers() ([]Scheduler, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/sdapi/v1/schedulers")
	if err != nil {
		return nil, fmt.Errorf("get schedulers: %w", err)
	}
	defer resp.Body.Close()

	var schedulers []Scheduler
	if err := json.NewDecoder(resp.Body).Decode(&schedulers); err != nil {
		return nil, fmt.Errorf("decode schedulers: %w", err)
	}
	return schedulers, nil
}

func (c *Client) GetUpscalers() ([]Upscaler, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/sdapi/v1/upscalers")
	if err != nil {
		return nil, fmt.Errorf("get upscalers: %w", err)
	}
	defer resp.Body.Close()

	var upscalers []Upscaler
	if err := json.NewDecoder(resp.Body).Decode(&upscalers); err != nil {
		return nil, fmt.Errorf("decode upscalers: %w", err)
	}
	return upscalers, nil
}

func (c *Client) GetVAEs() ([]VAE, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/sdapi/v1/sd-vae")
	if err != nil {
		return nil, fmt.Errorf("get vae: %w", err)
	}
	defer resp.Body.Close()

	var vaes []VAE
	if err := json.NewDecoder(resp.Body).Decode(&vaes); err != nil {
		return nil, fmt.Errorf("decode vae: %w", err)
	}
	return vaes, nil
}

type Img2ImgRequest struct {
	InitImages        []string `json:"init_images"`
	Prompt            string   `json:"prompt"`
	NegativePrompt    string   `json:"negative_prompt"`
	SamplerName       string   `json:"sampler_name"`
	Scheduler         string   `json:"scheduler,omitempty"`
	Steps             int      `json:"steps"`
	CfgScale          float64  `json:"cfg_scale"`
	Width             int      `json:"width"`
	Height            int      `json:"height"`
	Seed              *int64   `json:"seed,omitempty"`
	DenoisingStrength *float64 `json:"denoising_strength,omitempty"`
	ClipSkip          *int     `json:"clip_skip,omitempty"`
	BatchSize         *int     `json:"batch_size,omitempty"`
	BatchCount        *int     `json:"n_iter,omitempty"`
	DoNotSaveImages   bool     `json:"do_not_save_images"`
	DoNotSaveGrid     bool     `json:"do_not_save_grid"`
}

func (c *Client) Img2Img(req Img2ImgRequest) (*Txt2ImgResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/sdapi/v1/img2img", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s\nRequest: %s", resp.StatusCode, string(respBody), string(body))
	}

	var result Txt2ImgResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != "" {
		return &result, fmt.Errorf("SD error: %s", result.Error)
	}

	return &result, nil
}

func (c *Client) SetVAE(vaeName string) error {
	body, _ := json.Marshal(map[string]string{"sd_vae": vaeName})
	resp, err := c.httpClient.Post(c.baseURL+"/sdapi/v1/options", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("set vae: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

type LoRA struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (c *Client) GetLoRAs() ([]LoRA, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/sdapi/v1/loras")
	if err != nil {
		return nil, fmt.Errorf("get loras: %w", err)
	}
	defer resp.Body.Close()

	var loras []LoRA
	if err := json.NewDecoder(resp.Body).Decode(&loras); err != nil {
		return nil, fmt.Errorf("decode loras: %w", err)
	}
	return loras, nil
}
