package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vjranagit/argocd-observability-extensions/internal/models"
)

// handleExportMetrics handles exporting metrics to CSV or JSON
func (s *Server) handleExportMetrics(w http.ResponseWriter, r *http.Request) {
	// Extract path parameters
	_application := chi.URLParam(r, "application")
	groupkind := chi.URLParam(r, "groupkind")
	row := chi.URLParam(r, "row")
	graph := chi.URLParam(r, "graph")

	// Extract query parameters
	appQueryParam := r.URL.Query().Get("application_name")
	projectQueryParam := r.URL.Query().Get("project")
	format := r.URL.Query().Get("format") // csv or json

	// Validate format
	if format != "csv" && format != "json" {
		format = "json" // default to JSON
	}

	// Validate required parameters
	if appQueryParam == "" {
		s.respondError(w, http.StatusBadRequest, "missing parameter", "application_name is required")
		return
	}
	if projectQueryParam == "" {
		s.respondError(w, http.StatusBadRequest, "missing parameter", "project is required")
		return
	}

	// Build query
	query := &models.MetricsQuery{
		Application: appQueryParam,
		Project:     projectQueryParam,
		GroupKind:   groupkind,
		Row:         row,
		Graph:       graph,
	}

	// Execute query via provider
	response, err := s.provider.Query(r.Context(), query)
	if err != nil {
		s.logger.Error("query failed", "error", err)
		s.respondError(w, http.StatusInternalServerError, "query failed", err.Error())
		return
	}

	// Export based on format
	if format == "csv" {
		s.exportCSV(w, response)
	} else {
		s.exportJSON(w, response)
	}
}

// exportCSV exports metrics data as CSV
func (s *Server) exportCSV(w http.ResponseWriter, response *models.MetricsResponse) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", 
		fmt.Sprintf("attachment; filename=metrics_%s_%s.csv", 
			response.Application, time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"Timestamp", "Value"}
	
	// Add label columns (dynamically based on first data point)
	if len(response.Data) > 0 && len(response.Data[0].Labels) > 0 {
		for key := range response.Data[0].Labels {
			header = append(header, key)
		}
	}
	
	if err := writer.Write(header); err != nil {
		s.logger.Error("failed to write CSV header", "error", err)
		return
	}

	// Write data rows
	for _, data := range response.Data {
		row := []string{
			data.Timestamp.Format(time.RFC3339),
			strconv.FormatFloat(data.Value, 'f', -1, 64),
		}

		// Add label values in same order as header
		for i := 2; i < len(header); i++ {
			labelKey := header[i]
			if val, ok := data.Labels[labelKey]; ok {
				row = append(row, val)
			} else {
				row = append(row, "")
			}
		}

		if err := writer.Write(row); err != nil {
			s.logger.Error("failed to write CSV row", "error", err)
			return
		}
	}

	s.logger.Info("exported metrics as CSV", 
		"application", response.Application,
		"rows", len(response.Data))
}

// exportJSON exports metrics data as JSON
func (s *Server) exportJSON(w http.ResponseWriter, response *models.MetricsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", 
		fmt.Sprintf("attachment; filename=metrics_%s_%s.json", 
			response.Application, time.Now().Format("20060102_150405")))

	// Create export structure with metadata
	export := map[string]interface{}{
		"metadata": map[string]interface{}{
			"application":  response.Application,
			"project":      response.Project,
			"graph":        response.Graph,
			"exported_at":  time.Now().Format(time.RFC3339),
			"data_points":  len(response.Data),
		},
		"data": response.Data,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(export); err != nil {
		s.logger.Error("failed to encode JSON", "error", err)
		http.Error(w, "Failed to export metrics", http.StatusInternalServerError)
		return
	}

	s.logger.Info("exported metrics as JSON", 
		"application", response.Application,
		"rows", len(response.Data))
}
