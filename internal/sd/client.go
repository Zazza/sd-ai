package sd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	retryMaxAttempts  = 3
	retryInitialDelay = 2 * time.Second
)

type Client struct {
	baseURL    string
	httpClient *http.Client

	retryMaxAttempts int
	retryDelay       time.Duration
}

var _ Service = (*Client)(nil)

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
		retryMaxAttempts: retryMaxAttempts,
		retryDelay:       retryInitialDelay,
	}
}

func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/sdapi/v1/options", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/sdapi/v1/options", nil)
	if err != nil {
		return nil, fmt.Errorf("get options: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	HiresSecondPassSteps   int      `json:"hr_second_pass_steps"`
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+"/sdapi/v1/options", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("set model: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("set model: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func isRetryableError(err error, statusCode int) bool {
	if statusCode == http.StatusServiceUnavailable || statusCode == http.StatusBadGateway || statusCode == http.StatusInternalServerError || statusCode == http.StatusGatewayTimeout {
		return true
	}
	if err == nil {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Timeout() || errors.Is(urlErr, io.EOF)
	}
	return false
}

func (c *Client) doWithRetry(fn func() (*http.Response, error)) (*http.Response, error) {
	var resp *http.Response
	var err error
	var lastErr error
	delay := c.retryDelay

	for attempt := 0; attempt < c.retryMaxAttempts; attempt++ {
		resp, err = fn()
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if err != nil {
			lastErr = err
			if !isRetryableError(err, 0) {
				return nil, err
			}
		} else {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			if !isRetryableError(nil, resp.StatusCode) {
				return resp, lastErr
			}
		}
		if resp != nil && attempt < retryMaxAttempts-1 {
			resp.Body.Close()
			resp = nil
		}
		if attempt < retryMaxAttempts-1 {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return resp, fmt.Errorf("request failed after %d attempts: %w", c.retryMaxAttempts, lastErr)
}

func (c *Client) doPost(url string, body []byte) (*Txt2ImgResponse, error) {
	resp, err := c.doWithRetry(func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return c.httpClient.Do(req)
	})
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			if respBody, readErr := io.ReadAll(resp.Body); readErr == nil && len(respBody) > 0 {
				return nil, fmt.Errorf("%w\nSD response: %s", err, string(respBody))
			}
		}
		return nil, err
	}

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

func (c *Client) Txt2Img(req Txt2ImgRequest) (*Txt2ImgResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return c.doPost(c.baseURL+"/sdapi/v1/txt2img", body)
}

func (c *Client) GetModels() ([]SDModel, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/sdapi/v1/sd-models", nil)
	if err != nil {
		return nil, fmt.Errorf("get models: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/sdapi/v1/samplers", nil)
	if err != nil {
		return nil, fmt.Errorf("get samplers: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/sdapi/v1/schedulers", nil)
	if err != nil {
		return nil, fmt.Errorf("get schedulers: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/sdapi/v1/upscalers", nil)
	if err != nil {
		return nil, fmt.Errorf("get upscalers: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/sdapi/v1/sd-vae", nil)
	if err != nil {
		return nil, fmt.Errorf("get vae: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	Mask                  string `json:"mask,omitempty"`
	MaskBlur              int    `json:"mask_blur,omitempty"`
	InpaintingFill        int    `json:"inpainting_fill,omitempty"`
	InpaintFullRes        bool   `json:"inpaint_full_res,omitempty"`
	InpaintFullResPadding int    `json:"inpaint_full_res_padding,omitempty"`
	DoNotSaveImages       bool   `json:"do_not_save_images"`
	DoNotSaveGrid         bool   `json:"do_not_save_grid"`
}

func (c *Client) Img2Img(req Img2ImgRequest) (*Txt2ImgResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return c.doPost(c.baseURL+"/sdapi/v1/img2img", body)
}

func (c *Client) SetVAE(vaeName string) error {
	body, _ := json.Marshal(map[string]string{"sd_vae": vaeName})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+"/sdapi/v1/options", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("set vae: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.baseURL+"/sdapi/v1/loras", nil)
	if err != nil {
		return nil, fmt.Errorf("get loras: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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

type ExtraImageResponse struct {
	Image string `json:"image"`
}

func (c *Client) UpscaleImage(base64Img string, upscaler string, scale float64) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"image":                          base64Img,
		"resize_mode":                    0,
		"show_extras":                    true,
		"gfpgan_visibility":              0,
		"codeformer_visibility":          0,
		"codeformer_weight":              0,
		"upscaling_resize":               scale,
		"upscaler_1":                     upscaler,
		"upscaler_2":                     "None",
		"extras_upscaler_2_visibility":   0,
		"upscale_first":                  false,
	})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+"/sdapi/v1/extra-single-img", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("upscale image: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upscale image: %w", err)
	}
	defer resp.Body.Close()

	var result ExtraImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode upscale response: %w", err)
	}
	if result.Image == "" {
		return "", fmt.Errorf("upscale image: empty response")
	}
	return result.Image, nil
}
