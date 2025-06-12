package test

import (
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/AlexsanderHamir/GoFlow/pkg/simulator"
)

func CreateConfigsAndSimulator() (*simulator.StageConfig, *simulator.StageConfig, *simulator.Simulator) {
	sim := simulator.NewSimulator()
	sim.Duration = 10 * time.Second

	generatorConfig := &simulator.StageConfig{
		InputRate:   100 * time.Millisecond,
		RoutineNum:  100,
		BufferSize:  100,
		IsGenerator: true,
		ItemGenerator: func() any {
			return rand.Intn(100)
		},
	}

	globalConfig := &simulator.StageConfig{
		RoutineNum: 100,
		BufferSize: 100,
		WorkerFunc: func(item any) (any, error) {
			item = item.(int) + rand.Intn(100)
			return item, nil
		},
	}

	return generatorConfig, globalConfig, sim
}

func CreateStages(sim *simulator.Simulator, generatorConfig *simulator.StageConfig, globalConfig *simulator.StageConfig) {
	stage1 := simulator.NewStage("Generator", generatorConfig)
	stage2 := simulator.NewStage("Stage-1", globalConfig)
	stage3 := simulator.NewStage("Stage-2", globalConfig)
	stage4 := simulator.NewStage("Stage-3", globalConfig)
	stage5 := simulator.NewStage("Stage-4", globalConfig)
	stage6 := simulator.NewStage("Stage-5", globalConfig)
	stage7 := simulator.NewStage("Stage-6", globalConfig)
	stage8 := simulator.NewStage("Stage-7", globalConfig)
	stage9 := simulator.NewStage("Stage-8", globalConfig)

	sim.AddStage(stage1)
	sim.AddStage(stage2)
	sim.AddStage(stage3)
	sim.AddStage(stage4)
	sim.AddStage(stage5)
	sim.AddStage(stage6)
	sim.AddStage(stage7)
	sim.AddStage(stage8)
	sim.AddStage(stage9)
}

func CheckStageAccountingConsistency(simulator *simulator.Simulator, t *testing.T) {
	var lastStageOutput uint64
	var lastStageName string

	for _, stage := range simulator.Stages {
		stats := stage.GetMetrics().GetStats()

		if stage.Config.IsGenerator {
			output := stats["output_items"].(uint64)
			lastStageOutput = output
			lastStageName = stage.Name
			continue
		}

		currentOutput := stats["output_items"].(uint64)
		currentDropped := stats["dropped_items"].(uint64)

		total := currentOutput + currentDropped
		if lastStageOutput != total {
			t.Fatalf("%s last output %d does not match current stage %s total %d", lastStageName, lastStageOutput, stage.Name, total)
		}

		lastStageOutput = currentOutput
		lastStageName = stage.Name

		log.Printf("Stage %s: output=%d", stage.Name, currentOutput)
	}
}

func CreateConfigsAndSimulatorBurst() (*simulator.StageConfig, *simulator.StageConfig, *simulator.Simulator) {
	sim := simulator.NewSimulator()
	sim.Duration = 10 * time.Second

	generatorConfig := &simulator.StageConfig{
		InputRate:   100 * time.Millisecond,
		RoutineNum:  100,
		BufferSize:  100,
		IsGenerator: true,
		ItemGenerator: func() any {
			return rand.Intn(100)
		},
		InputBurst: func() []any {
			sliceLen := rand.Intn(10) + 1
			result := make([]any, sliceLen)
			for i := range sliceLen {
				result[i] = rand.Intn(100)
			}
			return result
		},
		BurstCountTotal: 1000,
		BurstInterval:   100 * time.Millisecond,
	}

	globalConfig := &simulator.StageConfig{
		RoutineNum: 100,
		BufferSize: 100,
		WorkerFunc: func(item any) (any, error) {
			item = item.(int) + rand.Intn(100)
			return item, nil
		},
	}

	return generatorConfig, globalConfig, sim
}
