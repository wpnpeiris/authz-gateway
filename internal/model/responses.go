package model

import (
	"log"
	"net/http"
	"strconv"
)

type mimeType string

const (
	mimeNone mimeType = ""
)

// WriteEmptyResponse writes only headers and the given status code.
func WriteEmptyResponse(w http.ResponseWriter, r *http.Request, statusCode int) {
	WriteResponse(w, r, statusCode, []byte{}, mimeNone)
}

// WriteResponse writes headers, status code, and optional body, flushing the
// response when done. Body logging is minimized to avoid leaking data.
func WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, response []byte, mType mimeType) {
	if response != nil {
		w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	}
	if mType != mimeNone {
		w.Header().Set("Content-Type", string(mType))
	}
	w.WriteHeader(statusCode)
	if response != nil {
		log.Printf("status %d %s len=%d", statusCode, mType, len(response))
		_, err := w.Write(response)
		if err != nil {
			log.Printf("Error writing the response, %s", err)
			return
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}
