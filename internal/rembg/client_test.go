package rembg

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_SetsBaseURL(t *testing.T) {
	t.Parallel()
	c := New("http://localhost:7000")
	assert.Equal(t, "http://localhost:7000", c.baseURL)
	assert.NotNil(t, c.httpClient)
}

func TestNew_EmptyURL(t *testing.T) {
	t.Parallel()
	c := New("")
	assert.Equal(t, "", c.baseURL)
}

func TestSetURL(t *testing.T) {
	t.Parallel()
	c := New("http://old:7000")
	c.SetURL("http://new:8000")
	assert.Equal(t, "http://new:8000", c.baseURL)
}

func TestSetURL_EmptyString(t *testing.T) {
	t.Parallel()

	c := New("http://localhost:7000")
	c.SetURL("")
	assert.Equal(t, "", c.baseURL)
}

func TestHealthCheck_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL)
	err := c.HealthCheck()
	assert.NoError(t, err)
}

func TestHealthCheck_EmptyURL(t *testing.T) {
	t.Parallel()
	c := New("")
	err := c.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rembg URL not configured")
}

func TestHealthCheck_ServerUnavailable(t *testing.T) {
	t.Parallel()
	c := New("http://127.0.0.1:1")
	err := c.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rembg unavailable")
}

func TestHealthCheck_NonOKStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantMsg    string
	}{
		{name: "500_internal_error", statusCode: http.StatusInternalServerError, wantMsg: "rembg status 500"},
		{name: "503_unavailable", statusCode: http.StatusServiceUnavailable, wantMsg: "rembg status 503"},
		{name: "404_not_found", statusCode: http.StatusNotFound, wantMsg: "rembg status 404"},
		{name: "401_unauthorized", statusCode: http.StatusUnauthorized, wantMsg: "rembg status 401"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			c := New(srv.URL)
			err := c.HealthCheck()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestRemoveBackground_Success(t *testing.T) {
	t.Parallel()
	expectedResult := []byte("processed-image-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/remove", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		file, _, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()

		buf := make([]byte, 100)
		n, _ := file.Read(buf)
		assert.Equal(t, "test-image-bytes", string(buf[:n]))

		w.WriteHeader(http.StatusOK)
		w.Write(expectedResult)
	}))
	defer srv.Close()

	c := New(srv.URL)
	result, err := c.RemoveBackground([]byte("test-image-bytes"))
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

func TestRemoveBackground_EmptyURL(t *testing.T) {
	t.Parallel()
	c := New("")
	result, err := c.RemoveBackground([]byte("data"))
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rembg URL not configured")
}

func TestRemoveBackground_ServerError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantMsg    string
	}{
		{name: "400_bad_request", statusCode: http.StatusBadRequest, body: "bad request", wantMsg: "rembg error 400"},
		{name: "422_unprocessable", statusCode: http.StatusUnprocessableEntity, body: "invalid image", wantMsg: "rembg error 422"},
		{name: "500_internal", statusCode: http.StatusInternalServerError, body: "internal error", wantMsg: "rembg error 500"},
		{name: "503_unavailable", statusCode: http.StatusServiceUnavailable, body: "unavailable", wantMsg: "rembg error 503"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := New(srv.URL)
			result, err := c.RemoveBackground([]byte("data"))
			assert.Nil(t, result)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMsg)
			assert.Contains(t, err.Error(), tt.body)
		})
	}
}

func TestRemoveBackground_ConnectionRefused(t *testing.T) {
	t.Parallel()
	c := New("http://127.0.0.1:1")
	result, err := c.RemoveBackground([]byte("data"))
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rembg request failed")
}

func TestRemoveBackground_EmptyImageData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()

		body, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Empty(t, body)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("empty-result"))
	}))
	defer srv.Close()

	c := New(srv.URL)
	result, err := c.RemoveBackground([]byte{})
	require.NoError(t, err)
	assert.Equal(t, []byte("empty-result"), result)
}

func TestRemoveBackground_LargeResponse(t *testing.T) {
	largeResponse := make([]byte, 512*1024)
	for i := range largeResponse {
		largeResponse[i] = byte(i % 256)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeResponse)
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.httpClient = srv.Client()

	result, err := c.RemoveBackground([]byte("image"))
	require.NoError(t, err)
	assert.Equal(t, largeResponse, result)
}

func TestRemoveBackground_SendsMultipartFormFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		assert.True(t, strings.HasPrefix(contentType, "multipart/form-data"), "expected multipart content-type, got %s", contentType)

		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		assert.Equal(t, "image.png", header.Filename)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.httpClient = srv.Client()

	result, err := c.RemoveBackground([]byte("some-image"))
	require.NoError(t, err)
	assert.Equal(t, []byte("ok"), result)
}

func TestRemoveBackgroundBase64_Success(t *testing.T) {
	t.Parallel()
	processedData := []byte("processed-png-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(processedData)
	}))
	defer srv.Close()

	c := New(srv.URL)
	input := base64.StdEncoding.EncodeToString([]byte("input-image"))
	result, err := c.RemoveBackgroundBase64(input)
	require.NoError(t, err)

	expected := base64.StdEncoding.EncodeToString(processedData)
	assert.Equal(t, expected, result)
}

func TestRemoveBackgroundBase64_InvalidBase64(t *testing.T) {
	t.Parallel()
	c := New("http://localhost:7000")
	result, err := c.RemoveBackgroundBase64("!!!not-base64!!!")
	assert.Empty(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode base64")
}

func TestRemoveBackgroundBase64_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL)
	input := base64.StdEncoding.EncodeToString([]byte("input"))
	result, err := c.RemoveBackgroundBase64(input)
	assert.Empty(t, result)
	assert.Error(t, err)
}

func TestRemoveBackgroundBase64_EmptyURL(t *testing.T) {
	t.Parallel()

	c := New("")
	input := base64.StdEncoding.EncodeToString([]byte("data"))
	result, err := c.RemoveBackgroundBase64(input)
	assert.Empty(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rembg URL not configured")
}

func TestRemoveBackgroundBase64_RoundTrip(t *testing.T) {
	originalData := []byte("test-image-content-for-round-trip-verification")
	processedData := []byte("processed-result-content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()

		received, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, originalData, received)

		w.WriteHeader(http.StatusOK)
		w.Write(processedData)
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.httpClient = srv.Client()

	b64Input := base64.StdEncoding.EncodeToString(originalData)
	b64Output, err := c.RemoveBackgroundBase64(b64Input)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(b64Output)
	require.NoError(t, err)
	assert.Equal(t, processedData, decoded)
}
