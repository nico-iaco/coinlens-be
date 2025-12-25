package service

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
)

type StorageService struct {
	UploadDir string
}

func NewStorageService(uploadDir string) *StorageService {
	// Ensure dir exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		_ = os.MkdirAll(uploadDir, 0755)
	}
	return &StorageService{UploadDir: uploadDir}
}

func (s *StorageService) SaveFile(file multipart.File, filename string) error {
	path := filepath.Join(s.UploadDir, filename)

	dst, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return fmt.Errorf("failed to save file content: %w", err)
	}
	log.Printf("File saved to storage: %s", path)

	return nil
}
