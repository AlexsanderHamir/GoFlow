package simulator

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
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
