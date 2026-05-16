package balancer

type Balancer struct {
	Backends []*Backend
	Counter  uint64
}

func NewBalancer(backends []*Backend) *Balancer {
	return &Balancer{
		Backends: backends,
		Counter:  0,
	}
}
