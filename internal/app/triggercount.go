package app

// TriggerCount returns the number of active triggers.
func (s *Service) TriggerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.engine.TriggerCount()
}
