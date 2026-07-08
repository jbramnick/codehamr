// Package tools provides view_image for vision-capable models.
package tools

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"os"
)

// ViewImage reads an image file, validates it is a decodable image, and returns
// the base64-encoded data URL along with metadata (format, dimensions). The TUI
// stores the data URL in Message.ImageURL so llm.toWire sends it as an OpenAI
// multimodal content part instead of plain text — the model actually sees it.
func ViewImage(path string) (string, string) {
	if path == "" {
		return "", "(view_image error: empty path)"
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Sprintf("(view_image error: %v)", err)
	}

	// Decode to validate format and get dimensions.
	img, formatName, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Sprintf("(view_image error: not a supported image format — tried GIF/JPEG/PNG; got %v)", err)
	}
	bounds := img.Bounds()

	// Build the data URL. MIME type maps from Go's decoder name.
	mime := mimeTypeForFormat(formatName)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(raw))

	meta := fmt.Sprintf("Image: %s (%dx%d)", formatName, bounds.Dx(), bounds.Dy())
	return dataURL, meta
}

func mimeTypeForFormat(format string) string {
	switch format {
	case "png":
		return "image/png"
	case "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	default:
		return "image/" + format
	}
}

// ViewImageSchema is the OpenAI tool definition for view_image.
func ViewImageSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        ViewImageName,
			"description": "View an image file (PNG/JPEG/GIF). Returns a base64-encoded data URL that the vision model can see. Supports paths to local images.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or relative path to the image file (PNG, JPEG, or GIF).",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

// RunViewImage is the raw executor called by runRaw. Returns (dataURL, resultText).
func RunViewImage(path string) (string, string) {
	return ViewImage(path)
}
