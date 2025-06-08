package simulator

import (
	"log"
	"time"
)

// processBurst handles sending a burst of items to the output channel
func (s *Stage) processBurst(items []any) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in processBurst")
		}
	}()

	for _, item := range items {
		select {
		case <-s.Config.Ctx.Done():
			log.Println("Context done, dropping burst")
			return
		case s.Output <- item:
			s.metrics.RecordOutput()
		default:
			if s.Config.DropOnBackpressure {
				s.metrics.RecordDropped()
			} else {
				s.Output <- item
				s.metrics.RecordOutput()
			}
		}
	}
}

// shouldProcessBurst determines if it's time to process a burst based on configuration and timing
func (s *Stage) shouldProcessBurst(burstCount int, lastBurstTime time.Time) bool {
	if s.Config.InputBurst == nil || s.Config.BurstCount <= 0 {
		return false
	}

	now := time.Now()
	return burstCount < s.Config.BurstCount && now.Sub(lastBurstTime) >= s.Config.BurstInterval
}

// processRegularGeneration handles the regular item generation flow
func (s *Stage) processRegularGeneration() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in processRegularGeneration")
		}
	}()

	if s.Config.ItemGenerator == nil {
		return
	}

	if s.Config.InputRate > 0 {
		time.Sleep(s.Config.InputRate)
	}

	item := s.Config.ItemGenerator()

	select {
	case <-s.Config.Ctx.Done():
		return
	case s.Output <- item:
		s.metrics.RecordOutput()
	default:
		if s.Config.DropOnBackpressure {
			s.metrics.RecordDropped()
		} else {
			s.Output <- item
			s.metrics.RecordOutput()
		}
	}
}

// processWorkerItem handles the processing of a single item in the worker loop
func (s *Stage) processWorkerItem(item any) (any, error) {
	result, err := s.processItem(item)
	s.metrics.RecordProcessing()

	return result, err
}

// handleWorkerOutput manages sending the processed item to the output channel with backpressure handling
func (s *Stage) handleWorkerOutput(result any) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic in handleWorkerOutput")
		}
	}()

	select {
	case <-s.Config.Ctx.Done():
		log.Println("Context done, dropping item")
		return
	case s.Output <- result:
		log.Println("Item sent to output from handleWorkerOutput")
		s.metrics.RecordOutput()
	default:
		if s.Config.DropOnBackpressure {
			log.Println("Dropping item")
			s.metrics.RecordDropped()
		} else {
			s.Output <- result
			log.Println("Item sent to output from backpressure from handleWorkerOutput")
			s.metrics.RecordOutput()
		}
	}
}
