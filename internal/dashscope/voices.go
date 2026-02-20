package dashscope

type SystemVoice struct {
	ID       string
	Name     string
	Language string
	Gender   string
	Model    string // which model to use
}

// System preset voices for qwen3-tts-flash-realtime
var SystemVoices = []SystemVoice{
	{ID: "Cherry", Name: "Cherry", Language: "zh/en", Gender: "Female", Model: ModelFlashRealtime},
	{ID: "Ethan", Name: "Ethan", Language: "zh/en", Gender: "Male", Model: ModelFlashRealtime},
	{ID: "Chelsie", Name: "Chelsie", Language: "zh/en", Gender: "Female", Model: ModelFlashRealtime},
	{ID: "Serena", Name: "Serena", Language: "zh/en", Gender: "Female", Model: ModelFlashRealtime},
	{ID: "Dylan", Name: "Dylan", Language: "zh (Beijing)", Gender: "Male", Model: ModelFlashRealtime},
	{ID: "Jada", Name: "Jada", Language: "zh (Shanghai)", Gender: "Female", Model: ModelFlashRealtime},
	{ID: "Sunny", Name: "Sunny", Language: "zh (Sichuan)", Gender: "Female", Model: ModelFlashRealtime},
}

// IsSystemVoice checks if the given voice ID is a system preset
func IsSystemVoice(voiceID string) bool {
	for _, v := range SystemVoices {
		if v.ID == voiceID {
			return true
		}
	}
	return false
}

// ModelForVoice returns the appropriate model for the given voice
func ModelForVoice(voiceID string) string {
	if IsSystemVoice(voiceID) {
		return ModelFlashRealtime
	}
	return ModelVCRealtime
}
