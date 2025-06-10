package websocket

import (
	"os"
	"os/exec"
)

// InitFrontend initializes the frontend by starting the development server.
// It runs 'npm run dev' in the UI directory to start the Vite development server.
func InitFrontend() error {
	// Create the command to run npm run dev
	cmd := exec.Command("npm", "run", "dev")
	cmd.Dir = "/Users/alexsandergomes/Documents/GoFlow/UI"

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
