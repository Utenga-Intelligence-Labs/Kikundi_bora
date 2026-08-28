package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kikundibora/middleware"

	"github.com/gofiber/fiber/v2"
)

const maxUploadSize = 5 * 1024 * 1024 // 5MB

var allowedImageTypes = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

var allowedDocTypes = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".csv":  true,
}

// magic bytes for common image types
var imageMagicBytes = map[string]string{
	"\xff\xd8\xff":      ".jpg",
	"\x89PNG\r\n\x1a\n": ".png",
	"GIF87a":            ".gif",
	"GIF89a":            ".gif",
	"RIFF":              ".webp",
}

// magic bytes for common document types
var docMagicBytes = map[string][]byte{
	".pdf":  {0x25, 0x50, 0x44, 0x46},
	".doc":  {0xD0, 0xCF, 0x11, 0xE0},
	".docx": {0x50, 0x4B, 0x03, 0x04},
	".xls":  {0xD0, 0xCF, 0x11, 0xE0},
	".xlsx": {0x50, 0x4B, 0x03, 0x04},
}

func verifyDocMagicBytes(data []byte, expectedExt string) bool {
	magic, ok := docMagicBytes[expectedExt]
	if !ok {
		return true // CSV has no magic bytes, allow it
	}
	if len(data) < len(magic) {
		return false
	}
	for i, b := range magic {
		if data[i] != b {
			return false
		}
	}
	return true
}

type UploadHandler struct {
	BaseURL string
}

func NewUploadHandler(baseURL string) *UploadHandler {
	return &UploadHandler{BaseURL: baseURL}
}

func detectContentType(data []byte) string {
	for magic, ext := range imageMagicBytes {
		if len(data) >= len(magic) && string(data[:len(magic)]) == magic {
			return ext
		}
	}
	return ""
}

func (h *UploadHandler) UploadAvatar(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Faili haijapatikana"})
	}

	if file.Size > maxUploadSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Faili ni kubwa mno. Kiwango cha juu ni 5MB"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageTypes[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Aina ya faili hairuhusiwi. Tumia JPG, PNG, GIF au WebP"})
	}

	// Verify magic bytes
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusoma faili"})
	}
	defer src.Close()

	header := make([]byte, 12)
	if _, err := src.Read(header); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Faili imeharibika"})
	}

	detected := detectContentType(header)
	if detected != "" && detected != ext {
		// Allow .jpeg/.jpg mismatch
		if !(detected == ".jpg" && ext == ".jpeg") && !(detected == ".jpeg" && ext == ".jpg") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Aina ya faili haifanani na yaliyomo"})
		}
	}

	// Verify content type via http.DetectContentType
	contentType := http.DetectContentType(header)
	if !strings.HasPrefix(contentType, "image/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Faili si picha halali"})
	}

	userID := middleware.GetUserID(c)
	filename := fmt.Sprintf("%s_%d%s", userID, time.Now().UnixNano(), ext)
	savePath := filepath.Join("uploads", "avatars", filename)

	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi faili"})
	}

	url := fmt.Sprintf("%s/uploads/avatars/%s", h.BaseURL, filename)

	return c.JSON(fiber.Map{
		"message": "Faili imepakiwa",
		"url":     url,
	})
}

func (h *UploadHandler) UploadDoc(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Faili haijapatikana"})
	}

	if file.Size > maxUploadSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Faili ni kubwa mno. Kiwango cha juu ni 5MB"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := false
	for t := range allowedImageTypes {
		if t == ext {
			allowed = true
			break
		}
	}
	if !allowed {
		for t := range allowedDocTypes {
			if t == ext {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Aina ya faili hairuhusiwi"})
	}

	// Verify magic bytes for document types
	if allowedDocTypes[ext] {
		src, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusoma faili"})
		}
		header := make([]byte, 12)
		n, _ := src.Read(header)
		src.Close()
		if n > 0 && !verifyDocMagicBytes(header[:n], ext) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Aina ya faili haifanani na yaliyomo"})
		}
	}

	category := c.FormValue("category", "docs")
	if category != "docs" && category != "reports" && category != "contributions" {
		category = "docs"
	}

	userID := middleware.GetUserID(c)
	filename := fmt.Sprintf("%s_%d%s", userID, time.Now().UnixNano(), ext)
	savePath := filepath.Join("uploads", category, filename)

	dir := filepath.Join("uploads", category)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda folda"})
	}

	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi faili"})
	}

	url := fmt.Sprintf("%s/uploads/%s/%s", h.BaseURL, category, filename)

	return c.JSON(fiber.Map{
		"message": "Faili imepakiwa",
		"url":     url,
	})
}
