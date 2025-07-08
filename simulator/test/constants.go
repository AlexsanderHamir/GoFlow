package test

// ErrorPriority represents the priority of the possible errors a test may find.
type errorPriority string

// Available priorities
const (
	priorityLow    errorPriority = "don't bother"
	priorityMedium errorPriority = "somebody should fix this"
	priorityHigh   errorPriority = "fix this immediately"
)

// ErrorMessages standardized
type errorMessages string

// Current cataloged
const (
	startSimulationBaseError errorMessages = "failed to initialize stages: "
	missingItemGenerator     errorMessages = "ItemGenerator must be set for generator stage"
)
