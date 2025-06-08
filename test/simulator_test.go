package test

import (
	"fmt"
	"log"
	"testing"
)

const (
	// Number of times to run the simulation
	SimulationIterations = 10
)

func TestSimulator(t *testing.T) {
	for i := range SimulationIterations {
		t.Run(fmt.Sprintf("Iteration_%d", i+1), func(t *testing.T) {
			generatorConfig, globalConfig, simulator := CreateConfigsAndSimulator()
			CreateStages(simulator, generatorConfig, globalConfig)

			if err := simulator.Start(); err != nil {
				log.Fatalf("Failed to start simulator in iteration %d: %v", i+1, err)
			}

			<-simulator.Done()

			CheckStageAccountingConsistency(simulator, t)

			simulator.Stop()
		})
	}
}
