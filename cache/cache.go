package cache

// Cache interface for cache
type Cache interface {
	Get(string) (string, error)
	Set(string, string) error
}
