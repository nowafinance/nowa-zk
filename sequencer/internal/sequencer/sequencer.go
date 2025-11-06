package sequencer

type Service struct{}

func New() *Service {
    return &Service{}
}

func (s *Service) Info() string {
    return "zk-sequencer initialized"
}


