package simulator

import (
	"fmt"
	"strings"

	"github.com/AlexsanderHamir/IdleSpy/tracker"
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
		ProcessedItems: stage.metrics.processedItems,
		OutputItems:    stage.metrics.outputItems,
		Throughput:     stats["throughput"].(float64),
		DroppedItems:   stage.metrics.droppedItems,
		DropRate:       stats["drop_rate"].(float64),
		GeneratedItems: stage.metrics.generatedItems,
		isGenerator:    stage.isGenerator,
		IsFinal:        stage.isFinal,
	}
}

// computeDiffs calculates the different between one stage and the other.
func computeDiffs(prev, curr *stageStats) (procDiffStr, thruDiffStr string) {
	procDiffStr = ""
	thruDiffStr = ""
	if prev == nil {
		return "", ""
	}

	// Skip Generator and DummyStage
	if curr.isGenerator || curr.IsFinal ||
		prev.isGenerator {
		return "", ""
	}

	if prev.Throughput > 0 {
		diff := ((curr.Throughput - prev.Throughput) / prev.Throughput) * 100
		thruDiffStr = fmt.Sprintf("%+.2f", diff)
	}
	if prev.ProcessedItems > 0 {
		diff := ((float64(curr.ProcessedItems) - float64(prev.ProcessedItems)) / float64(prev.ProcessedItems)) * 100
		procDiffStr = fmt.Sprintf("%+.2f", diff)
	}

	return procDiffStr, thruDiffStr
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

func (s *Simulator) writeDotHeader(b *strings.Builder) {
	b.WriteString("digraph Pipeline {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box, style=filled, fontname=\"Arial\", fontsize=10];\n")
	b.WriteString("  edge [fontname=\"Arial\", fontsize=8];\n\n")
}

func (s *Simulator) writeDotNodes(b *strings.Builder) error {
	var prevStats *stageStats
	stages := s.GetStages()
	first, last := 0, len(stages)-1

	for i, stage := range stages {
		currentStats := collectStageStats(stage)
		procDiffStr, thruDiffStr := computeDiffs(prevStats, &currentStats)
		prevStats = &currentStats

		nodeColor := s.getNodeColor(stage)
		label := s.formatNodeLabel(stage, &currentStats, procDiffStr, thruDiffStr)

		fmt.Fprintf(b, "  stage_%d [label=%s, style=filled, fillcolor=%s];\n",
			i, label, nodeColor)

		if i != first && i != last {
			if err := s.writeGoroutineStats(stage); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Simulator) getNodeColor(stage *Stage) string {
	switch {
	case stage.isGenerator:
		return "lightgreen"
	case stage.isFinal:
		return "lightcoral"
	default:
		return "lightblue"
	}
}

func (s *Simulator) formatNodeLabel(stage *Stage, stats *stageStats, procDiff, thruDiff string) string {
	return fmt.Sprintf(`"%s\nRoutines: %d\nBuffer: %d\nProcessed: %d (%s)\nDroppedItems: %d\nOutput: %d\nThroughput: %.2f (%s)"`,
		stage.Name,
		stage.Config.RoutineNum,
		stage.Config.BufferSize,
		stats.ProcessedItems, procDiff,
		stats.DroppedItems,
		stats.OutputItems,
		stats.Throughput, thruDiff,
	)
}

func (s *Simulator) writeGoroutineStats(stage *Stage) error {
	goroutineStats := stage.gm.GetAllStats()
	err := tracker.WriteBlockedTimeHistogramDot(goroutineStats, stage.Name)
	if err != nil {
		return fmt.Errorf("goroutine tracker failed: %w", err)
	}
	return nil
}

func (s *Simulator) writeDotEdges(b *strings.Builder) {
	b.WriteString("\n")
	stages := s.GetStages()
	for i := 0; i < len(stages)-1; i++ {
		fmt.Fprintf(b, "  stage_%d -> stage_%d;\n", i, i+1)
	}
}

func (s *Simulator) writeDotFooter(b *strings.Builder) {
	b.WriteString("}\n")
}
