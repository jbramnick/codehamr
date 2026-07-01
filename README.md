# codehamr

A minimal coding agent for the terminal. Built for local LLMs, also
runs on OpenAI-compatible endpoints.

![codehamr demo](codehamr.gif)

This is a fork of [codehamr/codehamr](https://github.com/codehamr/codehamr) with personal touch-ups and tweaks. The original project remains the upstream reference for full docs, config details, hardware recommendations, and more — this repo just ships my preferred defaults on top.

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
