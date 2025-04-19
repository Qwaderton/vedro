package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"
	"sort"
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

func NewHandler(cache *storage.Cache) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", rootHandler(cache))

	var handler http.Handler = mux
	if config.EnableRecover {
		handler = recoverMiddleware(handler)
	}
	if config.EnableLogging {
		handler = loggingMiddleware(handler)
	}
	return handler
}

func rootHandler(cache *storage.Cache) http.HandlerFunc {
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
			listBucketHandler(cache)(w, r)
		case 2:
			getObjectHandler(cache)(w, r)
		default:
			http.NotFound(w, r)
		}
	}
}

func listBucketHandler(cache *storage.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucket := strings.TrimPrefix(r.URL.Path, "/")
		files := cache.GetBucket(bucket)
        if files == nil {
            HandleError(w, r, http.StatusNotFound,
                "NoSuchBucket", "The specified bucket does not exist")
            return
        }
		keys := make([]string, 0, len(files))
		for k := range files {
            keys = append(keys, k)
        }
        sort.Strings(keys)

		var contents []Content
        for _, key := range keys {
            meta := files[key]
            contents = append(contents, Content{
                Key:          key,
                LastModified: meta.LastModified.Format(time.RFC3339),
                Size:         meta.Size,
                ETag:         meta.ETag,
            })
        }

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

func getObjectHandler(cache *storage.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}

		bucket, key := parts[0], parts[1]
		if valid := utils.ValidatePath(config.RootPath, bucket, key); !valid {
            HandleError(w, r, http.StatusBadRequest,
                "InvalidRequest", "Malformed request path")
            return
        }

		meta, exists := cache.Get(bucket, key)
		if !exists {
            HandleError(w, r, http.StatusNotFound,
                "NoSuchKey", "The specified key does not exist")
            return
        }

		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, meta.ETag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		if modSince := r.Header.Get("If-Modified-Since"); modSince != "" {
			if t, err := time.Parse(http.TimeFormat, modSince); err == nil {
				if meta.LastModified.Before(t.Add(time.Second)) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}

		w.Header().Set("ETag", meta.ETag)
		w.Header().Set("Last-Modified", meta.LastModified.Format(http.TimeFormat))
		http.ServeFile(w, r, utils.GetFullPath(config.RootPath, bucket, key))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}