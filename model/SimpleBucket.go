package model

type SimpleBucket struct {
	Labels  string
	Weights uint32
}

func (s *SimpleBucket) Label() string {
	return s.Labels
}

func (s *SimpleBucket) Weight() uint32 {
	return s.Weights
}
