package ingestion

// OTLPTracesPayload represents the root JSON payload from OpenTelemetry OTLP HTTP exporters.
type OTLPTracesPayload struct {
	ResourceSpans []OTLPResourceSpan `json:"resourceSpans"`
}

type OTLPResourceSpan struct {
	Resource   OTLPResource     `json:"resource"`
	ScopeSpans []OTLPScopeSpan  `json:"scopeSpans"`
}

type OTLPResource struct {
	Attributes []OTLPAttribute `json:"attributes"`
}

type OTLPScopeSpan struct {
	Scope OTLPScope  `json:"scope"`
	Spans []OTLPSpan `json:"spans"`
}

type OTLPScope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type OTLPSpan struct {
	TraceID           string          `json:"traceId"`
	SpanID            string          `json:"spanId"`
	ParentSpanID      string          `json:"parentSpanId,omitempty"`
	Name              string          `json:"name"`
	Kind              int             `json:"kind"`
	StartTimeUnixNano string          `json:"startTimeUnixNano"`
	EndTimeUnixNano   string          `json:"endTimeUnixNano"`
	Attributes        []OTLPAttribute `json:"attributes"`
	Status            OTLPStatus      `json:"status"`
}

type OTLPStatus struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type OTLPAttribute struct {
	Key   string    `json:"key"`
	Value OTLPValue `json:"value"`
}

type OTLPValue struct {
	StringValue string  `json:"stringValue,omitempty"`
	IntValue    any     `json:"intValue,omitempty"`
	DoubleValue float64 `json:"doubleValue,omitempty"`
	BoolValue   bool    `json:"boolValue,omitempty"`
}

