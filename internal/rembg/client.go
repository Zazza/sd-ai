package rembg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
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
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) SetURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *Client) HasURL() bool {
	return c.baseURL != ""
}

func (c *Client) URL() string {
	return c.baseURL
}

func (c *Client) HealthCheck() error {
	if c.baseURL == "" {
		return fmt.Errorf("rembg URL not configured")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(c.baseURL + "/api")
	if err != nil {
		return fmt.Errorf("rembg unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rembg status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) RemoveBackground(imageData []byte) ([]byte, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("rembg URL not configured")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "image.png")
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return nil, fmt.Errorf("write image data: %w", err)
	}
	w.Close()

	resp, err := c.httpClient.Post(c.baseURL+"/api/remove", w.FormDataContentType(), &buf)
	if err != nil {
		return nil, fmt.Errorf("rembg request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rembg error %d: %s", resp.StatusCode, string(body))
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rembg response: %w", err)
	}

	return result, nil
}

func (c *Client) RemoveBackgroundBase64(base64Image string) (string, error) {
	imgData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	result, err := c.RemoveBackground(imgData)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(result), nil
}
