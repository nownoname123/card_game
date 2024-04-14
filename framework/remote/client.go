package remote

type Client interface {
	Run() error
	Close() error
	SendMsg(string, []byte) error
}
