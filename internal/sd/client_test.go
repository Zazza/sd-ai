package sd

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsRetryableError_RetryableStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		statusCode int
		want       bool
	}{
		{name: "500_internal_server_error", statusCode: 500, want: true},
		{name: "502_bad_gateway", statusCode: 502, want: true},
		{name: "503_service_unavailable", statusCode: 503, want: true},
		{name: "504_gateway_timeout", statusCode: 504, want: true},
		{name: "200_ok", statusCode: 200, want: false},
		{name: "400_bad_request", statusCode: 400, want: false},
		{name: "401_unauthorized", statusCode: 401, want: false},
		{name: "404_not_found", statusCode: 404, want: false},
		{name: "429_too_many_requests", statusCode: 429, want: false},
		{name: "nil_error_nil_status", err: nil, statusCode: 0, want: false},
		{name: "nil_error_retryable_status", err: nil, statusCode: 503, want: true},
		{name: "nil_error_nonretryable_status", err: nil, statusCode: 200, want: false},
		{name: "url_error_timeout", err: &url.Error{Op: "Get", URL: "http://x", Err: errors.New("timeout")}, statusCode: 0, want: false},
		{name: "generic_error", err: errors.New("connection refused"), statusCode: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isRetryableError(tt.err, tt.statusCode)
			if got != tt.want {
				t.Errorf("isRetryableError(%v, %d) = %v, want %v", tt.err, tt.statusCode, got, tt.want)
			}
		})
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }
func (timeoutErr) Unwrap() error   { return net.ErrClosed }

var _ net.Error = timeoutErr{}

func TestIsRetryableError_URLErrorTimeout(t *testing.T) {
	t.Parallel()

	urlErr := &url.Error{
		Op:  "Get",
		URL: "http://localhost:8080/sdapi/v1/txt2img",
		Err: timeoutErr{},
	}
	if !urlErr.Timeout() {
		t.Fatal("expected url.Error.Timeout() to return true")
	}
	got := isRetryableError(urlErr, 0)
	if !got {
		t.Error("isRetryableError should return true for url.Error with timeout")
	}
}

func TestIsRetryableError_URLErrEOF(t *testing.T) {
	t.Parallel()

	eofErr := &url.Error{
		Op:  "Post",
		URL: "http://localhost/sdapi/v1/txt2img",
		Err: io.EOF,
	}
	got := isRetryableError(eofErr, 0)
	if !got {
		t.Error("isRetryableError should return true for url.Error wrapping io.EOF")
	}
}

func TestIsRetryableError_URLErrorNonTimeout(t *testing.T) {
	t.Parallel()

	nonTimeoutErr := &url.Error{
		Op:  "Get",
		URL: "http://localhost",
		Err: errors.New("connection refused"),
	}
	got := isRetryableError(nonTimeoutErr, 0)
	if got {
		t.Error("isRetryableError should return false for url.Error without timeout or EOF")
	}
}

func TestNew_SetsBaseURL(t *testing.T) {
	t.Parallel()

	c := New("http://localhost:7860")
	if c.baseURL != "http://localhost:7860" {
		t.Errorf("New().baseURL = %q, want %q", c.baseURL, "http://localhost:7860")
	}
	if c.httpClient == nil {
		t.Error("New().httpClient should not be nil")
	}
}

func TestSetURL(t *testing.T) {
	t.Parallel()

	c := New("http://old")
	c.SetURL("http://new:9090")
	if c.baseURL != "http://new:9090" {
		t.Errorf("SetURL: baseURL = %q, want %q", c.baseURL, "http://new:9090")
	}
}

func TestDoWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"images":["base64data"]}`))
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	resp, err := c.doWithRetry(func() (*http.Response, error) {
		return c.httpClient.Get(server.URL + "/test")
	})
	if err != nil {
		t.Fatalf("doWithRetry returned unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestDoWithRetry_Retry500_SuccessOnSecondAttempt(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`internal error`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	c.retryDelay = 0

	resp, err := c.doWithRetry(func() (*http.Response, error) {
		return c.httpClient.Get(server.URL + "/test")
	})
	if err != nil {
		t.Fatalf("doWithRetry returned unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDoWithRetry_Retry502_SuccessOnThirdAttempt(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`bad gateway`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	c.retryDelay = 0

	resp, err := c.doWithRetry(func() (*http.Response, error) {
		return c.httpClient.Get(server.URL + "/test")
	})
	if err != nil {
		t.Fatalf("doWithRetry returned unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDoWithRetry_ExhaustAllAttempts(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`unavailable`))
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	c.retryDelay = 0

	resp, err := c.doWithRetry(func() (*http.Response, error) {
		return c.httpClient.Get(server.URL + "/test")
	})
	if err == nil {
		t.Fatal("expected error when all retries exhausted")
	}
	if !strings.Contains(err.Error(), "request failed after") {
		t.Errorf("error should mention retry exhaustion, got: %v", err)
	}
	if !strings.Contains(err.Error(), "status 503") {
		t.Errorf("error should contain status code, got: %v", err)
	}
	if resp == nil {
		t.Error("expected non-nil response on exhausted retries with status code")
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDoWithRetry_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	c := New(serverURL)
	c.httpClient = &http.Client{Timeout: 100 * time.Millisecond}
	c.retryDelay = 0

	_, err := c.doWithRetry(func() (*http.Response, error) {
		return c.httpClient.Get(serverURL + "/test")
	})
	if err == nil {
		t.Fatal("expected error on network failure")
	}
}

func TestDoWithRetry_TimeoutErrorRetries(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			time.Sleep(200 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = &http.Client{Timeout: 50 * time.Millisecond}
	c.retryDelay = 0

	resp, err := c.doWithRetry(func() (*http.Response, error) {
		return c.httpClient.Get(server.URL + "/test")
	})
	if err != nil {
		t.Fatalf("doWithRetry returned unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if atomic.LoadInt32(&attempts) < 2 {
		t.Errorf("expected at least 2 attempts due to timeout retry, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDoWithRetry_NonRetryableHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`bad request`))
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	resp, err := c.doWithRetry(func() (*http.Response, error) {
		return c.httpClient.Get(server.URL + "/test")
	})
	if err != nil {
		t.Fatalf("doWithRetry returned unexpected error for <500 status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestDoPost_Success(t *testing.T) {
	wantImages := []string{"img1base64", "img2base64"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %s, want application/json", r.Header.Get("Content-Type"))
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if req["prompt"] != "a beautiful landscape" {
			t.Errorf("prompt = %v, want 'a beautiful landscape'", req["prompt"])
		}

		resp := Txt2ImgResponse{
			Images: wantImages,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()
	c.retryDelay = 0

	body, _ := json.Marshal(Txt2ImgRequest{Prompt: "a beautiful landscape", Steps: 20})
	result, err := c.doPost(server.URL+"/txt2img", body)
	if err != nil {
		t.Fatalf("doPost returned unexpected error: %v", err)
	}
	if len(result.Images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(result.Images))
	}
	if result.Images[0] != "img1base64" {
		t.Errorf("images[0] = %q, want %q", result.Images[0], "img1base64")
	}
	if result.Images[1] != "img2base64" {
		t.Errorf("images[1] = %q, want %q", result.Images[1], "img2base64")
	}
}

func TestDoPost_SDError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Txt2ImgResponse{
			Error: "out of memory",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()
	c.retryDelay = 0

	body, _ := json.Marshal(Txt2ImgRequest{Prompt: "test"})
	result, err := c.doPost(server.URL+"/txt2img", body)
	if err == nil {
		t.Fatal("expected error when SD returns error field")
	}
	if !strings.Contains(err.Error(), "out of memory") {
		t.Errorf("error should contain SD error message, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result even on SD error")
	}
	if result.Error != "out of memory" {
		t.Errorf("result.Error = %q, want %q", result.Error, "out of memory")
	}
}

func TestDoPost_EmptyImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Txt2ImgResponse{
			Images: []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()
	c.retryDelay = 0

	body, _ := json.Marshal(Txt2ImgRequest{Prompt: "test"})
	result, err := c.doPost(server.URL+"/txt2img", body)
	if err != nil {
		t.Fatalf("doPost returned unexpected error: %v", err)
	}
	if len(result.Images) != 0 {
		t.Errorf("len(images) = %d, want 0", len(result.Images))
	}
}

func TestDoPost_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"not found"}`))
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()
	c.retryDelay = 0

	body, _ := json.Marshal(Txt2ImgRequest{Prompt: "test"})
	_, err := c.doPost(server.URL+"/txt2img", body)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("error should mention API error, got: %v", err)
	}
}

func TestTxt2Img_SendsCorrectRequestBody(t *testing.T) {
	seed := int64(42)
	cfg := float64(7.5)
	batch := 2

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/txt2img" {
			t.Errorf("path = %s, want /sdapi/v1/txt2img", r.URL.Path)
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(bodyBytes, &req)

		if req["prompt"] != "a cat" {
			t.Errorf("prompt = %v, want 'a cat'", req["prompt"])
		}
		if req["negative_prompt"] != "blurry" {
			t.Errorf("negative_prompt = %v, want 'blurry'", req["negative_prompt"])
		}
		if req["sampler_name"] != "Euler a" {
			t.Errorf("sampler_name = %v, want 'Euler a'", req["sampler_name"])
		}
		if req["steps"] != float64(30) {
			t.Errorf("steps = %v, want 30", req["steps"])
		}
		if req["cfg_scale"] != 7.5 {
			t.Errorf("cfg_scale = %v, want 7.5", req["cfg_scale"])
		}
		if req["width"] != float64(512) {
			t.Errorf("width = %v, want 512", req["width"])
		}
		if req["height"] != float64(768) {
			t.Errorf("height = %v, want 768", req["height"])
		}
		if req["seed"] != float64(42) {
			t.Errorf("seed = %v, want 42", req["seed"])
		}
		if req["batch_size"] != float64(2) {
			t.Errorf("batch_size = %v, want 2", req["batch_size"])
		}

		resp := Txt2ImgResponse{Images: []string{"img"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()
	c.retryDelay = 0

	req := Txt2ImgRequest{
		Prompt:         "a cat",
		NegativePrompt: "blurry",
		SamplerName:    "Euler a",
		Steps:          30,
		CfgScale:       cfg,
		Width:          512,
		Height:         768,
		Seed:           &seed,
		BatchSize:      &batch,
	}
	result, err := c.Txt2Img(req)
	if err != nil {
		t.Fatalf("Txt2Img returned unexpected error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Errorf("len(images) = %d, want 1", len(result.Images))
	}
}

func TestTxt2Img_OmitsOmitEmptyFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(bodyBytes, &req)

		if _, ok := req["seed"]; ok {
			t.Error("seed should be omitted when nil")
		}
		if _, ok := req["scheduler"]; ok {
			t.Error("scheduler should be omitted when empty")
		}
		if _, ok := req["clip_skip"]; ok {
			t.Error("clip_skip should be omitted when nil")
		}
		if _, ok := req["enable_hr"]; ok {
			t.Error("enable_hr should be omitted when nil")
		}

		resp := Txt2ImgResponse{Images: []string{"img"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()
	c.retryDelay = 0

	req := Txt2ImgRequest{
		Prompt:      "test",
		Steps:       20,
		CfgScale:    7.0,
		Width:       512,
		Height:      512,
		SamplerName: "Euler",
	}
	result, err := c.Txt2Img(req)
	if err != nil {
		t.Fatalf("Txt2Img returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestImg2Img_SendsCorrectRequestBody(t *testing.T) {
	denoise := 0.75

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/img2img" {
			t.Errorf("path = %s, want /sdapi/v1/img2img", r.URL.Path)
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(bodyBytes, &req)

		if req["prompt"] != "a dog" {
			t.Errorf("prompt = %v, want 'a dog'", req["prompt"])
		}
		initImages, ok := req["init_images"].([]interface{})
		if !ok || len(initImages) != 1 {
			t.Errorf("init_images = %v, want array of 1 element", req["init_images"])
		}
		if req["denoising_strength"] != 0.75 {
			t.Errorf("denoising_strength = %v, want 0.75", req["denoising_strength"])
		}
		if req["mask"] != "maskbase64" {
			t.Errorf("mask = %v, want 'maskbase64'", req["mask"])
		}
		if req["mask_blur"] != float64(4) {
			t.Errorf("mask_blur = %v, want 4", req["mask_blur"])
		}
		if req["inpainting_fill"] != float64(1) {
			t.Errorf("inpainting_fill = %v, want 1", req["inpainting_fill"])
		}

		resp := Txt2ImgResponse{Images: []string{"result"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()
	c.retryDelay = 0

	req := Img2ImgRequest{
		InitImages:        []string{"sourcebase64"},
		Prompt:            "a dog",
		SamplerName:       "Euler",
		Steps:             25,
		CfgScale:          7.0,
		Width:             512,
		Height:            512,
		DenoisingStrength: &denoise,
		Mask:              "maskbase64",
		MaskBlur:          4,
		InpaintingFill:    1,
	}
	result, err := c.Img2Img(req)
	if err != nil {
		t.Fatalf("Img2Img returned unexpected error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Errorf("len(images) = %d, want 1", len(result.Images))
	}
}

func TestHealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/sd-models" {
			t.Errorf("path = %s, want /sdapi/v1/sd-models", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]SDModel{})
	}))
	defer server.Close()

	c := New(server.URL)
	if err := c.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck returned unexpected error: %v", err)
	}
}

func TestHealthCheck_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := New(server.URL)
	err := c.HealthCheck()
	if err == nil {
		t.Fatal("expected error for non-200 health check")
	}
	if !strings.Contains(err.Error(), "health check status") {
		t.Errorf("error should mention health check status, got: %v", err)
	}
}

func TestHealthCheck_ConnectionError(t *testing.T) {
	c := New("http://127.0.0.1:0")
	err := c.HealthCheck()
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
	if !strings.Contains(err.Error(), "health check failed") {
		t.Errorf("error should mention health check failed, got: %v", err)
	}
}

func TestGetModels(t *testing.T) {
	wantModels := []SDModel{
		{Title: "Model A", Name: "model_a", Hash: "abc123", Config: "config.yaml"},
		{Title: "Model B", Name: "model_b", Hash: "def456", Config: "config2.yaml"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/sd-models" {
			t.Errorf("path = %s, want /sdapi/v1/sd-models", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantModels)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	models, err := c.GetModels()
	if err != nil {
		t.Fatalf("GetModels returned unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}
	if models[0].Title != "Model A" {
		t.Errorf("models[0].Title = %q, want %q", models[0].Title, "Model A")
	}
	if models[1].Hash != "def456" {
		t.Errorf("models[1].Hash = %q, want %q", models[1].Hash, "def456")
	}
}

func TestGetSamplers(t *testing.T) {
	wantSamplers := []Sampler{
		{Name: "Euler a", Aliases: []string{"k_euler_a"}},
		{Name: "DPM++ 2M Karras", Aliases: []string{"k_dpmpp_2m_ka"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/samplers" {
			t.Errorf("path = %s, want /sdapi/v1/samplers", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantSamplers)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	samplers, err := c.GetSamplers()
	if err != nil {
		t.Fatalf("GetSamplers returned unexpected error: %v", err)
	}
	if len(samplers) != 2 {
		t.Fatalf("len(samplers) = %d, want 2", len(samplers))
	}
	if samplers[0].Name != "Euler a" {
		t.Errorf("samplers[0].Name = %q, want %q", samplers[0].Name, "Euler a")
	}
}

func TestGetSchedulers(t *testing.T) {
	wantSchedulers := []Scheduler{
		{Name: "automatic", Label: "Automatic"},
		{Name: "karras", Label: "Karras"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/schedulers" {
			t.Errorf("path = %s, want /sdapi/v1/schedulers", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantSchedulers)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	schedulers, err := c.GetSchedulers()
	if err != nil {
		t.Fatalf("GetSchedulers returned unexpected error: %v", err)
	}
	if len(schedulers) != 2 {
		t.Fatalf("len(schedulers) = %d, want 2", len(schedulers))
	}
	if schedulers[0].Label != "Automatic" {
		t.Errorf("schedulers[0].Label = %q, want %q", schedulers[0].Label, "Automatic")
	}
}

func TestGetUpscalers(t *testing.T) {
	wantUpscalers := []Upscaler{
		{Name: "Real-ESRGAN 4x", Model: "R-ESRGAN 4x+"},
		{Name: "Lanczos", Model: ""},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/upscalers" {
			t.Errorf("path = %s, want /sdapi/v1/upscalers", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantUpscalers)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	upscalers, err := c.GetUpscalers()
	if err != nil {
		t.Fatalf("GetUpscalers returned unexpected error: %v", err)
	}
	if len(upscalers) != 2 {
		t.Fatalf("len(upscalers) = %d, want 2", len(upscalers))
	}
	if upscalers[0].Model != "R-ESRGAN 4x+" {
		t.Errorf("upscalers[0].Model = %q, want %q", upscalers[0].Model, "R-ESRGAN 4x+")
	}
}

func TestGetVAEs(t *testing.T) {
	wantVAEs := []VAE{
		{ModelName: "vae-ft-mse", Path: "/models/vae/vae-ft-mse.safetensors"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/sd-vae" {
			t.Errorf("path = %s, want /sdapi/v1/sd-vae", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantVAEs)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	vaes, err := c.GetVAEs()
	if err != nil {
		t.Fatalf("GetVAEs returned unexpected error: %v", err)
	}
	if len(vaes) != 1 {
		t.Fatalf("len(vaes) = %d, want 1", len(vaes))
	}
	if vaes[0].ModelName != "vae-ft-mse" {
		t.Errorf("vaes[0].ModelName = %q, want %q", vaes[0].ModelName, "vae-ft-mse")
	}
}

func TestGetLoRAs(t *testing.T) {
	wantLoRAs := []LoRA{
		{Name: "detail_tweaker", Path: "/models/lora/detail_tweaker.safetensors"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/loras" {
			t.Errorf("path = %s, want /sdapi/v1/loras", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantLoRAs)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	loras, err := c.GetLoRAs()
	if err != nil {
		t.Fatalf("GetLoRAs returned unexpected error: %v", err)
	}
	if len(loras) != 1 {
		t.Fatalf("len(loras) = %d, want 1", len(loras))
	}
	if loras[0].Name != "detail_tweaker" {
		t.Errorf("loras[0].Name = %q, want %q", loras[0].Name, "detail_tweaker")
	}
}

func TestSetModel(t *testing.T) {
	var receivedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/options" {
			t.Errorf("path = %s, want /sdapi/v1/options", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	err := c.SetModel("v1-5-pruned-emaonly")
	if err != nil {
		t.Fatalf("SetModel returned unexpected error: %v", err)
	}
	if receivedBody["sd_model_checkpoint"] != "v1-5-pruned-emaonly" {
		t.Errorf("sd_model_checkpoint = %q, want %q", receivedBody["sd_model_checkpoint"], "v1-5-pruned-emaonly")
	}
}

func TestSetVAE(t *testing.T) {
	var receivedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/options" {
			t.Errorf("path = %s, want /sdapi/v1/options", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	err := c.SetVAE("vae-ft-mse")
	if err != nil {
		t.Fatalf("SetVAE returned unexpected error: %v", err)
	}
	if receivedBody["sd_vae"] != "vae-ft-mse" {
		t.Errorf("sd_vae = %q, want %q", receivedBody["sd_vae"], "vae-ft-mse")
	}
}

func TestGetOptions(t *testing.T) {
	wantOpts := map[string]interface{}{
		"sd_model_checkpoint": "v1-5-pruned-emaonly",
		"sd_vae":              "auto",
		"CLIP_stop_at_last_layers": float64(1),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdapi/v1/options" {
			t.Errorf("path = %s, want /sdapi/v1/options", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantOpts)
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()

	opts, err := c.GetOptions()
	if err != nil {
		t.Fatalf("GetOptions returned unexpected error: %v", err)
	}
	if opts["sd_model_checkpoint"] != "v1-5-pruned-emaonly" {
		t.Errorf("sd_model_checkpoint = %v, want 'v1-5-pruned-emaonly'", opts["sd_model_checkpoint"])
	}
}

func TestDoPost_RetryExhausted_ContainsResponseBody(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"CUDA out of memory"}`))
	}))
	defer server.Close()

	c := New(server.URL)
	c.httpClient = server.Client()
	c.retryDelay = 0
	c.retryMaxAttempts = 3

	body, _ := json.Marshal(Txt2ImgRequest{Prompt: "test"})
	_, err := c.doPost(server.URL+"/txt2img", body)
	if err == nil {
		t.Fatal("expected error when all retries exhausted")
	}
	if !strings.Contains(err.Error(), "CUDA out of memory") {
		t.Errorf("error should contain SD response body, got: %v", err)
	}
}
