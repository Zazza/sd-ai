package llm

const (
	BackendLMStudio = "lmstudio"
	BackendOllama   = "ollama"
	BackendLlamaCpp = "llamacpp"
)

type BackendConfig struct {
	KeepAlive  string
	NumCtx     int
	NumGPU     int
	NumPredict int
	NumThread  int
	TopP       float64
}

var BackendLabels = map[string]string{
	BackendLMStudio: "LM Studio",
	BackendOllama:   "Ollama",
	BackendLlamaCpp: "llama.cpp",
}
