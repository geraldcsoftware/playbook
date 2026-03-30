package credentials

// Provider abstracts credential retrieval.
// Each provider fetches a password from its backing store.
type Provider interface {
	Fetch() (string, error)
}
