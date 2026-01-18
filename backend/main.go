package main

import (
	"docker-downloader/handler"
	"docker-downloader/manager"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	currentDir, _ := os.Getwd()
	imagesDir := currentDir + "/images"

	log.Printf("Starting Docker Downloader Service...")
	log.Printf("Images will be stored in: %s", imagesDir)

	mgr := manager.NewManager(imagesDir)
	h := handler.NewHandler(mgr)

	r := gin.Default()

	frontendDir := os.Getenv("FRONTEND_DIST")
	if frontendDir == "" {
		frontendDir = currentDir + "/frontend-out"
	}

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/download", h.Download)
		api.GET("/list", h.ListImages)
		api.GET("/files/:name", h.ServeFile)
	}

	// Serve exported Next.js static files without conflicting with /api routes.
	// Using NoRoute avoids Gin's catch-all wildcard panic with existing /api prefix.
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// Clean and map requested path to file in frontendDir
		rel := filepath.Clean(path)
		if rel == "/" {
			rel = "/index.html"
		}
		target := filepath.Join(frontendDir, rel)

		// If file exists and is not a directory, serve it
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			c.File(target)
			return
		}

		// If it's a directory, try its index.html
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			idx := filepath.Join(target, "index.html")
			if _, err := os.Stat(idx); err == nil {
				c.File(idx)
				return
			}
		}

		// Try path/index.html for routes like /list
		idx := filepath.Join(frontendDir, rel, "index.html")
		if _, err := os.Stat(idx); err == nil {
			c.File(idx)
			return
		}

		// Fallback to root index.html
		fallback := filepath.Join(frontendDir, "index.html")
		if _, err := os.Stat(fallback); err == nil {
			c.File(fallback)
			return
		}

		c.Status(http.StatusNotFound)
	})

	log.Println("Server running on :8080")
	r.Run(":8080")
}
