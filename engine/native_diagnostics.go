package main

import (
	"encoding/json"
	"io"
	"time"
)

// Shared Unix microsecond timestamps locate pipe/GUI delays. The recorder
// rejects negative intervals; these are diagnostic clocks, never physics time.
func nativeDiagnosticNowUs() int64 { return time.Now().UnixMicro() }

type nativeFrameTiming struct {
	Sequence int64 `json:"sequence"`
	SentUs   int64 `json:"qt_sent_us"`
	ReadUs   int64 `json:"engine_read_us"`
	ReadyUs  int64 `json:"engine_ready_us"`
}

func emitTimedNativeFrame(writer io.Writer, packet map[string]any, timing *nativeFrameTiming) error {
	// Encode the large scene before stamping readiness. Only the small timing
	// trailer is encoded afterward, so engine work includes scene JSON encoding.
	body, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	timing.ReadyUs = nativeDiagnosticNowUs()
	trailer, err := json.Marshal(timing)
	if err != nil {
		return err
	}
	body = append(body[:len(body)-1], `,"_diagnostics":`...)
	body = append(body, trailer...)
	body = append(body, '}', '\n')
	n, err := writer.Write(body)
	if err == nil && n != len(body) {
		return io.ErrShortWrite
	}
	return err
}
