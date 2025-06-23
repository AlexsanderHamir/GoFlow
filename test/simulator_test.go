package test

import (
	"fmt"
	"log"
	"testing"
)

const (
	SimulationIterations = 10
)

func TestSimulatorGeneratedStatsConsistency(t *testing.T) {
	for i := range SimulationIterations {
		t.Run(fmt.Sprintf("Iteration_%d_Regular", i+1), func(t *testing.T) {
			generatorConfig, globalConfig, simulator := CreateConfigsAndSimulator()
			CreateStages(simulator, generatorConfig, globalConfig)

			if err := simulator.Start(); err != nil {
				log.Fatalf("Failed to start simulator in iteration %d: %v", i+1, err)
			}

			<-simulator.Done()

			CheckStageAccountingConsistency(simulator, t)

			simulator.Stop()
		})

		t.Run(fmt.Sprintf("Iteration_%d_Burst", i+1), func(t *testing.T) {
			generatorConfig, globalConfig, simulator := CreateConfigsAndSimulatorBurst()
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
