package config

const (
	VisionProviderGroq      = "groq"
	VisionProviderDeepInfra = "deepinfra"
	VisionProviderTogether  = "together"
	VisionProviderOpenAI    = "openai"
	VisionProviderCustom    = "custom"
)

type visionProviderPreset struct {
	id    string
	url   string
	model string
}

var visionProviderPresets = []visionProviderPreset{
	{
		id:    VisionProviderGroq,
		url:   "https://api.groq.com/openai/v1",
		model: "meta-llama/llama-4-scout-17b-16e-instruct",
	},
	{
		id:    VisionProviderDeepInfra,
		url:   "https://api.deepinfra.com/v1/openai",
		model: "google/gemini-2.5-flash",
	},
	{
		id:    VisionProviderTogether,
		url:   "https://api.together.xyz/v1",
		model: "meta-llama/Llama-4-Scout-17B-16E-Instruct",
	},
	{
		id:    VisionProviderOpenAI,
		url:   "https://api.openai.com/v1",
		model: "gpt-4o-mini",
	},
}

func resolveVisionEndpoint(provider, customURL, customModel string) (string, string) {
	if provider == "" {
		provider = VisionProviderGroq
	}

	if provider != VisionProviderCustom {
		for _, preset := range visionProviderPresets {
			if preset.id == provider {
				return preset.url, preset.model
			}
		}
	}

	url := customURL
	model := customModel
	if url == "" {
		url = visionProviderPresets[0].url
	}
	if model == "" {
		model = visionProviderPresets[0].model
	}

	return url, model
}
