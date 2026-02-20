---
name: vox
description: Speak text aloud with TTS voice cloning. Be the user's voice — read, narrate, announce, or just talk.
---

# vox

Speak text aloud through the terminal. Uses Qwen3-TTS via DashScope API with ~500ms latency. Supports system voices and user-cloned voices.

## When to Use

- User asks you to **read something aloud**, narrate, or speak
- User says "say this", "read this to me", "speak", "announce"
- User wants to **preview how text sounds** in a specific voice
- User wants to **test their cloned voice** with new text

## Commands

### Speak text

```bash
# Speak with the user's last-used voice (most common)
vox say "Your text here"

# Speak with a specific system voice
vox say "Hello world" --voice Cherry

# Specify language for better pronunciation
vox say "你好世界" --lang Chinese
vox say "こんにちは" --lang Japanese

# Save audio to file
vox say "Save this" --output ~/Desktop/output.wav
```

### Manage voices

```bash
# List all available voices (system + cloned)
vox voice list

# Enroll a new cloned voice from an audio file
vox voice record --file ~/my-recording.wav --name myvoice --lang zh

# Record from microphone (interactive)
vox voice record --lang zh --name myvoice --duration 10

# Delete a cloned voice
vox voice delete <voice-id>
```

### Auth

```bash
# Check if authenticated
vox auth status

# Login (only needed once)
vox auth login dashscope --token <api-key>
```

## Behavior

- **Auto voice**: `vox say` without `--voice` uses the last voice. No need to pass `--voice` every time.
- **Caching**: Same text + voice combination plays instantly from cache on repeat.
- **Streaming**: Audio streams to speaker as it generates — no wait for full download.
- **Language auto-detect**: Usually correct, but pass `--lang` for mixed-language or ambiguous text.

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

## Tips

- For long text, just pass it all — the API handles streaming well.
- If the user has a cloned voice set up, prefer using it (no `--voice` flag needed).
- Check `vox auth status` first if you get auth errors.
- Use `--output` when the user wants to keep the audio file.
