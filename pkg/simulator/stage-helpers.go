package simulator

import (
	"log"
	"time"
)

// processBurst handles sending a burst of items to the output channel
func (s *Stage) processBurst(items []any) {
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
	if s.Config.ItemGenerator == nil {
		return
	}

	if s.Config.InputRate > 0 {
		time.Sleep(s.Config.InputRate)
	}

	select {
	case <-s.Config.Ctx.Done():
		log.Println("Context done, stopping generation")
		return
	case s.Output <- s.Config.ItemGenerator():
		log.Println("Item generated and sent to output")
		s.metrics.RecordOutput()
	default:
		log.Println("Backpressure detected")
		if s.Config.DropOnBackpressure {
			s.metrics.RecordDropped()
		} else {
			s.Output <- s.Config.ItemGenerator()
			s.metrics.RecordOutput()
		}
	}
}

// processWorkerItem handles the processing of a single item in the worker loop
func (s *Stage) processWorkerItem(item any) (any, error) {
	startTime := time.Now()
	result, err := s.processItem(item)
	processingTime := time.Since(startTime)
	s.metrics.RecordProcessing(processingTime, err)
	
	return result, err
}

// handleWorkerOutput manages sending the processed item to the output channel with backpressure handling
func (s *Stage) handleWorkerOutput(result any) {
	select {
	case <-s.Config.Ctx.Done():
		log.Println("Context done, dropping item")
		return
	case s.Output <- result:
		s.metrics.RecordOutput()
	default:
		if s.Config.DropOnBackpressure {
			s.metrics.RecordDropped()
		} else {
			s.Output <- result
			s.metrics.RecordOutput()
		}
	}
}
