package simulator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// DirectoryEntry represents a file or directory entry in the response
type DirectoryEntry struct {
	Name string `json:"name"`
	Size int64  `json:"size,omitempty"`
	Path string `json:"path"`
}

// listDirectoryAPIHandler handles API requests to list directory contents
func listDirectoryAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get directory path from query parameter
	dirPath := r.URL.Query().Get("dir")
	if dirPath == "" {
		http.Error(w, "Directory path is required", http.StatusBadRequest)
		return
	}

	// Get the absolute path of the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		http.Error(w, "Failed to get current directory", http.StatusInternalServerError)
		return
	}

	// Create the absolute path for the requested directory
	absPath := filepath.Join(cwd, "static", dirPath)

	// Verify the path is within the static directory
	if !strings.HasPrefix(absPath, filepath.Join(cwd, "static")) {
		http.Error(w, "Access denied: directory must be within static folder", http.StatusForbidden)
		return
	}

	// Read the directory contents
	entries, err := os.ReadDir(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Directory not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Create response with file information
	var response []DirectoryEntry
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't get info for
		}

		// Create relative path for the entry
		relPath := filepath.Join(dirPath, entry.Name())
		// Normalize path separators for web
		relPath = strings.ReplaceAll(relPath, string(os.PathSeparator), "/")

		response = append(response, DirectoryEntry{
			Name: entry.Name(),
			Size: info.Size(),
			Path: relPath,
		})
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

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

	// Create a file server handler for static files
	fs := http.FileServer(http.Dir(staticDir))

	// Set up routes
	mux := http.NewServeMux()

	// API endpoint for listing directory contents
	mux.HandleFunc("/api/list", listDirectoryAPIHandler)

	// Serve static files for all other routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})

	// Apply CORS middleware to the mux
	handler := corsMiddleware(mux)

	// Start the server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on http://localhost%s", addr)
	log.Printf("Serving files from: %s", staticDir)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// findGoFlowRoot finds the GoFlow root directory by looking for go.mod
func findGoFlowRoot() (string, error) {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Keep going up until we find go.mod or hit root
	for {
		// Check if go.mod exists in current directory
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Read go.mod to verify it's GoFlow
			content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			if err != nil {
				return "", fmt.Errorf("failed to read go.mod: %w", err)
			}
			// Check if this is GoFlow's go.mod
			if bytes.Contains(content, []byte("github.com/AlexsanderHamir/GoFlow")) {
				return dir, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've hit the root directory
			return "", fmt.Errorf("could not find GoFlow root directory")
		}
		dir = parent
	}
}

// InitFrontend initializes the frontend by starting the development server.
// It runs 'npm run dev' in the UI directory to start the Vite development server.
func StartFrontend() error {
	// Find the GoFlow root directory
	goFlowRoot, err := findGoFlowRoot()
	if err != nil {
		return fmt.Errorf("failed to find GoFlow root: %w", err)
	}

	// Create the command to run npm run dev
	cmd := exec.Command("npm", "run", "dev")
	cmd.Dir = filepath.Join(goFlowRoot, "UI")

	// Set up pipes for stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command in the background
	if err := cmd.Start(); err != nil {
		return err
	}

	// Detach the process so it continues running after the parent process exits
	if err := cmd.Process.Release(); err != nil {
		return err
	}

	return nil
}
