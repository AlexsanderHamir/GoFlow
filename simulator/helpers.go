package simulator

import (
	"fmt"
	"strings"
)

type stageStats struct {
	StageName      string
	ProcessedItems uint64
	OutputItems    uint64
	Throughput     float64
	DroppedItems   uint64
	DropRate       float64
	GeneratedItems uint64
	ThruDiffPct    float64
	ProcDiffPct    float64
	isGenerator    bool
	IsFinal        bool
}

func collectStageStats(stage *Stage) stageStats {
	stats := stage.GetMetrics().GetStats()
	return stageStats{
		StageName:      stage.Name,
		ProcessedItems: getIntMetric(stats, "processed_items"),
		OutputItems:    getIntMetric(stats, "output_items"),
		Throughput:     getFloatMetric(stats, "throughput"),
		DroppedItems:   getIntMetric(stats, "dropped_items"),
		DropRate:       getFloatMetric(stats, "drop_rate") * 100,
		GeneratedItems: getIntMetric(stats, "generated_items"),
		isGenerator:    stage.isGenerator,
		IsFinal:        stage.isFinal,
	}
}

func getIntMetric(stats map[string]any, key string) uint64 {
	if val, ok := stats[key]; ok && val != nil {
		if intVal, ok := val.(uint64); ok {
			return intVal
		}
	}
	return 0
}

func getFloatMetric(stats map[string]any, key string) float64 {
	if val, ok := stats[key]; ok && val != nil {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return 0.0
}

func computeDiffs(prev, curr *stageStats) (procDiffStr, thruDiffStr string) {
	procDiffStr = ""
	thruDiffStr = ""
	if prev == nil {
		return
	}

	// Skip Generator and DummyStage
	if curr.isGenerator || curr.IsFinal ||
		prev.isGenerator || prev.IsFinal {
		return
	}

	if prev.Throughput > 0 {
		diff := ((curr.Throughput - prev.Throughput) / prev.Throughput) * 100
		thruDiffStr = fmt.Sprintf("%+.2f", diff)
	}
	if prev.ProcessedItems > 0 {
		diff := ((float64(curr.ProcessedItems) - float64(prev.ProcessedItems)) / float64(prev.ProcessedItems)) * 100
		procDiffStr = fmt.Sprintf("%+.2f", diff)
	}

	return
}

func printHeader() {
	fmt.Printf("\n%-20s %12s %12s %12s %12s %12s %12s %12s\n",
		"Stage", "Processed", "Output", "Throughput", "Dropped", "Drop Rate %", "Proc Δ%", "Thru Δ%")
	fmt.Println(strings.Repeat("-", 114))
}

func printStageRow(stat *stageStats, procDiff, thruDiff string) {
	fmt.Printf("%-20s %12d %12d %12.2f %12d %12.2f %12s %12s\n",
		stat.StageName,
		stat.ProcessedItems,
		stat.OutputItems,
		stat.Throughput,
		stat.DroppedItems,
		stat.DropRate,
		procDiff,
		thruDiff,
	)
}
