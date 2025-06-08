package test

import (
	"context"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/AlexsanderHamir/GoFlow/pkg/simulator"
)

func CreateConfigsAndSimulator() (*simulator.StageConfig, *simulator.StageConfig, *simulator.Simulator) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create simulator
	sim := simulator.NewSimulator(ctx, cancel)
	sim.Duration = 10 * time.Second

	// Create configuration for stages
	generatorConfig := &simulator.StageConfig{
		InputRate:   100 * time.Millisecond,
		RoutineNum:  100,
		BufferSize:  100,
		IsGenerator: true,
		ItemGenerator: func() any {
			return rand.Intn(100)
		},
		Ctx: ctx,
	}

	globalConfig := &simulator.StageConfig{
		RoutineNum: 100,
		BufferSize: 100,
		WorkerFunc: func(item any) (any, error) {
			item = item.(int) + rand.Intn(100)
			return item, nil
		},
		Ctx: ctx,
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
	var lastStageAccountingObjects uint64
	log.Printf("Starting stage accounting consistency check")
	for i, stage := range simulator.Stages {
		stats := stage.GetMetrics().GetStats()
		if stage.Config.IsGenerator || stage.IsFinal {
			log.Printf("Skipping stage %s (Generator: %v, Final: %v)", stage.Name, stage.Config.IsGenerator, stage.IsFinal)
			continue
		}

		processed := stats["processed_items"].(uint64)
		dropped := stats["dropped_items"].(uint64)
		total := processed + dropped

		if i == 1 {
			lastStageAccountingObjects = total
			log.Printf("Stage %s (first non-generator): processed=%d, dropped=%d, total=%d",
				stage.Name, processed, dropped, total)
			continue
		}

		currentStageAccountingObjects := total
		log.Printf("Stage %s: processed=%d, dropped=%d, total=%d, expected=%d",
			stage.Name, processed, dropped, currentStageAccountingObjects, lastStageAccountingObjects)

		if currentStageAccountingObjects != lastStageAccountingObjects {
			log.Printf("❌ Accounting mismatch detected in stage %s", stage.Name)
			t.Fatalf("Stage %s processed %d objects, expected %d", stage.Name, currentStageAccountingObjects, lastStageAccountingObjects)
			simulator.PrintStats()
		} else {
			log.Printf("✓ Stage %s accounting consistent", stage.Name)
		}
	}
	log.Printf("Completed stage accounting consistency check")
}
