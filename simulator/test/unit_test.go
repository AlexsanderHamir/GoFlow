package test

import (
	"testing"

	lib "github.com/AlexsanderHamir/GoFlow/simulator"
	"github.com/stretchr/testify/assert"
)

func TestAddStageValidation(t *testing.T) {
	sim := lib.NewSimulator()

	err := sim.AddStage(nil)
	assert.Error(t, err, "stage cannot be nil")

	err = sim.AddStage(&lib.Stage{Name: "Testing"})
	assert.Error(t, err, "must provide configuration")

	err = sim.AddStage(&lib.Stage{})
	assert.Error(t, err, "stage name cannot be empty")

	err = sim.AddStage(&lib.Stage{Name: "first", Config: lib.DefaultConfig()})
	assert.NoError(t, err)

	err = sim.AddStage(&lib.Stage{Name: "first"})
	assert.Error(t, err, "repeated name not allowed:")
}

func TestConfigValidation(t *testing.T) {
	sim := lib.NewSimulator()

	err := sim.AddStage(&lib.Stage{Name: "Testing", Config: &lib.StageConfig{}})
	assert.NoError(t, err)

	err = sim.Start(lib.Nothing)
	assert.Error(t, err, startSimulationBaseError+missingItemGenerator)
}
