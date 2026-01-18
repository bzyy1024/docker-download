package manager

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type DownloadParams struct {
	Image    string
	Registry string // Defaults to "docker.io" if empty or "docker"
	Username string
	Password string
	Platform string
}

type JobResult struct {
	Success bool
	Error   error
	Path    string
}

type Job struct {
	done   chan struct{}
	result JobResult
}

type Manager struct {
	mu        sync.Mutex
	jobs      map[string]*Job
	sem       chan struct{}
	outputDir string

	// housekeeping
	minFreeBytes  uint64
	maxAge        time.Duration
	cleanInterval time.Duration
}

func NewManager(outputDir string) *Manager {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.MkdirAll(outputDir, 0755)
	}

	mgr := &Manager{
		jobs:          make(map[string]*Job),
		sem:           make(chan struct{}, 3), // Max 3 concurrent downloads
		outputDir:     outputDir,
		minFreeBytes:  2 * 1024 * 1024 * 1024, // default 2GB
		maxAge:        7 * 24 * time.Hour,
		cleanInterval: time.Hour,
	}

	// Override by env if provided
	if v := os.Getenv("DOWNLOAD_MIN_FREE_BYTES"); v != "" {
		if parsed, err := parseUint(v); err == nil {
			mgr.minFreeBytes = parsed
		}
	}
	if v := os.Getenv("DOWNLOAD_MAX_AGE_HOURS"); v != "" {
		if parsed, err := parseUint(v); err == nil {
			mgr.maxAge = time.Duration(parsed) * time.Hour
		}
	}
	if v := os.Getenv("DOWNLOAD_CLEAN_INTERVAL_MINUTES"); v != "" {
		if parsed, err := parseUint(v); err == nil {
			mgr.cleanInterval = time.Duration(parsed) * time.Minute
		}
	}

	go mgr.periodicClean()

	return mgr
}

func parseUint(v string) (uint64, error) {
	var n uint64
	_, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (m *Manager) GetImages() ([]string, error) {
	var images []string
	files, err := os.ReadDir(m.outputDir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".tar") {
			images = append(images, f.Name())
		}
	}
	return images, nil
}

func (m *Manager) OutputDir() string {
	return m.outputDir
}

func (m *Manager) Download(params DownloadParams) (string, error) {
	// 1. Sanitize and Determine full image name
	imageName := params.Image
	if params.Registry != "" && params.Registry != "docker" && !strings.HasPrefix(imageName, params.Registry) {
		// If registry is custom, pre-pend it if not present, though docker pull usually handles "reg/img"
		// For simplicity, we assume params.Image includes strictly what "docker pull" needs,
		// OR we construct it.
		// Requirement: "specify docker registry address, default docker".
		// If user provides "myregistry.com" and image "nginx", we allow "myregistry.com/nginx".
		if !strings.Contains(imageName, "/") {
			imageName = fmt.Sprintf("%s/%s", params.Registry, imageName)
		}
	}

	// Default to latest if no tag
	if !strings.Contains(imageName, ":") {
		imageName = imageName + ":latest"
	}

	platform := strings.TrimSpace(params.Platform)
	if platform == "" {
		platform = "linux/amd64"
	}

	// Clean filename for storage
	safeName := strings.ReplaceAll(imageName, "/", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")
	safePlatform := strings.ReplaceAll(platform, "/", "_")
	targetFile := filepath.Join(m.outputDir, safeName+"__"+safePlatform+".tar")

	// 2. Check if already exists on disk
	if _, err := os.Stat(targetFile); err == nil {
		log.Printf("[Manager] Image %s (%s) already exists at %s", imageName, platform, targetFile)
		return targetFile, nil
	}

	// 2.5 ensure disk space before starting
	if err := m.ensureFreeSpace(); err != nil {
		return "", err
	}

	m.mu.Lock()
	// 3. Check if currently downloading
	jobKey := fmt.Sprintf("%s|%s", imageName, platform)
	if job, exists := m.jobs[jobKey]; exists {
		m.mu.Unlock()
		log.Printf("[Manager] Waiting for existing job for %s (%s)", imageName, platform)
		<-job.done // Wait for completion
		if job.result.Success {
			return job.result.Path, nil
		}
		return "", job.result.Error
	}

	// 4. Create new job
	job := &Job{
		done: make(chan struct{}),
	}
	m.jobs[jobKey] = job
	m.mu.Unlock()

	// 5. Execute asynchronously (but we wait here to return sync response as per requirement "return after download")
	// Actually, the requirement says "return to all requests".
	// The pattern here effectively blocks the FIRST request too.

	go func() {
		log.Printf("[Manager] Queuing download for %s (%s)", imageName, platform)
		m.sem <- struct{}{}        // Acquire token
		defer func() { <-m.sem }() // Release token

		log.Printf("[Manager] Starting download for %s (%s)", imageName, platform)

		var err error
		// Login if needed
		if params.Username != "" && params.Password != "" {
			// WARNING: This is basic. In prod, use stdin or config file to avoid password in process list.
			registry := params.Registry
			if registry == "" || registry == "docker" {
				registry = "docker.io"
			}
			cmd := exec.Command("docker", "login", "-u", params.Username, "-p", params.Password, registry)
			if out, e := cmd.CombinedOutput(); e != nil {
				log.Printf("[Manager] Login failed: %s", string(out))
				// Continue? Maybe public pull works. But likely fail.
				// We'll record error but try pull anyway just in case? No, usually fail specific.
			}
		}

		// Pull
		// --platform linux/amd64 as requested "amd default"
		pullCmd := exec.Command("docker", "pull", "--platform", platform, imageName)
		if out, e := pullCmd.CombinedOutput(); e != nil {
			err = fmt.Errorf("pull failed: %v, output: %s", e, string(out))
		} else {
			// Save
			saveCmd := exec.Command("docker", "save", "-o", targetFile, imageName)
			if out, e := saveCmd.CombinedOutput(); e != nil {
				err = fmt.Errorf("save failed: %v, output: %s", e, string(out))
				// Clean up partial file
				os.Remove(targetFile)
			} else {
				// update atime/mtime to now for housekeeping
				now := time.Now()
				os.Chtimes(targetFile, now, now)
			}
		}

		// Cleanup image from docker daemon to save space?
		// Requirement doesn't say, but good practice. keep it simple for now.

		m.mu.Lock()
		job.result.Success = (err == nil)
		job.result.Error = err
		job.result.Path = targetFile
		if err != nil {
			log.Printf("[Manager] Job failed for %s (%s): %v", imageName, platform, err)
		} else {
			log.Printf("[Manager] Job success for %s (%s)", imageName, platform)
		}
		close(job.done)
		delete(m.jobs, jobKey) // Remove from active jobs
		m.mu.Unlock()
	}()

	// Wait for the job that this specific request spawned (or found)
	<-job.done
	if job.result.Success {
		return job.result.Path, nil
	}
	return "", job.result.Error
}

// ensureFreeSpace deletes oldest images until free space is above threshold.
func (m *Manager) ensureFreeSpace() error {
	free, err := m.freeBytes()
	if err != nil {
		return err
	}
	if free >= m.minFreeBytes {
		return nil
	}

	log.Printf("[Manager] Free space %d below threshold %d, cleaning up", free, m.minFreeBytes)
	// delete oldest tar files until enough
	for free < m.minFreeBytes {
		removed, err := m.deleteOldest()
		if err != nil {
			return err
		}
		if !removed {
			return errors.New("no files to delete but space still low")
		}
		free, err = m.freeBytes()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) freeBytes() (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(m.outputDir, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

// deleteOldest removes the oldest .tar file by modtime.
func (m *Manager) deleteOldest() (bool, error) {
	entries, err := os.ReadDir(m.outputDir)
	if err != nil {
		return false, err
	}
	var files []os.DirEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tar") {
			continue
		}
		files = append(files, e)
	}
	if len(files) == 0 {
		return false, nil
	}

	sort.Slice(files, func(i, j int) bool {
		infoI, _ := files[i].Info()
		infoJ, _ := files[j].Info()
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	oldest := files[0]
	path := filepath.Join(m.outputDir, oldest.Name())
	log.Printf("[Manager] Deleting oldest image %s for space", path)
	if err := os.Remove(path); err != nil {
		return false, err
	}
	return true, nil
}

// periodicClean removes images not touched for maxAge.
func (m *Manager) periodicClean() {
	ticker := time.NewTicker(m.cleanInterval)
	defer ticker.Stop()
	for range ticker.C {
		if err := m.removeStale(); err != nil {
			log.Printf("[Manager] periodic cleanup error: %v", err)
		}
	}
}

func (m *Manager) removeStale() error {
	entries, err := os.ReadDir(m.outputDir)
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-m.maxAge)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tar") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(m.outputDir, e.Name())
			log.Printf("[Manager] Removing stale image %s (last mod %s)", path, info.ModTime())
			os.Remove(path)
		}
	}
	return nil
}
