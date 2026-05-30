package llm

type Service interface {
	Chat(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error)
	ChatJSON(model, systemPrompt, userMessage string, temperature float64, maxTokens int) (string, error)
	ChatVision(model, systemPrompt, userText, imageBase64 string, temperature float64, maxTokens int) (string, error)
	ChatWithMessages(model string, messages []Message, temperature float64, maxTokens int) (string, error)
	GenerateSDPrompt(systemPrompt, description, presetType, model string, maxTokens int) (string, error)
	AnalyzeImage(model, systemPrompt, imageBase64 string, maxTokens int) (string, error)
	GetModels() ([]LLMModel, error)
	HealthCheck() error
	SetURL(baseURL string)
	SetBackend(backend string)
	SetBackendConfig(cfg BackendConfig)
}
