package cache

import (
	"testing"
	"time"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache(3, time.Minute)

	// Test Set and Get
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	if val, found := cache.Get("key1"); !found || val != "value1" {
		t.Error("Failed to get key1")
	}

	// Test Size
	if size := cache.Size(); size != 3 {
		t.Errorf("Expected size 3, got %d", size)
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := NewLRUCache(2, time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3") // Should evict key1

	// key1 should be evicted
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should have been evicted")
	}

	// key2 and key3 should still exist
	if _, found := cache.Get("key2"); !found {
		t.Error("key2 should exist")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("key3 should exist")
	}

	// Check stats
	stats := cache.Stats()
	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
	}
}

func TestLRUCache_LRUOrder(t *testing.T) {
	cache := NewLRUCache(2, time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	
	// Access key1 (moves it to front)
	cache.Get("key1")
	
	// Add key3 (should evict key2, not key1)
	cache.Set("key3", "value3")

	// key2 should be evicted
	if _, found := cache.Get("key2"); found {
		t.Error("key2 should have been evicted (LRU)")
	}

	// key1 and key3 should exist
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should exist")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("key3 should exist")
	}
}

func TestLRUCache_Expiration(t *testing.T) {
	cache := NewLRUCache(10, 100*time.Millisecond)

	cache.Set("key1", "value1")

	// Should exist immediately
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should exist")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	if _, found := cache.Get("key1"); found {
		t.Error("key1 should have expired")
	}

	// Check stats
	stats := cache.Stats()
	if stats.Expirations == 0 {
		t.Error("Expected expiration count > 0")
	}
}

func TestLRUCache_Stats(t *testing.T) {
	cache := NewLRUCache(10, time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Generate hits
	cache.Get("key1")
	cache.Get("key1")
	cache.Get("key2")

	// Generate miss
	cache.Get("nonexistent")

	stats := cache.Stats()

	if stats.Hits != 3 {
		t.Errorf("Expected 3 hits, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.HitRate < 74.9 || stats.HitRate > 75.1 {
		t.Errorf("Expected hit rate ~75%%, got %.2f%%", stats.HitRate)
	}

	if stats.Size != 2 {
		t.Errorf("Expected size 2, got %d", stats.Size)
	}

	// Test reset
	cache.ResetStats()
	stats = cache.Stats()

	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("Stats should be reset to 0")
	}
}

func TestLRUCache_Update(t *testing.T) {
	cache := NewLRUCache(5, time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key1", "value2") // Update

	if val, found := cache.Get("key1"); !found || val != "value2" {
		t.Error("Failed to update value")
	}

	// Size should still be 1
	if size := cache.Size(); size != 1 {
		t.Errorf("Expected size 1 after update, got %d", size)
	}
}

func TestLRUCache_Delete(t *testing.T) {
	cache := NewLRUCache(5, time.Minute)

	cache.Set("key1", "value1")
	cache.Delete("key1")

	if _, found := cache.Get("key1"); found {
		t.Error("key1 should be deleted")
	}

	if size := cache.Size(); size != 0 {
		t.Errorf("Expected size 0 after delete, got %d", size)
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRUCache(5, time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	cache.Clear()

	if size := cache.Size(); size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", size)
	}
}
