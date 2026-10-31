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
	Events            []OTLPEvent     `json:"events,omitempty"`
	Status            OTLPStatus      `json:"status"`
}

type OTLPEvent struct {
	Name         string          `json:"name"`
	TimeUnixNano string          `json:"timeUnixNano"`
	Attributes   []OTLPAttribute `json:"attributes"`
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

// OTLPLogsPayload represents the root JSON payload from OpenTelemetry OTLP HTTP log exporters.
type OTLPLogsPayload struct {
	ResourceLogs []OTLPResourceLog `json:"resourceLogs"`
}

type OTLPResourceLog struct {
	Resource  OTLPResource   `json:"resource"`
	ScopeLogs []OTLPScopeLog `json:"scopeLogs"`
}

type OTLPScopeLog struct {
	Scope      OTLPScope       `json:"scope"`
	LogRecords []OTLPLogRecord `json:"logRecords"`
}

type OTLPLogRecord struct {
	TimeUnixNano         string          `json:"timeUnixNano"`
	ObservedTimeUnixNano string          `json:"observedTimeUnixNano"`
	Body                 OTLPValue       `json:"body"`
	Attributes           []OTLPAttribute `json:"attributes"`
	TraceID              string          `json:"traceId"`
	SpanID               string          `json:"spanId"`
}


