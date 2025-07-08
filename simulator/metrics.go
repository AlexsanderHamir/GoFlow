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

func (m *StageMetrics) RecordProcessed() {
	atomic.AddUint64(&m.processedItems, 1)
}

func (m *StageMetrics) RecordGenerated() {
	atomic.AddUint64(&m.generatedItems, 1)
}

func (m *StageMetrics) RecordDropped() {
	atomic.AddUint64(&m.droppedItems, 1)
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

	commonMap := m.GetCommons()

	isGenerator := atomic.LoadUint64(&m.generatedItems) > 0
	if isGenerator {
		commonMap["generated_items"] = atomic.LoadUint64(&m.generatedItems)
		return commonMap
	}

	processed := atomic.LoadUint64(&m.processedItems)
	noProcessingHappaned := processed == 0
	if noProcessingHappaned {
		return m.GetEmpty()
	}

	commonMap["processed_items"] = processed
	return commonMap
}

// Returns an empty map indicating that no processing happened.
func (m *StageMetrics) GetEmpty() map[string]any {
	return map[string]any{
		"processed_items": 0,
		"dropped_items":   0,
		"drop_rate":       0.0,
		"throughput":      0.0,
		"output_items":    0,
	}
}

// Returns a map with fields that the generators and workers
// have in common.
func (m *StageMetrics) GetCommons() map[string]any {
	duration := m.endTime.Sub(m.startTime)
	if m.endTime.IsZero() {
		duration = time.Since(m.startTime)
	}

	return map[string]any{
		"drop_rate":     float64(atomic.LoadUint64(&m.droppedItems)) / float64(atomic.LoadUint64(&m.generatedItems)),
		"dropped_items": atomic.LoadUint64(&m.droppedItems),
		"output_items":  atomic.LoadUint64(&m.outputItems),
		"throughput":    float64(atomic.LoadUint64(&m.outputItems)) / duration.Seconds(),
	}
}
