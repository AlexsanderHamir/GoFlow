package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/AlexsanderHamir/GoFlow/pkg/simulator"
)

func main() {
	// Create a context without timeout since simulator duration controls execution time
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create simulator
	sim := simulator.NewSimulator(ctx, cancel)
	sim.MaxGeneratedItems = 10000

	// Create configuration for stages
	generatorConfig := &simulator.StageConfig{
		InputRate:   100 * time.Millisecond,
		RoutineNum:  100,
		BufferSize:  1000,
		IsGenerator: true,
		ItemGenerator: func() any {
			return rand.Intn(100)
		},
		Ctx: ctx,
	}

	globalConfig := &simulator.StageConfig{
		RoutineNum: 100,
		BufferSize: 1000,
		Ctx:        ctx,
	}

	stage1 := simulator.NewStage("Generator", generatorConfig)
	stage2 := simulator.NewStage("Worker-1", globalConfig)
	stage2.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(10 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage3 := simulator.NewStage("Worker-2", globalConfig)
	stage3.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(40 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage4 := simulator.NewStage("Worker-3", globalConfig)
	stage4.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(100 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage5 := simulator.NewStage("Worker-4", globalConfig)
	stage5.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(150 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage6 := simulator.NewStage("Worker-5", globalConfig)
	stage6.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(200 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage7 := simulator.NewStage("Worker-6", globalConfig)
	stage7.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(250 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage8 := simulator.NewStage("Worker-7", globalConfig)
	stage8.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(300 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage9 := simulator.NewStage("Worker-8", globalConfig)
	stage9.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(350 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	sim.AddStage(stage1)
	sim.AddStage(stage2)
	sim.AddStage(stage3)
	sim.AddStage(stage4)
	sim.AddStage(stage5)
	sim.AddStage(stage6)
	sim.AddStage(stage7)
	sim.AddStage(stage8)
	sim.AddStage(stage9)

	if err := sim.Start(); err != nil {
		log.Fatalf("Failed to start simulator: %v", err)
	}

	<-sim.Done()

	sim.PrintStats()
}
