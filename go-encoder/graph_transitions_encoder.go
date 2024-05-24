package main

// GraphTransitionEncoder store transition map
// 1K x uin16 -> (ordered most frequent cache of 64x uint6 -> uint16, where they transition).
// this fit that most of samples transition into <64 other states.
// this is same as simple cache encoder compression ratio.
type GraphTransitionEncoder struct {
	numTotal int
	g        map[uint16]map[uint16]int
	prev     uint16
}

func NewGraphTransitionEncoder() *GraphTransitionEncoder {
	return &GraphTransitionEncoder{
		g: make(map[uint16]map[uint16]int),
	}
}

func (s *GraphTransitionEncoder) NumPossibleTransitions() int {
	count := 0
	for _, to := range s.g {
		count += len(to)
	}
	return count
}

func (s *GraphTransitionEncoder) NumFrom() int { return len(s.g) }

func (s *GraphTransitionEncoder) ToCount() map[uint16]int {
	count := make(map[uint16]int)
	for _, to := range s.g {
		for q, n := range to {
			count[q] += n
		}
	}
	return count
}

func (s *GraphTransitionEncoder) NumTo() int { return len(s.ToCount()) }

func (s *GraphTransitionEncoder) MaxFrom() int {
	max := 0
	for _, to := range s.g {
		if len(to) > max {
			max = len(to)
		}
	}
	return max
}

func (s *GraphTransitionEncoder) MaxTo() int {
	max := 0
	for _, k := range s.ToCount() {
		if k > max {
			max = k
		}
	}
	return max
}

type GraphTransitionEncoderStats struct {
	NumPossibleTransitions int
	NumFrom                int
	NumTo                  int
	MaxFrom                int
	MaxTo                  int
}

func (s *GraphTransitionEncoder) Stats() GraphTransitionEncoderStats {
	return GraphTransitionEncoderStats{
		NumPossibleTransitions: s.NumPossibleTransitions(),
		NumFrom:                s.NumFrom(),
		NumTo:                  s.NumTo(),
		MaxFrom:                s.MaxFrom(),
		MaxTo:                  s.MaxTo(),
	}
}

func (s *GraphTransitionEncoder) Write(v uint16) error {
	defer func() {
		s.numTotal++
		s.prev = v
	}()

	if s.numTotal == 0 {
		return nil
	}

	if _, ok := s.g[s.prev]; !ok {
		s.g[s.prev] = make(map[uint16]int)
	}
	s.g[s.prev][v]++

	return nil
}
