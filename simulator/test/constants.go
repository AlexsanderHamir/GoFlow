package test

type ErrorPriority string

const (
	PriorityLow    ErrorPriority = "don't bother"
	PriorityMedium ErrorPriority = "somebody should fix this"
	PriorityHigh   ErrorPriority = "fix this immediately"
)
