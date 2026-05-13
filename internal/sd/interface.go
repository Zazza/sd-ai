package sd

type Service interface {
	Txt2Img(req Txt2ImgRequest) (*Txt2ImgResponse, error)
	Img2Img(req Img2ImgRequest) (*Txt2ImgResponse, error)
	GetModels() ([]SDModel, error)
	GetSamplers() ([]Sampler, error)
	GetSchedulers() ([]Scheduler, error)
	GetUpscalers() ([]Upscaler, error)
	GetVAEs() ([]VAE, error)
	GetLoRAs() ([]LoRA, error)
	GetOptions() (map[string]interface{}, error)
	GetProgress() (*ProgressResponse, error)
	Interrupt() error
	HealthCheck() error
	SetURL(baseURL string)
	SetModel(modelName string) error
	SetVAE(vaeName string) error
	UpscaleImage(base64Img string, upscaler string, scale float64) (string, error)
}
