package llm

const (
	BackendLMStudio = "lmstudio"
	BackendOllama   = "ollama"
	BackendLlamaCpp = "llamacpp"
)

type BackendConfig struct {
	KeepAlive string
	NumCtx    int
	NumGPU    int
}

func DefaultURL(backend string) string {
	switch backend {
	case BackendOllama:
		return "http://localhost:11434"
	case BackendLlamaCpp:
		return "http://localhost:8081"
	default:
		return "http://localhost:1234"
	}
}

var BackendLabels = map[string]string{
	BackendLMStudio: "LM Studio",
	BackendOllama:   "Ollama",
	BackendLlamaCpp: "llama.cpp",
}
