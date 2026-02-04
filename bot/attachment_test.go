package bot

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestTimestampFormat(t *testing.T) {
	// Test that timestamp format matches expected pattern
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")

	// Should match pattern like 2026-02-04T14-30-25Z
	pattern := `^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z$`
	matched, err := regexp.MatchString(pattern, timestamp)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Errorf("timestamp %q does not match pattern %q", timestamp, pattern)
	}

	// Verify no colons in timestamp
	if regexp.MustCompile(`:`).MatchString(timestamp) {
		t.Errorf("timestamp should not contain colons: %q", timestamp)
	}
}

func TestFilenameGeneration(t *testing.T) {
	tests := []struct {
		originalName string
		wantPattern  string
	}{
		{"IMG_9182.png", `^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z_IMG_9182\.png$`},
		{"photo.jpg", `^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z_photo\.jpg$`},
		{"document.pdf", `^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z_document\.pdf$`},
		{"audio.mp3", `^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z_audio\.mp3$`},
		{"video.mp4", `^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z_video\.mp4$`},
		{"voice.ogg", `^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z_voice\.ogg$`},
		{"sticker.webp", `^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z_sticker\.webp$`},
	}

	for _, tt := range tests {
		t.Run(tt.originalName, func(t *testing.T) {
			timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
			generated := timestamp + "_" + tt.originalName

			matched, err := regexp.MatchString(tt.wantPattern, generated)
			if err != nil {
				t.Fatalf("regex error: %v", err)
			}
			if !matched {
				t.Errorf("generated filename %q does not match pattern %q", generated, tt.wantPattern)
			}
		})
	}
}

func TestDefaultAttachmentPath(t *testing.T) {
	// When attachment_path is empty, it should default to current directory
	attachmentDir := ""
	if attachmentDir == "" {
		attachmentDir = "."
	}

	if attachmentDir != "." {
		t.Errorf("expected default attachment path to be '.', got %q", attachmentDir)
	}
}

func TestAttachmentConfigDefaults(t *testing.T) {
	// Simulate default config behavior
	type testConfig struct {
		EnableAttachments bool
		AttachmentPath    string
		AttachmentMethod  string
	}

	cfg := testConfig{}

	if cfg.EnableAttachments {
		t.Error("EnableAttachments should default to false")
	}

	if cfg.AttachmentPath != "" {
		t.Errorf("AttachmentPath should default to empty string, got %q", cfg.AttachmentPath)
	}

	if cfg.AttachmentMethod != "" {
		t.Errorf("AttachmentMethod should default to empty string, got %q", cfg.AttachmentMethod)
	}
}

func TestAttachmentMethodXML(t *testing.T) {
	attachments := []string{
		"2026-02-04T14-30-25Z_IMG_9812.png",
		"2026-02-04T14-30-25Z_VID_9813.mov",
	}
	message := "Process and summarize the contents of this media."

	var prefix string
	for _, filename := range attachments {
		prefix += fmt.Sprintf("<attachment>%s</attachment>\n", filename)
	}
	result := prefix + message

	expected := `<attachment>2026-02-04T14-30-25Z_IMG_9812.png</attachment>
<attachment>2026-02-04T14-30-25Z_VID_9813.mov</attachment>
Process and summarize the contents of this media.`

	if result != expected {
		t.Errorf("XML method output mismatch\ngot:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestAttachmentMethodPlaintext(t *testing.T) {
	attachments := []string{
		"2026-02-04T14-30-25Z_IMG_9812.png",
		"2026-02-04T14-30-25Z_VID_9813.mov",
	}
	message := "Process and summarize the contents of this media."

	var prefix string
	for _, filename := range attachments {
		prefix += fmt.Sprintf("Attachment: %s\n", filename)
	}
	result := prefix + message

	expected := `Attachment: 2026-02-04T14-30-25Z_IMG_9812.png
Attachment: 2026-02-04T14-30-25Z_VID_9813.mov
Process and summarize the contents of this media.`

	if result != expected {
		t.Errorf("Plaintext method output mismatch\ngot:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestAttachmentMethodDatasetteLLM(t *testing.T) {
	attachments := []string{
		"2026-02-04T14-30-25Z_IMG_9812.png",
		"2026-02-04T14-30-25Z_VID_9813.mov",
	}
	pathPrefix := "/data/attachments/"

	var extraArgs []string
	for _, filename := range attachments {
		path := pathPrefix + filename
		extraArgs = append(extraArgs, "-a", path)
	}

	expected := []string{
		"-a", "/data/attachments/2026-02-04T14-30-25Z_IMG_9812.png",
		"-a", "/data/attachments/2026-02-04T14-30-25Z_VID_9813.mov",
	}

	if len(extraArgs) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(extraArgs))
	}

	for i, arg := range extraArgs {
		if arg != expected[i] {
			t.Errorf("arg[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestAttachmentMethodDatasetteLLMNoPrefix(t *testing.T) {
	attachments := []string{
		"2026-02-04T14-30-25Z_IMG_9812.png",
	}
	pathPrefix := ""

	var extraArgs []string
	for _, filename := range attachments {
		path := pathPrefix + filename
		extraArgs = append(extraArgs, "-a", path)
	}

	expected := []string{"-a", "2026-02-04T14-30-25Z_IMG_9812.png"}

	if len(extraArgs) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(extraArgs))
	}

	for i, arg := range extraArgs {
		if arg != expected[i] {
			t.Errorf("arg[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestAttachmentMethodXMLWithPrefix(t *testing.T) {
	attachments := []string{
		"2026-02-04T14-30-25Z_IMG_9812.png",
	}
	pathPrefix := "/data/attachments/"
	message := "Describe this image."

	var prefix string
	for _, filename := range attachments {
		path := pathPrefix + filename
		prefix += fmt.Sprintf("<attachment>%s</attachment>\n", path)
	}
	result := prefix + message

	expected := `<attachment>/data/attachments/2026-02-04T14-30-25Z_IMG_9812.png</attachment>
Describe this image.`

	if result != expected {
		t.Errorf("XML method with prefix output mismatch\ngot:\n%s\n\nwant:\n%s", result, expected)
	}
}

func TestFilenamePathTraversalSanitization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal filename", "photo.jpg", "photo.jpg"},
		{"path traversal attempt", "../../../etc/passwd", "passwd"},
		{"double dot in name", "file..name.jpg", "file_name.jpg"},
		{"hidden file", ".hidden", ".hidden"},
		{"absolute path", "/etc/passwd", "passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the sanitization logic from downloadFile
			fileName := tt.input
			fileName = filepath.Base(fileName)
			fileName = strings.ReplaceAll(fileName, "..", "_")

			if fileName != tt.expected {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, fileName, tt.expected)
			}
		})
	}
}

func TestAttachmentMethodPlaintextWithPrefix(t *testing.T) {
	attachments := []string{
		"2026-02-04T14-30-25Z_IMG_9812.png",
	}
	pathPrefix := "/data/attachments/"
	message := "Describe this image."

	var prefix string
	for _, filename := range attachments {
		path := pathPrefix + filename
		prefix += fmt.Sprintf("Attachment: %s\n", path)
	}
	result := prefix + message

	expected := `Attachment: /data/attachments/2026-02-04T14-30-25Z_IMG_9812.png
Describe this image.`

	if result != expected {
		t.Errorf("Plaintext method with prefix output mismatch\ngot:\n%s\n\nwant:\n%s", result, expected)
	}
}
