package websocket

type MessageType string

const (
	MessageTypeStageSetUp MessageType = "stage_setup"
)

type StageSetUp struct {
	Type        MessageType `json:"type"`
	StageName   string      `json:"stage_name"`
	RoutineNum  int         `json:"routine_num"`
	IsFinal     bool        `json:"is_final"`
	IsGenerator bool        `json:"is_generator"`
}
