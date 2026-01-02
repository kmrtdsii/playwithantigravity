package state

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
)

// FileCache caches file listings to avoid synchronous filesystem walks on every request.
type FileCache struct {
	mu       sync.RWMutex
	files    []string
	cachedAt time.Time
	dirty    bool
}

// MaxCacheAge defines how long the file cache is considered valid.
const MaxCacheAge = 5 * time.Second

// IsDirty returns true if the cache needs refreshing.
func (fc *FileCache) IsDirty() bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.dirty || time.Since(fc.cachedAt) > MaxCacheAge
}

// Get returns cached files if valid, nil otherwise.
func (fc *FileCache) Get() []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	if fc.dirty || time.Since(fc.cachedAt) > MaxCacheAge {
		return nil
	}
	result := make([]string, len(fc.files))
	copy(result, fc.files)
	return result
}

// Set updates the cache with new file list.
func (fc *FileCache) Set(files []string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.files = files
	fc.cachedAt = time.Now()
	fc.dirty = false
}

// Invalidate marks the cache as needing refresh.
func (fc *FileCache) Invalidate() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.dirty = true
}

// WalkFilesystem walks the filesystem and returns a list of files.
// This is extracted so it can be called in a background goroutine.
func WalkFilesystem(fs billy.Filesystem, startPath string, activeProject string) []string {
	const MaxFileCount = 1000
	count := 0
	var files []string

	log.Printf("Walking Filesystem: startPath=%s (ActiveProject=%s)", startPath, activeProject)

	_ = util.Walk(fs, startPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if count >= MaxFileCount {
			return filepath.SkipDir
		}

		// Calculate relative path for display (relative to startPath)
		relPath := path
		if startPath != "/" {
			if path == startPath {
				return nil // Skip the directory itself
			}
			if !strings.HasPrefix(path, startPath+"/") {
				return nil
			}
			relPath = strings.TrimPrefix(path, startPath+"/")
		} else {
			relPath = strings.TrimPrefix(path, "/")
		}

		if relPath != "" {
			// Skip .git directory content for the explorer
			if strings.Contains(relPath, ".git/") || relPath == ".git" {
				if fi.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			displayPath := relPath
			if fi.IsDir() {
				displayPath += "/"
			}
			files = append(files, displayPath)
			count++
		}

		return nil
	})

	log.Printf("Walking Filesystem: found %d files", len(files))
	if count >= MaxFileCount {
		files = append(files, "... (limit reached)")
	}

	return files
}
