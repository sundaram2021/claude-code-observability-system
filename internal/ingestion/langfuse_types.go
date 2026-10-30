package ingestion

import "time"

// LangfuseIngestionPayload matches the official Langfuse batch ingestion request.
type LangfuseIngestionPayload struct {
	Batch []LangfuseBatchItem `json:"batch"`
}

type LangfuseBatchItem struct {
	ID        string              `json:"id"`
	Type      string              `json:"type"` // "trace-create", "span-create", "score-create"
	Timestamp time.Time           `json:"timestamp"`
	Body      LangfuseObservation `json:"body"`
}

type LangfuseObservation struct {
	ID                  string            `json:"id"`
	TraceID             string            `json:"traceId"`
	ParentObservationID string            `json:"parentObservationId,omitempty"`
	SessionID           string            `json:"sessionId,omitempty"`
	Name                string            `json:"name"`
	StartTime           time.Time         `json:"startTime"`
	EndTime             time.Time         `json:"endTime"`
	Model               string            `json:"model,omitempty"`
	Input               any               `json:"input,omitempty"`
	Output              any               `json:"output,omitempty"`
	Value               float64           `json:"value,omitempty"`
	Comment             string            `json:"comment,omitempty"`
	Usage               *LangfuseUsage    `json:"usage,omitempty"`
	StatusMessage       string            `json:"statusMessage,omitempty"`
	Level               string            `json:"level,omitempty"` // "DEFAULT", "ERROR", "WARNING"
	Metadata            map[string]string `json:"metadata,omitempty"`
}

type LangfuseUsage struct {
	Input            int     `json:"input"`
	Output           int     `json:"output"`
	Total            int     `json:"total"`
	TotalCost        float64 `json:"totalCost"`
	InputCost        float64 `json:"inputCost"`
	OutputCost       float64 `json:"outputCost"`
	PromptTokens     int     `json:"promptTokens"`
	CompletionTokens int     `json:"completionTokens"`
}
