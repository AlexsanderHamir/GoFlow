package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/AlexsanderHamir/GoFlow/pkg/visualizer"
)

func main() {
	stagesDirFlag := flag.String("stages-dir", "", "Path to the directory containing stage statistics files (*_stats.json)")
	chartFlag := flag.String("chart", "score", "Type of chart to generate (see descriptions below)")
	goroutineFileFlag := flag.String("goroutine-file", "", "Path to the goroutine information file (goroutine_info_*.json)")

	flag.Parse()

	// Validate flags based on visualization type
	if *stagesDirFlag == "" && *goroutineFileFlag == "" {
		if *stagesDirFlag == "" {
			fmt.Println("Error: -stages-dir flag is required for stages visualization")
		}
		if *goroutineFileFlag == "" {
			fmt.Println("Error: -goroutine-file flag is required for goroutines visualization")
		}
		flag.Usage()
		os.Exit(1)
	}

	if *stagesDirFlag != "" {
		if err := visualizer.VisualizeStageStats(*stagesDirFlag); err != nil {
			fmt.Printf("Error visualizing stages: %v\n", err)
			os.Exit(1)
		}
	} else if *goroutineFileFlag != "" {
		if err := visualizer.VisualizeStageGoroutines(*goroutineFileFlag, *chartFlag); err != nil {
			fmt.Printf("Error visualizing goroutines: %v\n", err)
			os.Exit(1)
		}
	}
}
