package simulator

import (
	"sync"
	"sync/atomic"
	"time"
)

// StageMetrics tracks performance metrics for a stage
type StageMetrics struct {
	mu             sync.RWMutex
	processedItems uint64
	droppedItems   uint64
	outputItems    uint64
	startTime      time.Time
	endTime        time.Time
	generatedItems uint64
}

func NewStageMetrics() *StageMetrics {
	return &StageMetrics{
		startTime: time.Now(),
	}
}

func (m *StageMetrics) RecordProcessing() {
	atomic.AddUint64(&m.processedItems, 1)
}

func (m *StageMetrics) RecordGenerated() {
	atomic.AddUint64(&m.generatedItems, 1)
}

func (m *StageMetrics) RecordGeneratedBurst(items int) {
	atomic.AddUint64(&m.generatedItems, uint64(items))
}

func (m *StageMetrics) RecordDropped() {
	atomic.AddUint64(&m.droppedItems, 1)
}

func (m *StageMetrics) RecordDroppedBurst(items int) {
	atomic.AddUint64(&m.droppedItems, uint64(items))
}

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

	duration := m.endTime.Sub(m.startTime)
	if m.endTime.IsZero() {
		duration = time.Since(m.startTime)
	}

	// For generator stages, return only generator-specific metrics
	if atomic.LoadUint64(&m.generatedItems) > 0 {
		return map[string]any{
			"generated_items": atomic.LoadUint64(&m.generatedItems),
			"drop_rate":       float64(atomic.LoadUint64(&m.droppedItems)) / float64(atomic.LoadUint64(&m.generatedItems)),
			"dropped_items":   atomic.LoadUint64(&m.droppedItems),
			"output_items":    atomic.LoadUint64(&m.outputItems),
			"throughput":      float64(atomic.LoadUint64(&m.outputItems)) / duration.Seconds(),
		}
	}

	// For worker stages, return processing metrics
	processed := atomic.LoadUint64(&m.processedItems)
	if processed == 0 {
		return map[string]any{
			"processed_items": 0,
			"dropped_items":   0,
			"drop_rate":       0.0,
			"throughput":      0.0,
			"output_items":    0,
		}
	}

	return map[string]any{
		"processed_items": processed,
		"drop_rate":       float64(atomic.LoadUint64(&m.droppedItems)) / float64(processed),
		"dropped_items":   atomic.LoadUint64(&m.droppedItems),
		"throughput":      float64(atomic.LoadUint64(&m.outputItems)) / duration.Seconds(),
		"output_items":    atomic.LoadUint64(&m.outputItems),
	}
}
