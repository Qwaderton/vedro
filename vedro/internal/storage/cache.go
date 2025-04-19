package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileMeta struct {
	Size         int64
	LastModified time.Time
	ETag         string
}

type Cache struct {
	sync.RWMutex
	data map[string]map[string]FileMeta
}

func NewCache() *Cache {
	return &Cache{data: make(map[string]map[string]FileMeta)}
}

func (c *Cache) Scan(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Printf("Error scanning root: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, entry := range entries {
		if entry.IsDir() {
			wg.Add(1)
			go func(e os.DirEntry) {
				defer wg.Done()
				bucket := e.Name()
				bucketPath := filepath.Join(root, bucket)
				c.scanBucket(bucket, bucketPath)
			}(entry)
		}
	}
	wg.Wait()
}

func (c *Cache) scanBucket(bucket, path string) {
	filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
		if filePath == path {
            return nil
        }

		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(path, filePath)
		if err != nil {
			log.Printf("Path error: %v", err)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			log.Printf("File info error: %v", err)
			return nil
		}

		isDir := d.IsDir()
		if isDir {
			relPath += "/"
		}

		c.RLock()
		existing, exists := c.data[bucket][relPath]
		c.RUnlock()

		if exists && existing.LastModified.Equal(info.ModTime().UTC()) && (isDir || existing.Size == info.Size()) {
			return nil
		}

		var meta FileMeta

		hash, err := ComputeETag(filePath, isDir)
		if err != nil {
			log.Printf("ETag error: %v", err)
			return nil
		}
		meta = FileMeta{
			Size:         info.Size(),
			LastModified: info.ModTime().UTC(),
			ETag:         fmt.Sprintf(`%x`, hash),
		}

		c.Lock()
		if c.data[bucket] == nil {
			c.data[bucket] = make(map[string]FileMeta)
		}
		c.data[bucket][relPath] = meta
		c.Unlock()
		return nil
	})

	c.cleanupDeleted(bucket, path)
}

func (c *Cache) cleanupDeleted(bucket, path string) {
	c.Lock()
	defer c.Unlock()

	existing := make(map[string]bool)
	filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(path, filePath)
		if err != nil {
			return nil
		}

		if d.IsDir() {
			relPath += "/"
		}

		existing[relPath] = true
		return nil
	})

	for key := range c.data[bucket] {
		if !existing[key] {
			delete(c.data[bucket], key)
		}
	}
}

func (c *Cache) GetBucket(bucket string) map[string]FileMeta {
	c.RLock()
	defer c.RUnlock()
	return c.data[bucket]
}

func (c *Cache) Get(bucket, key string) (FileMeta, bool) {
	c.RLock()
	defer c.RUnlock()
	meta, exists := c.data[bucket][key]
	return meta, exists
}