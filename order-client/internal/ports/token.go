package ports

type TokenProvider interface {
	GetAccessToken() (string, error)
	HasToken() bool
	Close() error
}
