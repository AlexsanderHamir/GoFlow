package simulator

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// StartStaticServer starts an HTTP server to serve static files from the static directory
func StartStaticServer(port int) {
	// Get the absolute path of the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	// Create the absolute path for the static directory
	staticDir := filepath.Join(cwd, "static")

	// Ensure the static directory exists
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Fatalf("Static directory %s does not exist", staticDir)
	}

	// Create a file server handler
	fs := http.FileServer(http.Dir(staticDir))

	// Create a custom handler that adds logging
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		fs.ServeHTTP(w, r)
	})

	// Start the server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on http://localhost%s", addr)
	log.Printf("Serving files from: %s", staticDir)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
