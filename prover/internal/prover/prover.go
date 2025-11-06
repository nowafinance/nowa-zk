package prover

type Service struct{}

func New() *Service {
    return &Service{}
}

func (s *Service) Info() string {
    return "zk-prover initialized"
}


