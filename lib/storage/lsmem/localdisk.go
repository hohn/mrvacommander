package lsmem

type Storage struct {
	CurrentID int
}

func (s *Storage) NextID() int {
	s.CurrentID += 1
	return s.CurrentID
}
