package server

import (
	"encoding/json"
	"net/http"
)

// handleCacheStats returns cache performance statistics
func (s *Server) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		s.respondError(w, http.StatusServiceUnavailable, "cache not enabled", "cache is not configured")
		return
	}

	// Try to get stats if the cache supports it
	type Statable interface {
		Stats() interface{}
	}

	var stats interface{}
	if statCache, ok := s.cache.(Statable); ok {
		stats = statCache.Stats()
	} else {
		// Fallback for basic cache without stats
		stats = map[string]interface{}{
			"size": s.cache.Size(),
			"note": "detailed statistics not available for this cache implementation",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cache_stats": stats,
	})

	s.logger.Debug("cache stats requested")
}
