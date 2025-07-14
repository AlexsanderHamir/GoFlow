package simulator

import (
	"sync"
	"sync/atomic"
	"time"
)

type stageMetrics struct {
	mu             sync.RWMutex
	processedItems uint64
	droppedItems   uint64
	outputItems    uint64
	startTime      time.Time
	endTime        time.Time
	generatedItems uint64
}

func newStageMetrics() *stageMetrics {
	return &stageMetrics{
		startTime: time.Now(),
	}
}

func (m *stageMetrics) recordProcessed() {
	atomic.AddUint64(&m.processedItems, 1)
}

func (m *stageMetrics) recordGenerated() {
	atomic.AddUint64(&m.generatedItems, 1)
}

func (m *stageMetrics) recordDropped() {
	atomic.AddUint64(&m.droppedItems, 1)
}

func (m *stageMetrics) recordOutput() {
	atomic.AddUint64(&m.outputItems, 1)
}

func (m *stageMetrics) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endTime = time.Now()
}

// GetStats returns a map of current metrics
func (m *stageMetrics) GetStats() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	commonMap := m.getCommons()

	isGenerator := atomic.LoadUint64(&m.generatedItems) > 0
	if isGenerator {
		commonMap["generated_items"] = atomic.LoadUint64(&m.generatedItems)
		return commonMap
	}

	processed := atomic.LoadUint64(&m.processedItems)
	noProcessingHappaned := processed == 0
	if noProcessingHappaned {
		return m.getEmpty()
	}

	commonMap["processed_items"] = processed
	return commonMap
}

func (m *stageMetrics) getEmpty() map[string]any {
	return map[string]any{
		"processed_items": 0,
		"dropped_items":   0,
		"drop_rate":       0.0,
		"throughput":      0.0,
		"output_items":    0,
	}
}

func (m *stageMetrics) getCommons() map[string]any {
	duration := m.endTime.Sub(m.startTime)
	if m.endTime.IsZero() {
		duration = time.Since(m.startTime)
	}

	gen := atomic.LoadUint64(&m.generatedItems)
	drop := atomic.LoadUint64(&m.droppedItems)
	out := atomic.LoadUint64(&m.outputItems)

	var dropRate float64
	if gen > 0 {
		dropRate = float64(drop) / float64(gen)
	}

	var throughput float64
	if duration.Seconds() > 0 {
		throughput = float64(out) / duration.Seconds()
	}

	return map[string]any{
		"drop_rate":     dropRate,
		"dropped_items": drop,
		"output_items":  out,
		"throughput":    throughput,
	}
}
