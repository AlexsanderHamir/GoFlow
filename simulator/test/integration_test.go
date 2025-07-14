package test

import (
	"fmt"
	"log"
	"testing"

	lib "github.com/AlexsanderHamir/GoFlow/simulator"
)

const (
	SimulationIterations = 5
)

func TestStatsConsistency(t *testing.T) {
	for i := range SimulationIterations {
		t.Run(fmt.Sprintf("Iteration_%d_Regular", i+1), func(t *testing.T) {
			generatorConfig, globalConfig, simulator := createConfigsAndSimulator()
			createStages(t, simulator, generatorConfig, globalConfig)

			if err := simulator.Start(lib.Nothing); err != nil {
				log.Fatalf("Failed to start simulator in iteration %d: %v", i+1, err)
			}

			checkStageAccountingConsistency(simulator, t)
		})
	}
}
