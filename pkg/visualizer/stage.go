package visualizer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StageStats represents the statistics for a single stage
type StageStats struct {
	StageName      string  `json:"stage_name"`
	ProcessedItems int     `json:"processed_items"`
	OutputItems    int     `json:"output_items"`
	Throughput     float64 `json:"throughput"`
	DroppedItems   int     `json:"dropped_items"`
	DropRate       float64 `json:"drop_rate"`
	ThruDiffPct    float64 `json:"-"` // Throughput difference percentage from previous stage
	ProcDiffPct    float64 `json:"-"` // Processed items difference percentage from previous stage
}

// ReadStageStats reads all stage statistics files from the given directory
func ReadStageStats(stagesDir string) ([]StageStats, error) {
	var stats []StageStats

	// Read all files in the directory
	files, err := os.ReadDir(stagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read stages directory: %w", err)
	}

	// Process each file
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), "_stats.json") {
			continue
		}

		// Read and parse the JSON file
		data, err := os.ReadFile(filepath.Join(stagesDir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file.Name(), err)
		}

		var stageStat StageStats
		if err := json.Unmarshal(data, &stageStat); err != nil {
			return nil, fmt.Errorf("failed to parse JSON from %s: %w", file.Name(), err)
		}

		stats = append(stats, stageStat)
	}

	return stats, nil
}

// VisualizeStageStats creates a table visualization of stage statistics
func VisualizeStageStats(stagesDir string) error {
	stats, err := ReadStageStats(stagesDir)
	if err != nil {
		return fmt.Errorf("failed to read stage stats: %w", err)
	}

	// Calculate throughput and processed items differences
	for i := 1; i < len(stats); i++ {
		// Skip calculation for Generator and DummyStage
		if stats[i].StageName == "Generator" || stats[i].StageName == "DummyStage" ||
			stats[i-1].StageName == "Generator" || stats[i-1].StageName == "DummyStage" {
			continue
		}
		if stats[i-1].Throughput > 0 {
			stats[i].ThruDiffPct = ((stats[i].Throughput - stats[i-1].Throughput) / stats[i-1].Throughput) * 100
		}
		if stats[i-1].ProcessedItems > 0 {
			stats[i].ProcDiffPct = ((float64(stats[i].ProcessedItems) - float64(stats[i-1].ProcessedItems)) / float64(stats[i-1].ProcessedItems)) * 100
		}
	}

	// Print header
	fmt.Printf("\n%-20s %12s %12s %12s %12s %12s %12s %12s\n",
		"Stage", "Processed", "Output", "Throughput", "Dropped", "Drop Rate %", "Proc Δ%", "Thru Δ%")
	fmt.Println(strings.Repeat("-", 114))

	// Print each row
	for _, stat := range stats {
		thruDiffStr := "-"
		if stat.ThruDiffPct != 0 {
			thruDiffStr = fmt.Sprintf("%+.2f", stat.ThruDiffPct)
		}
		procDiffStr := "-"
		if stat.ProcDiffPct != 0 {
			procDiffStr = fmt.Sprintf("%+.2f", stat.ProcDiffPct)
		}
		fmt.Printf("%-20s %12d %12d %12.2f %12d %12.2f %12s %12s\n",
			stat.StageName,
			stat.ProcessedItems,
			stat.OutputItems,
			stat.Throughput,
			stat.DroppedItems,
			stat.DropRate,
			procDiffStr,
			thruDiffStr)
	}
	fmt.Println()

	return nil
}
