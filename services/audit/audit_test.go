// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileLogger_Stdout(t *testing.T) {
	logger, err := NewFileLogger("")
	if err != nil {
		t.Fatalf("NewFileLogger stdout: %v", err)
	}
	defer logger.Close()

	err = logger.Log(Entry{
		RequestID: "req-1",
		Source:    "test",
		Target:    "server",
		Action:    "test.action",
		Decision:  "allowed",
		Result:    "ok",
	})
	if err != nil {
		t.Errorf("Log to stdout: %v", err)
	}
}

func TestFileLogger_File(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "audit.jsonl")
	logger, err := NewFileLogger(tmpFile)
	if err != nil {
		t.Fatalf("NewFileLogger file: %v", err)
	}

	err = logger.Log(Entry{
		RequestID: "req-1",
		Source:    "test.client",
		Target:    "eipc-server",
		Action:    "authenticate",
		Decision:  "allowed",
		Result:    "session created",
	})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	err = logger.Log(Entry{
		RequestID: "req-2",
		Source:    "test.client",
		Target:    "eipc-server",
		Action:    "ui.cursor.move",
		Decision:  "denied",
		Result:    "capability violation",
	})
	if err != nil {
		t.Fatalf("Log second entry: %v", err)
	}

	logger.Close()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var entry Entry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("unmarshal first line: %v", err)
	}
	if entry.RequestID != "req-1" {
		t.Errorf("expected request_id 'req-1', got %q", entry.RequestID)
	}
	if entry.Timestamp == "" {
		t.Error("expected auto-filled timestamp")
	}
}

func TestFileLogger_TimestampAutoFill(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "audit-ts.jsonl")
	logger, err := NewFileLogger(tmpFile)
	if err != nil {
		t.Fatalf("NewFileLogger: %v", err)
	}

	logger.Log(Entry{Source: "test", Action: "check"})
	logger.Close()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	var entry Entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry.Timestamp == "" {
		t.Error("timestamp should be auto-filled when empty")
	}
}

func TestFileLogger_PresetTimestamp(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "audit-preset.jsonl")
	logger, err := NewFileLogger(tmpFile)
	if err != nil {
		t.Fatalf("NewFileLogger: %v", err)
	}

	customTS := "2026-01-01T00:00:00Z"
	logger.Log(Entry{Timestamp: customTS, Source: "test", Action: "check"})
	logger.Close()

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	var entry Entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry.Timestamp != customTS {
		t.Errorf("expected preset timestamp %q, got %q", customTS, entry.Timestamp)
	}
}

func TestFileLogger_Close_Stdout(t *testing.T) {
	logger, _ := NewFileLogger("")
	err := logger.Close()
	if err != nil {
		t.Errorf("closing stdout logger should not error: %v", err)
	}
}
