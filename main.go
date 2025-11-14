package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maxUploadSize = 50 << 20 // 50MB
	port          = "3003"
)

func main() {
	os.MkdirAll("tmp", 0755)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Middleware: limit body size (50MB)
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)
		c.Next()
	})

	// Serve static files safely (avoid wildcard conflict)
	r.Static("/tmp", "./tmp")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// Upload endpoint
	r.POST("/upload", handleUpload)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Cleanup goroutine
	go cleanOldTmpFiles()

	fmt.Printf("🚀 Server running at http://localhost:%s\n", port)
	r.Run(":" + port)
}

func handleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing file"})
		return
	}

	if file.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "File too large"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to open file"})
		return
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	newName := uuid.New().String() + ext
	dstPath := filepath.Join("tmp", newName)

	dst, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to save file"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to write file"})
		return
	}

	expiresAt := time.Now().Add(60 * time.Minute).Format("2006-01-02 15:04:05")
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s/tmp/%s", scheme, c.Request.Host, newName)
	isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"filename": file.Filename,
		"url":      url,
		"expires":  expiresAt,
		"isImage":  isImage,
	})
}

func cleanOldTmpFiles() {
	for {
		files, _ := os.ReadDir("tmp")
		for _, f := range files {
			info, err := os.Stat(filepath.Join("tmp", f.Name()))
			if err == nil && time.Since(info.ModTime()) > time.Hour {
				fmt.Println("🗑️ Removing old file:", f.Name())
				os.Remove(filepath.Join("tmp", f.Name()))
			}
		}
		time.Sleep(10 * time.Minute)
	}
}
