# jimmyhamr

A minimal coding agent for the terminal. Built for local LLMs, also
runs on OpenAI-compatible endpoints.

![codehamr demo](codehamr.gif)

This is a fork of [codehamr/codehamr](https://github.com/codehamr/codehamr) with personal touch-ups and tweaks. The original project remains the upstream reference for full docs, config details, hardware recommendations, and more — this repo just ships my preferred defaults on top.

## Differences from upstream

| Change | This fork (`jbramnick/codehamr`) | Upstream (`codehamr/codehamr`) |
|---|---|---|
| **Name** | `jimmyhamr` — config lives in `.jimmyhamr/`, module path is `github.com/jbramnick/codehamr` | `codehamr` — config lives in `.codehamr/`, module path is `github.com/codehamr/codehamr` |
| **Cursor focus** | Cursor blink follows terminal focus: visible on focus, hidden on blur (`FocusMsg` → re-focus textarea, `BlurMsg` → blur it) | Focus/blur events are swallowed entirely to prevent escape fragments leaking into the prompt — cursor stays whatever state it was in |
| **HamrPass** | Removed. No HamrPass profile or config section is seeded on first run | Optional paid endpoint (`hamrpass`) seeded alongside `local` and `openai` profiles; waitlist at codehamr.com |
| **Import / Export** | Two new slash commands: `/export` writes the full conversation to `hamr_session_export.md`, `/import` loads it back into context and deletes the file. Useful for pausing a session and resuming later in a fresh run | Not available — no built-in way to persist and reload a conversation outside of chat history |

## Install

Linux, macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/jbramnick/codehamr/main/install.sh | bash
```

Windows:

```cmd
curl -fsSL https://raw.githubusercontent.com/jbramnick/codehamr/main/install.cmd -o install.cmd && install.cmd
```

Then run `codehamr` in your project.

> **Warning:** AI systems like codehamr run model-generated shell commands with full filesystem access. Best run inside safe sandboxes like devcontainers or isolated VMs.

## License

[MIT](LICENSE). Do whatever you want with it.
