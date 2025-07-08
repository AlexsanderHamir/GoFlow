package test

type ErrorPriority string
const (
	PriorityLow    ErrorPriority = "don't bother"
	PriorityMedium ErrorPriority = "somebody should fix this"
	PriorityHigh   ErrorPriority = "fix this immediately"
)

type ErrorMessages string
const (
	StartSimulationBaseError = "failed to initialize stages: "
	MissingItemGenerator     = "ItemGenerator must be set for generator stage"
)
