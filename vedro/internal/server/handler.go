package server

import (
	"encoding/xml"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"vedro/config"
	"vedro/internal/storage"
	"vedro/internal/utils"
)

type Content struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	Size         int64  `xml:"Size"`
	ETag         string `xml:"ETag"`
}

type ListResult struct {
	XMLName  xml.Name  `xml:"ListBucketResult"`
	Xmlns    string    `xml:"xmlns,attr"`
	Name     string    `xml:"Name"`
	Contents []Content `xml:"Contents"`
}

func NewHandler(root string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", rootHandler(root))

	var handler http.Handler = mux
	if config.EnableRecover {
		handler = recoverMiddleware(handler)
	}
	if config.EnableLogging {
		handler = loggingMiddleware(handler)
	}
	return handler
}

func rootHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			HandleError(w, r, http.StatusMethodNotAllowed,
				"MethodNotAllowed", "The specified method is not allowed")
			return
		}

		if strings.Trim(r.URL.Path, "/") == "" {
			HandleError(w, r, http.StatusBadRequest,
				"InvalidRequest", "Empty request path")
			return
		}

		path := strings.Trim(r.URL.Path, "/")
		parts := strings.SplitN(path, "/", 2)
		switch len(parts) {
		case 1:
			listBucketHandler(root)(w, r)
		case 2:
			getObjectHandler(root)(w, r)
		default:
			http.NotFound(w, r)
		}
	}
}

func listBucketHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucket := strings.TrimPrefix(r.URL.Path, "/")
		bucketPath := filepath.Join(root, bucket)

		if _, err := os.Stat(bucketPath); os.IsNotExist(err) {
			HandleError(w, r, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist")
			return
		}

		var contents []Content

		err := filepath.WalkDir(bucketPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == bucketPath {
				return nil
			}

			relPath, err := filepath.Rel(bucketPath, path)
			if err != nil {
				return err
			}

			if d.IsDir() {
				relPath += "/"
			} else {
				info, err := d.Info()
				if err != nil {
					return err
				}

				etag, err := storage.ComputeETag(path, false)
				if err != nil {
					return err
				}

				contents = append(contents, Content{
					Key:          relPath,
					LastModified: info.ModTime().UTC().Format(time.RFC3339),
					Size:         info.Size(),
					ETag:         etag,
				})
			}
			return nil
		})

		if err != nil {
			HandleError(w, r, http.StatusInternalServerError, "InternalError", "Failed to list bucket contents")
			return
		}

		sort.Slice(contents, func(i, j int) bool {
			return contents[i].Key < contents[j].Key
		})

		result := ListResult{
			Xmlns:    "http://s3.amazonaws.com/doc/2006-03-01/",
			Name:     bucket,
			Contents: contents,
		}

		w.Header().Set("Content-Type", "application/xml")
		enc := xml.NewEncoder(w)
		enc.Indent("", "  ")
		if err := enc.Encode(result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func getObjectHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}

		bucket, key := parts[0], parts[1]
		fullPath := utils.GetFullPath(root, bucket, key)

		if !utils.ValidatePath(root, bucket, key) {
			HandleError(w, r, http.StatusBadRequest, "InvalidRequest", "Malformed request path")
			return
		}

		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			HandleError(w, r, http.StatusNotFound, "NoSuchKey", "The specified key does not exist")
			return
		} else if err != nil {
			HandleError(w, r, http.StatusInternalServerError, "InternalError", "Error accessing file")
			return
		}

		etag, err := storage.ComputeETag(fullPath, false)
		if err != nil {
			HandleError(w, r, http.StatusInternalServerError, "InternalError", "Failed to compute ETag")
			return
		}

		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		if modSince := r.Header.Get("If-Modified-Since"); modSince != "" {
			if t, err := time.Parse(http.TimeFormat, modSince); err == nil {
				if info.ModTime().Before(t.Add(time.Second)) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}

		w.Header().Set("ETag", etag)
		w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
		http.ServeFile(w, r, fullPath)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
