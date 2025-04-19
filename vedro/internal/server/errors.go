package server

import (
    "encoding/xml"
    "fmt"
    "net/http"
	"strings"
)

type ErrorResponse struct {
    XMLName xml.Name `xml:"Error"`
    Code    string   `xml:"Code"`
    Message string   `xml:"Message"`
    Bucket  string   `xml:"BucketName,omitempty"`
    Key     string   `xml:"Key,omitempty"`
}

func HandleError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
    w.Header().Set("Content-Type", "application/xml")
    w.WriteHeader(status)

    resp := ErrorResponse{
        Code:    code,
        Message: message,
    }

    path := strings.Trim(r.URL.Path, "/")
    parts := strings.SplitN(path, "/", 2)
    if len(parts) > 0 {
        resp.Bucket = parts[0]
    }
    if len(parts) > 1 {
        resp.Key = parts[1]
    }

    if _, err := fmt.Fprint(w, xml.Header); err != nil {
        return
    }
    enc := xml.NewEncoder(w)
    enc.Indent("", "  ")
    if err := enc.Encode(resp); err != nil {
        http.Error(w, "Failed to generate error response", http.StatusInternalServerError)
    }
}