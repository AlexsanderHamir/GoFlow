package simulator

import (
	"sync"
	"sync/atomic"
	"time"
)

// StageMetrics tracks performance metrics for a stage
type StageMetrics struct {
	mu             sync.RWMutex
	ProcessedItems uint64
	DroppedItems   uint64
	OutputItems    uint64
	StartTime      time.Time
	EndTime        time.Time
	GeneratedItems uint64
}

func NewStageMetrics() *StageMetrics {
	return &StageMetrics{
		StartTime: time.Now(),
	}
}

func (m *StageMetrics) RecordProcessing() {
	atomic.AddUint64(&m.ProcessedItems, 1)
}

func (m *StageMetrics) RecordGenerated() {
	atomic.AddUint64(&m.GeneratedItems, 1)
}

func (m *StageMetrics) RecordGeneratedBurst(items int) {
	atomic.AddUint64(&m.GeneratedItems, uint64(items))
}

func (m *StageMetrics) RecordDropped() {
	atomic.AddUint64(&m.DroppedItems, 1)
}

func (m *StageMetrics) RecordDroppedBurst(items int) {
	atomic.AddUint64(&m.DroppedItems, uint64(items))
}

func (m *StageMetrics) RecordOutput() {
	atomic.AddUint64(&m.OutputItems, 1)
}

// Stop marks the end of metrics collection
func (m *StageMetrics) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EndTime = time.Now()
}

// GetStats returns a map of current metrics
func (m *StageMetrics) GetStats() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	duration := m.EndTime.Sub(m.StartTime)
	if m.EndTime.IsZero() {
		duration = time.Since(m.StartTime)
	}

	// For generator stages, return only generator-specific metrics
	if atomic.LoadUint64(&m.GeneratedItems) > 0 {
		return map[string]any{
			"generated_items": atomic.LoadUint64(&m.GeneratedItems),
			"drop_rate":       float64(atomic.LoadUint64(&m.DroppedItems)) / float64(atomic.LoadUint64(&m.GeneratedItems)),
			"dropped_items":   atomic.LoadUint64(&m.DroppedItems),
			"output_items":    atomic.LoadUint64(&m.OutputItems),
			"throughput":      float64(atomic.LoadUint64(&m.OutputItems)) / duration.Seconds(),
		}
	}

	// For worker stages, return processing metrics
	processed := atomic.LoadUint64(&m.ProcessedItems)
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
		"drop_rate":       float64(atomic.LoadUint64(&m.DroppedItems)) / float64(processed),
		"dropped_items":   atomic.LoadUint64(&m.DroppedItems),
		"throughput":      float64(atomic.LoadUint64(&m.OutputItems)) / duration.Seconds(),
		"output_items":    atomic.LoadUint64(&m.OutputItems),
	}
}
