package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var primaryEndpoint string
var secondaryEndpoint string
var listenAddr = ":9000"

func fetchJSON(url string) map[string]interface{} {
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("error fetching %s: %v", url, err)
		return map[string]interface{}{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading %s: %v", url, err)
		return map[string]interface{}{}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("error parsing %s: %v", url, err)
		return map[string]interface{}{}
	}
	return data
}

func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	for key, val := range src {
		if existing, ok := dst[key]; ok {
			// if both maps, merge recursively
			mapA, okA := existing.(map[string]interface{})
			mapB, okB := val.(map[string]interface{})
			if okA && okB {
				dst[key] = deepMerge(mapA, mapB)
				continue
			}
			// if both arrays, append
			arrA, okArrA := existing.([]interface{})
			arrB, okArrB := val.([]interface{})
			if okArrA && okArrB {
				dst[key] = append(arrA, arrB...)
				continue
			}
		}
		dst[key] = val
	}
	return dst
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data1 := fetchJSON(primaryEndpoint)
		data2 := fetchJSON(secondaryEndpoint)
		// Start with secondary, merge primary on top (primary overrides on conflicts)
		merged := data2
		merged = deepMerge(merged, data1)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(merged); err != nil {
			log.Printf("error encoding merged data: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// For non-GET (write operations), proxy to secondary endpoint
	proxyReq, err := http.NewRequest(r.Method, secondaryEndpoint, io.NopCloser(r.Body))
	if err != nil {
		log.Printf("error creating proxy request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Copy headers (skip Hop-by-Hop headers if needed, but simple copy for now)
	for k, vv := range r.Header {
		for _, v := range vv {
			proxyReq.Header.Add(k, v)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("error proxying to secondary %s: %v", secondaryEndpoint, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("error copying proxy response body: %v", err)
	}
}

func main() {
	if env := os.Getenv("MERGE_ENDPOINTS"); env != "" {
		parts := strings.Split(env, ",")
		if len(parts) < 2 {
			log.Fatal("MERGE_ENDPOINTS must provide at least two comma-separated endpoints (primary,secondary)")
		}
		primaryEndpoint = strings.TrimSpace(parts[0])
		secondaryEndpoint = strings.TrimSpace(parts[1])
	} else {
		log.Fatal("MERGE_ENDPOINTS environment variable is required")
	}

	if addr := os.Getenv("MERGE_LISTEN"); addr != "" {
		listenAddr = addr
	}

	http.HandleFunc("/traefik-merged", handler)
	log.Printf("Traefik merge shim running on %s (primary: read-only %s, secondary: read-write %s)", listenAddr, primaryEndpoint, secondaryEndpoint)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}