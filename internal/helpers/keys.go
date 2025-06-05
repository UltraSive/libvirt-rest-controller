package helpers

type contextKey string

func (c contextKey) String() string {
	return "domain context key " + string(c)
}

// Define specific keys for vmID and vmDir
const (
	VMIDKey  contextKey = "vmID"
	VMDirKey contextKey = "vmDir"
)
