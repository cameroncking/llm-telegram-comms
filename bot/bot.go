package bot

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/exedev/llm-telegram-comms/backend"
	"github.com/exedev/llm-telegram-comms/config"
)

type Bot struct {
	cfg     *config.Config
	backend *backend.Backend
	bot     *bot.Bot
}

func New(cfg *config.Config) (*Bot, error) {
	b := &Bot{
		cfg:     cfg,
		backend: backend.New(cfg),
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(b.handleMessage),
	}

	tgBot, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating telegram bot: %w", err)
	}

	b.bot = tgBot
	return b, nil
}

func (b *Bot) Start(ctx context.Context) {
	log.Println("Bot starting...")
	b.bot.Start(ctx)
}

func (b *Bot) handleMessage(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	chatID := msg.Chat.ID
	userID := msg.From.ID

	// Check permissions
	if !b.isAllowed(msg) {
		log.Printf("Rejected message from user %d in chat %d", userID, chatID)
		return
	}

	// Handle attachments if enabled
	var savedAttachments []string
	if b.cfg.EnableAttachments {
		savedAttachments = b.handleAttachments(ctx, tgBot, msg, chatID)
	}

	if msg.Text == "" && msg.Caption == "" && len(savedAttachments) == 0 {
		return
	}

	// Use caption if text is empty (for messages with attachments)
	msgText := msg.Text
	if msgText == "" {
		msgText = msg.Caption
	}

	// Prepend attachment info based on attachment_method
	if b.cfg.EnableAttachments && len(savedAttachments) > 0 {
		var prefix string
		method := b.cfg.AttachmentMethod
		if method == "" {
			method = "xml"
		}

		switch method {
		case "xml":
			for _, filename := range savedAttachments {
				path := b.cfg.AttachmentPathChatPrefix + filename
				prefix += fmt.Sprintf("<attachment>%s</attachment>\n", path)
			}
		case "plaintext":
			for _, filename := range savedAttachments {
				path := b.cfg.AttachmentPathChatPrefix + filename
				prefix += fmt.Sprintf("Attachment: %s\n", path)
			}
		}
		msgText = prefix + msgText
	}

	log.Printf("Processing message from user %d in chat %d: %s", userID, chatID, truncate(msgText, 50))

	// Send typing indicator
	b.sendTypingAction(ctx, tgBot, chatID)

	// Start time for logging
	startTime := time.Now()

	// Build extra args for datasette_llm method
	var extraArgs []string
	if b.cfg.EnableAttachments && b.cfg.AttachmentMethod == "datasette_llm" && len(savedAttachments) > 0 {
		for _, filename := range savedAttachments {
			path := b.cfg.AttachmentPathChatPrefix + filename
			extraArgs = append(extraArgs, "-a", path)
		}
	}

	// Build exec options for chat type/id env vars
	execOpts := &backend.ExecOptions{
		ChatID: chatID,
	}
	if msg.Chat.Type == "private" {
		execOpts.ChatType = "user"
	} else {
		execOpts.ChatType = "group"
	}

	response, err := b.backend.Execute(ctx, msgText, execOpts, extraArgs...)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("Backend error (after %v): %v", elapsed, err)
		response = fmt.Sprintf("Error: %v", err)
	} else {
		log.Printf("Backend completed in %v", elapsed)
	}

	// Trim whitespace and check for empty response
	response = strings.TrimSpace(response)
	if response == "" {
		response = "(empty response)"
	}

	sendParams := &bot.SendMessageParams{
		ChatID: chatID,
		Text:   response,
	}

	// Only use reply in group chats, not in private 1:1 chats
	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		sendParams.ReplyParameters = &models.ReplyParameters{
			MessageID: msg.ID,
		}
	}

	_, err = tgBot.SendMessage(ctx, sendParams)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

func (b *Bot) isAllowed(msg *models.Message) bool {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	chatType := msg.Chat.Type

	// Check user allowlist
	if !b.cfg.IsUserAllowed(userID) {
		return false
	}

	// Check group allowlist for group/supergroup chats
	if chatType == "group" || chatType == "supergroup" {
		if !b.cfg.IsGroupAllowed(chatID) {
			return false
		}
	}

	return true
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (b *Bot) sendTypingAction(ctx context.Context, tgBot *bot.Bot, chatID int64) {
	_, err := tgBot.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: chatID,
		Action: models.ChatActionTyping,
	})
	if err != nil {
		log.Printf("Failed to send typing action: %v", err)
	}
}

func (b *Bot) handleAttachments(ctx context.Context, tgBot *bot.Bot, msg *models.Message, chatID int64) []string {
	var savedFiles []string
	var fileIDs []struct {
		fileID   string
		fileName string
	}

	// Check for photos (get largest size)
	if len(msg.Photo) > 0 {
		largest := msg.Photo[len(msg.Photo)-1]
		fileIDs = append(fileIDs, struct {
			fileID   string
			fileName string
		}{largest.FileID, "photo.jpg"})
	}

	// Check for document
	if msg.Document != nil {
		fileName := msg.Document.FileName
		if fileName == "" {
			fileName = "document"
		}
		fileIDs = append(fileIDs, struct {
			fileID   string
			fileName string
		}{msg.Document.FileID, fileName})
	}

	// Check for audio
	if msg.Audio != nil {
		fileName := msg.Audio.FileName
		if fileName == "" {
			fileName = "audio.mp3"
		}
		fileIDs = append(fileIDs, struct {
			fileID   string
			fileName string
		}{msg.Audio.FileID, fileName})
	}

	// Check for video
	if msg.Video != nil {
		fileName := msg.Video.FileName
		if fileName == "" {
			fileName = "video.mp4"
		}
		fileIDs = append(fileIDs, struct {
			fileID   string
			fileName string
		}{msg.Video.FileID, fileName})
	}

	// Check for voice
	if msg.Voice != nil {
		fileIDs = append(fileIDs, struct {
			fileID   string
			fileName string
		}{msg.Voice.FileID, "voice.ogg"})
	}

	// Check for video note
	if msg.VideoNote != nil {
		fileIDs = append(fileIDs, struct {
			fileID   string
			fileName string
		}{msg.VideoNote.FileID, "video_note.mp4"})
	}

	// Check for sticker
	if msg.Sticker != nil {
		ext := ".webp"
		if msg.Sticker.IsAnimated {
			ext = ".tgs"
		} else if msg.Sticker.IsVideo {
			ext = ".webm"
		}
		fileIDs = append(fileIDs, struct {
			fileID   string
			fileName string
		}{msg.Sticker.FileID, "sticker" + ext})
	}

	for _, f := range fileIDs {
		savedName, err := b.downloadFile(ctx, tgBot, f.fileID, f.fileName, chatID)
		if err != nil {
			log.Printf("Failed to download attachment %s: %v", f.fileName, err)
		} else {
			savedFiles = append(savedFiles, savedName)
		}
	}

	return savedFiles
}

func (b *Bot) downloadFile(ctx context.Context, tgBot *bot.Bot, fileID, originalName string, chatID int64) (string, error) {
	file, err := tgBot.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("getting file info: %w", err)
	}

	if file.FilePath == "" {
		return "", fmt.Errorf("file path is empty")
	}

	// Build download URL
	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.cfg.TelegramToken, file.FilePath)

	// Download the file
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Generate filename with timestamp
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	
	// Use original filename from file path if available and originalName is generic
	fileName := originalName
	if pathName := filepath.Base(file.FilePath); pathName != "" && !strings.HasPrefix(originalName, "photo") && !strings.HasPrefix(originalName, "video_note") && !strings.HasPrefix(originalName, "voice") && !strings.HasPrefix(originalName, "sticker") {
		// Keep the provided name
	} else if strings.Contains(file.FilePath, "/") {
		// For photos etc, use extension from file path
		ext := filepath.Ext(file.FilePath)
		if ext != "" {
			baseName := strings.TrimSuffix(originalName, filepath.Ext(originalName))
			fileName = baseName + ext
		}
	}

	// Sanitize filename to prevent path traversal
	fileName = filepath.Base(fileName)
	fileName = strings.ReplaceAll(fileName, "..", "_")

	destName := fmt.Sprintf("%s_%s", timestamp, fileName)

	// Determine attachment path
	attachmentDir := b.cfg.AttachmentPath
	if attachmentDir == "" {
		attachmentDir = "."
	}

	// Add chat ID suffix if configured
	if b.cfg.AttachmentPathChatIDSuffix {
		attachmentDir = filepath.Join(attachmentDir, fmt.Sprintf("%d", chatID))
		// Create the directory if it doesn't exist
		if err := os.MkdirAll(attachmentDir, 0755); err != nil {
			return "", fmt.Errorf("creating attachment directory: %w", err)
		}
	}

	destPath := filepath.Join(attachmentDir, destName)

	// Create the file
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	log.Printf("Saved attachment: %s", destPath)
	return destName, nil
}
