package test

import (
	"fmt"
	"log"
	"testing"
)

const (
	SimulationIterations = 3
)

func TestStatsConsistency(t *testing.T) {
	for i := range SimulationIterations {
		t.Run(fmt.Sprintf("Iteration_%d_Regular", i+1), func(t *testing.T) {
			generatorConfig, globalConfig, simulator := CreateConfigsAndSimulator()
			CreateStages(simulator, generatorConfig, globalConfig)

			if err := simulator.Start(false); err != nil {
				log.Fatalf("Failed to start simulator in iteration %d: %v", i+1, err)
			}

			CheckStageAccountingConsistency(simulator, t)
		})
	}
}
