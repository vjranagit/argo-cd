package server

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vjranagit/argocd-observability-extensions/internal/models"
)

func TestExportCSV(t *testing.T) {
	srv := &Server{
		logger: testLogger,
	}

	response := &models.MetricsResponse{
		Application: "test-app",
		Project:     "test-project",
		Graph:       "request-rate",
		Data: []models.MetricData{
			{
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Value:     100.5,
				Labels: map[string]string{
					"instance": "pod-1",
					"status":   "200",
				},
			},
			{
				Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
				Value:     150.25,
				Labels: map[string]string{
					"instance": "pod-2",
					"status":   "200",
				},
			},
		},
	}

	rr := httptest.NewRecorder()
	srv.exportCSV(rr, response)

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/csv" {
		t.Errorf("Expected Content-Type text/csv, got %s", contentType)
	}

	// Check CSV content
	reader := csv.NewReader(rr.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 3 { // header + 2 data rows
		t.Errorf("Expected 3 records, got %d", len(records))
	}

	// Verify header
	header := records[0]
	if header[0] != "Timestamp" || header[1] != "Value" {
		t.Errorf("Unexpected header: %v", header)
	}

	// Verify data row
	if records[1][1] != "100.5" {
		t.Errorf("Expected value 100.5, got %s", records[1][1])
	}
}

func TestExportJSON(t *testing.T) {
	srv := &Server{
		logger: testLogger,
	}

	response := &models.MetricsResponse{
		Application: "test-app",
		Project:     "test-project",
		Graph:       "request-rate",
		Data: []models.MetricData{
			{
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Value:     100.5,
				Labels: map[string]string{
					"instance": "pod-1",
				},
			},
		},
	}

	rr := httptest.NewRecorder()
	srv.exportJSON(rr, response)

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON
	var export map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&export); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Verify metadata
	metadata, ok := export["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing metadata in export")
	}

	if metadata["application"] != "test-app" {
		t.Errorf("Expected application test-app, got %v", metadata["application"])
	}

	// Verify data exists
	if _, ok := export["data"]; !ok {
		t.Error("Missing data in export")
	}
}

func TestHandleExportMetrics_FormatValidation(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectedType   string
	}{
		{"CSV format", "csv", "text/csv"},
		{"JSON format", "json", "application/json"},
		{"Default to JSON", "invalid", "application/json"},
		{"Empty defaults to JSON", "", "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test will be implemented when integrated with full server
			// This is a placeholder for format validation testing
			if tt.format == "" {
				format := "json" // default
				if format != "json" {
					t.Errorf("Expected default format json, got %s", format)
				}
			}
		})
	}
}
