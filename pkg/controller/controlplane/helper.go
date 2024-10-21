package controlplane

type (
	key int

	cacheKey struct {
		nodeCIDR  string
		projectID string
	}
)

const (
	ClientKey key = iota
)
