package example

import (
	"log"
	"math/rand"
	"time"

	"github.com/AlexsanderHamir/GoFlow/pkg/simulator"
)

func Example() {
	// Create simulator
	sim := simulator.NewSimulator()
	sim.Duration = 10 * time.Second
	// sim.MaxGeneratedItems = 10000

	// Create configuration for generator stage
	generatorConfig := &simulator.StageConfig{
		InputRate:  100 * time.Millisecond,
		RoutineNum: 100,
		BufferSize: 5000, // output buffer size
		ItemGenerator: func() any {
			return rand.Intn(100)
		},
	}

	// Create configuration for other stages
	globalConfig := &simulator.StageConfig{
		RoutineNum: 100,
		BufferSize: 5000, // input buffer size
	}

	// The first stage will be considered the generator stage,
	// responsible for feeding data into the pipeline.
	stage1 := simulator.NewStage("Generators", generatorConfig)

	stage2 := simulator.NewStage("Stage-1", globalConfig)
	stage2.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(10 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage3 := simulator.NewStage("Stage-2", globalConfig)
	stage3.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(40 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage4 := simulator.NewStage("Stage-3", globalConfig)
	stage4.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(100 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage5 := simulator.NewStage("Stage-4", globalConfig)
	stage5.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(200 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage6 := simulator.NewStage("Stage-5", globalConfig)
	stage6.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(100 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage7 := simulator.NewStage("Stage-6", globalConfig)
	stage7.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(10 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	stage8 := simulator.NewStage("Stage-7", globalConfig)
	stage8.Config.WorkerFunc = func(item any) (any, error) {
		time.Sleep(300 * time.Millisecond)
		item = item.(int) + rand.Intn(100)
		return item, nil
	}

	// The last stage will be considered the dummy stage, its stats
	// won't be accounted for, its only job is to discard the items.
	stage9 := simulator.NewStage("DummyStages", globalConfig)
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
}
