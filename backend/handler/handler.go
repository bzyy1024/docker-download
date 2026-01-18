package handler

import (
	"docker-downloader/manager"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	mgr *manager.Manager
}

// ServeFile updates access time then serves the tar.
func (h *Handler) ServeFile(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	full := filepath.Join(h.mgr.OutputDir(), name)
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			c.Status(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Touch file to mark recent access for cleanup policy
	now := time.Now()
	_ = os.Chtimes(full, now, now)
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size()))
	c.File(full)
}

func NewHandler(mgr *manager.Manager) *Handler {
	return &Handler{mgr: mgr}
}

type DownloadRequest struct {
	Image    string `json:"image" binding:"required"`
	Registry string `json:"registry"`
	Username string `json:"username"`
	Password string `json:"password"`
	Platform string `json:"platform"`
}

func (h *Handler) Download(c *gin.Context) {
	var req DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	path, err := h.mgr.Download(manager.DownloadParams{
		Image:    req.Image,
		Registry: req.Registry,
		Username: req.Username,
		Password: req.Password,
		Platform: req.Platform,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Just return success and filename, frontend can then download it or list it
	filename := filepath.Base(path)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Download successful",
		"filename": filename,
		"url":      "/api/files/" + filename,
	})
}

func (h *Handler) ListImages(c *gin.Context) {
	keyword := c.Query("q")
	images, err := h.mgr.GetImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filtered := []string{}
	for _, img := range images {
		if keyword == "" || strings.Contains(strings.ToLower(img), strings.ToLower(keyword)) {
			filtered = append(filtered, img)
		}
	}

	c.JSON(http.StatusOK, gin.H{"images": filtered})
}
