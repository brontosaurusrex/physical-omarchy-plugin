package main

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

type shortDiagnosticWriter struct{}

func (shortDiagnosticWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestDiagnosticPacketTiming(t *testing.T) {
	var output bytes.Buffer
	stamp := nativeDiagnosticNowUs()
	timing := &nativeFrameTiming{Sequence: 7, SentUs: stamp - 8000, ReadUs: stamp - 5000}
	if err := emitTimedNativeFrame(&output, map[string]any{"type": "frame", "ops": []any{"test"}}, timing); err != nil {
		t.Fatal(err)
	}
	var packet struct {
		Type   string
		Timing nativeFrameTiming `json:"_diagnostics"`
	}
	if err := json.Unmarshal(output.Bytes(), &packet); err != nil {
		t.Fatal(err)
	}
	if packet.Type != "frame" || packet.Timing.Sequence != 7 || packet.Timing.ReadyUs < stamp || packet.Timing.ReadUs != stamp-5000 {
		t.Fatalf("invalid diagnostic packet: %+v", packet)
	}
	if err := emitTimedNativeFrame(shortDiagnosticWriter{}, map[string]any{}, timing); err != io.ErrShortWrite {
		t.Fatalf("short write not reported: %v", err)
	}
}
