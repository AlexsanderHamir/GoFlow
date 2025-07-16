package test

import (
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/AlexsanderHamir/GoFlow/simulator"
	"github.com/stretchr/testify/assert"
)

// CreateConfigsAndSimulator creates a basic config for testing
func createConfigsAndSimulator() (generatorConfig, globalConfig *simulator.StageConfig, sim *simulator.Simulator) {
	sim = simulator.NewSimulator()
	sim.Duration = 2 * time.Second

	generatorConfig = &simulator.StageConfig{
		InputRate:  100 * time.Millisecond,
		RoutineNum: 100,
		BufferSize: 100,
		ItemGenerator: func() any {
			return rand.Intn(100)
		},
	}

	globalConfig = &simulator.StageConfig{
		RoutineNum:         100,
		BufferSize:         100,
		DropOnBackpressure: true,
		WorkerFunc: func(item any) (any, error) {
			item = item.(int) + rand.Intn(100)
			return item, nil
		},
	}

	return generatorConfig, globalConfig, sim
}

func createStages(t *testing.T, sim *simulator.Simulator, generatorConfig, globalConfig *simulator.StageConfig) {
	stage1 := simulator.NewStage("Generator", generatorConfig)
	stage2 := simulator.NewStage("Stage-1", globalConfig)
	stage3 := simulator.NewStage("Stage-2", globalConfig)
	stage4 := simulator.NewStage("Stage-3", globalConfig)
	stage5 := simulator.NewStage("Stage-4", globalConfig)
	stage6 := simulator.NewStage("Stage-5", globalConfig)
	stage7 := simulator.NewStage("Stage-6", globalConfig)
	stage8 := simulator.NewStage("Stage-7", globalConfig)
	stage9 := simulator.NewStage("Stage-8", globalConfig)

	err := sim.AddStage(stage1)
	assert.NoError(t, err)

	err = sim.AddStage(stage2)
	assert.NoError(t, err)

	err = sim.AddStage(stage3)
	assert.NoError(t, err)

	err = sim.AddStage(stage4)
	assert.NoError(t, err)

	err = sim.AddStage(stage5)
	assert.NoError(t, err)

	err = sim.AddStage(stage6)
	assert.NoError(t, err)

	err = sim.AddStage(stage7)
	assert.NoError(t, err)

	err = sim.AddStage(stage8)
	assert.NoError(t, err)

	err = sim.AddStage(stage9)
	assert.NoError(t, err)
}

func checkStageAccountingConsistency(simulator *simulator.Simulator, t *testing.T) {
	var lastStageOutput uint64
	var lastStageName string

	for _, stage := range simulator.GetStages() {
		stats := stage.GetMetrics().GetStats()

		if stage.GetIsGenerator() {
			output := stats["output_items"].(uint64)
			generated := stats["generated_items"].(uint64)
			dropped := stats["dropped_items"].(uint64)
			lastStageOutput = output
			lastStageName = stage.Name

			totalProcessed := output + dropped

			if totalProcessed != generated {
				t.Fatalf("Generator Inconsistency: output(%d) + dropped(%d) = %d, different than generated: %d \n error priority: %s", output, dropped, totalProcessed, generated, priorityMedium)
			}
			continue
		}

		currentProcessed := stats["processed_items"].(uint64)
		currentOutput := stats["output_items"].(uint64)
		currentDropped := stats["dropped_items"].(uint64)
		currentStageTotalReceived := currentOutput + currentDropped

		if currentProcessed > currentStageTotalReceived {
			t.Fatalf("processed_items(%d), can't be bigger than output + dropped(%d) \n error priority: %s", currentProcessed, currentStageTotalReceived, priorityHigh)
		}

		if lastStageOutput != currentStageTotalReceived {
			t.Fatalf("%s output %d does not match current %s total %d \n missing count: %d \n error priority: %s", lastStageName, lastStageOutput, stage.Name, currentStageTotalReceived, lastStageOutput-currentStageTotalReceived, priorityLow)
		}

		lastStageOutput = currentOutput
		lastStageName = stage.Name

		log.Printf("Stage %s: output=%d", stage.Name, currentOutput)
	}
}
