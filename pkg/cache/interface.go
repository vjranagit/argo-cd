package cache

// Cacher defines the interface for cache implementations
type Cacher interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Clear()
	Size() int
}

// Ensure both implementations satisfy the interface
var (
	_ Cacher = (*Cache)(nil)
	_ Cacher = (*LRUCache)(nil)
)
