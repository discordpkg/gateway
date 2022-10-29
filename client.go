package gateway

type State interface {
}

type Client struct {
	state State
}
