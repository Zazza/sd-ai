package sd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryInfo(t *testing.T) {
	t.Parallel()

	t.Run("parses ram and cuda", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/sdapi/v1/memory" {
				t.Errorf("expected /sdapi/v1/memory, got %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ram":{"free":8589934592,"used":17179869184,"total":25769803776},"cuda":{"system":{"free":12884901888,"used":4294967296,"total":17179869184}}}`))
		}))
		defer server.Close()

		client := New(server.URL)
		stats, err := client.MemoryInfo()
		require.NoError(t, err)
		require.NotNil(t, stats)
		assert.Equal(t, float64(8589934592), stats.RAM.Free)
		assert.Equal(t, float64(17179869184), stats.RAM.Used)
		assert.Equal(t, float64(25769803776), stats.RAM.Total)
		require.NotNil(t, stats.CUDA)
		assert.Equal(t, float64(12884901888), stats.CUDA.System.Free)
		assert.Equal(t, float64(4294967296), stats.CUDA.System.Used)
		assert.Equal(t, float64(17179869184), stats.CUDA.System.Total)
	})

	t.Run("cpu only returns nil cuda", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ram":{"free":100,"used":28,"total":128}}`))
		}))
		defer server.Close()

		client := New(server.URL)
		stats, err := client.MemoryInfo()
		require.NoError(t, err)
		require.NotNil(t, stats)
		assert.Nil(t, stats.CUDA)
		assert.Equal(t, float64(100), stats.RAM.Free)
	})

	t.Run("http 404 returns error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		}))
		defer server.Close()

		client := New(server.URL)
		stats, err := client.MemoryInfo()
		require.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "status 404")
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("not-json"))
		}))
		defer server.Close()

		client := New(server.URL)
		stats, err := client.MemoryInfo()
		require.Error(t, err)
		assert.Nil(t, stats)
		assert.True(t, strings.Contains(err.Error(), "decode memory info"), err.Error())
	})
}
