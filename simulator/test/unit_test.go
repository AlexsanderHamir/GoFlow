package test

import (
	"testing"

	"github.com/AlexsanderHamir/GoFlow/simulator"
	"github.com/stretchr/testify/assert"
)

func TestAddStageValidation(t *testing.T) {
	sim := simulator.NewSimulator()

	err := sim.AddStage(nil)
	assert.Error(t, err, "stage cannot be nil")

	err = sim.AddStage(&simulator.Stage{Name: "Testing"})
	assert.Error(t, err, "must provide configuration")

	err = sim.AddStage(&simulator.Stage{})
	assert.Error(t, err, "stage name cannot be empty")

	err = sim.AddStage(&simulator.Stage{Name: "first", Config: simulator.DefaultConfig()})
	assert.NoError(t, err)

	err = sim.AddStage(&simulator.Stage{Name: "first"})
	assert.Error(t, err, "repeated name not allowed:")
}

func TestConfigValidation(t *testing.T) {
	sim := simulator.NewSimulator()

	err := sim.AddStage(&simulator.Stage{Name: "Testing", Config: &simulator.StageConfig{}})
	assert.NoError(t, err)

	err = sim.Start(false)
	assert.Error(t, err, startSimulationBaseError+missingItemGenerator)
}
