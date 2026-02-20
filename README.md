# vox

Voice clone TTS CLI — powered by [Qwen3-TTS](https://github.com/QwenLM/Qwen3-TTS) via DashScope API.

Record your voice once, speak in any language.

## Install

```bash
go install github.com/ontypehq/vox@latest
```

## Quick Start

```bash
# Authenticate
vox auth login dashscope --token <your-api-key>

# Speak with a system voice
vox say "Hello world" --voice Cherry

# Clone your voice (from existing audio file)
vox voice record --file ~/my-voice.wav --name myvoice --lang zh

# Clone your voice (record from microphone)
vox voice record --lang zh --name myvoice

# Speak with your cloned voice
vox say "你好世界，这是我的声音。"

# Auto-reuses last voice — just type and speak
vox say "No need to pass --voice every time."
```

## Commands

```
vox auth login dashscope --token <key>   Save API credentials
vox auth status                          Show current auth status

vox say <text> [flags]                   Speak text with TTS
  -v, --voice    Voice ID or system voice name
  -l, --lang     Language hint (auto, Chinese, English, Japanese, ...)
  -o, --output   Save audio to WAV file
  --no-cache     Skip audio cache

vox voice list                           List system + cloned voices
vox voice record [flags]                 Record and enroll a voice clone
  -f, --file     Use existing audio file instead of recording
  -n, --name     Name for the cloned voice
  -l, --lang     Language for sample text (zh, en, ja)
  -d, --duration Recording duration in seconds (default: 10)
vox voice delete <voice-id>              Delete a cloned voice
```

## System Voices

| Voice | Gender | Language |
|-------|--------|----------|
| Cherry | Female | zh/en |
| Ethan | Male | zh/en |
| Chelsie | Female | zh/en |
| Serena | Female | zh/en |
| Dylan | Male | zh (Beijing) |
| Jada | Female | zh (Shanghai) |
| Sunny | Female | zh (Sichuan) |

## How It Works

- **TTS**: WebSocket streaming via DashScope Realtime API → direct audio playback (~500ms to first audio)
- **Voice Clone**: Upload reference audio → DashScope enrolls a voice profile → use the voice ID for TTS
- **Caching**: Generated audio is cached locally in `~/.vox/cache/` by content hash
- **State**: Last used voice ID is remembered in `~/.vox/state.json`

## API Key

Get a DashScope API key from [阿里云百炼](https://bailian.console.aliyun.com/). The key is stored locally in `~/.vox/config.json`.

## License

MIT
