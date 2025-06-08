package simulator

import (
	"sync"
	"sync/atomic"
	"time"
)

// StageMetrics tracks performance metrics for a stage
type StageMetrics struct {
	mu sync.RWMutex

	// Counters
	processedItems uint64
	droppedItems   uint64
	outputItems    uint64

	// State
	startTime time.Time
	endTime   time.Time

	// Generator stats
	generatedItems uint64
}

// NewStageMetrics creates a new metrics collector
func NewStageMetrics() *StageMetrics {
	return &StageMetrics{
		startTime: time.Now(),
	}
}

// RecordProcessing records the processing of an item
func (m *StageMetrics) RecordProcessing() {
	atomic.AddUint64(&m.processedItems, 1)
}

// RecordGenerated records a generated item
func (m *StageMetrics) RecordGenerated() {
	atomic.AddUint64(&m.generatedItems, 1)
}

// RecordDropped records a dropped item
func (m *StageMetrics) RecordDropped() {
	atomic.AddUint64(&m.droppedItems, 1)
}

// RecordOutput records a successful output
func (m *StageMetrics) RecordOutput() {
	atomic.AddUint64(&m.outputItems, 1)
}

// Stop marks the end of metrics collection
func (m *StageMetrics) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endTime = time.Now()
}

// GetStats returns a map of current metrics
func (m *StageMetrics) GetStats() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// For generator stages, return only generator-specific metrics
	if atomic.LoadUint64(&m.generatedItems) > 0 {
		return map[string]any{
			"generated_items": atomic.LoadUint64(&m.generatedItems),
		}
	}

	// For worker stages, return processing metrics
	processed := atomic.LoadUint64(&m.processedItems)
	if processed == 0 {
		return map[string]any{
			"processed_items": 0,
			"drop_rate":       0.0,
			"throughput":      0.0,
		}
	}

	duration := m.endTime.Sub(m.startTime)
	if m.endTime.IsZero() {
		duration = time.Since(m.startTime)
	}

	return map[string]any{
		"processed_items": processed,
		"drop_rate":       float64(atomic.LoadUint64(&m.droppedItems)) / float64(processed),
		"throughput":      float64(processed) / duration.Seconds(),
	}
}
