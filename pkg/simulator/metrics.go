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
	errorCount     uint64
	propagatedErrors uint64
	retryCount     uint64
	droppedItems   uint64
	outputItems    uint64

	// Timing
	totalProcessingTime time.Duration
	minProcessingTime   time.Duration
	maxProcessingTime   time.Duration

	// State
	startTime time.Time
	endTime   time.Time
}

// NewStageMetrics creates a new metrics collector
func NewStageMetrics() *StageMetrics {
	return &StageMetrics{
		startTime:         time.Now(),
		minProcessingTime: time.Duration(1<<63 - 1), // Max int64
	}
}

// RecordError records an error
func (m *StageMetrics) RecordError(propagated bool) {
	if propagated {
		atomic.AddUint64(&m.propagatedErrors, 1)
	} else {
		atomic.AddUint64(&m.errorCount, 1)
	}
}

// RecordProcessing records the processing of an item
func (m *StageMetrics) RecordProcessing(duration time.Duration, err error) {
	atomic.AddUint64(&m.processedItems, 1)
	if err != nil {
		atomic.AddUint64(&m.errorCount, 1)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalProcessingTime += duration
	if duration < m.minProcessingTime {
		m.minProcessingTime = duration
	}
	
	if duration > m.maxProcessingTime {
		m.maxProcessingTime = duration
	}
}

// RecordRetry records a retry attempt
func (m *StageMetrics) RecordRetry() {
	atomic.AddUint64(&m.retryCount, 1)
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

// Clone creates a copy of the current metrics
func (m *StageMetrics) Clone() *StageMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &StageMetrics{
		processedItems:      atomic.LoadUint64(&m.processedItems),
		errorCount:          atomic.LoadUint64(&m.errorCount),
		retryCount:          atomic.LoadUint64(&m.retryCount),
		droppedItems:        atomic.LoadUint64(&m.droppedItems),
		outputItems:         atomic.LoadUint64(&m.outputItems),
		totalProcessingTime: m.totalProcessingTime,
		minProcessingTime:   m.minProcessingTime,
		maxProcessingTime:   m.maxProcessingTime,
		startTime:           m.startTime,
		endTime:             m.endTime,
	}
}

// GetStats returns a map of current metrics
func (m *StageMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	processed := atomic.LoadUint64(&m.processedItems)
	if processed == 0 {
		return map[string]any{
			"processed_items": 0,
			"error_rate":      0.0,
			"retry_rate":      0.0,
			"drop_rate":       0.0,
			"avg_processing":  time.Duration(0),
			"min_processing":  time.Duration(0),
			"max_processing":  time.Duration(0),
			"throughput":      0.0,
		}
	}

	duration := m.endTime.Sub(m.startTime)
	if m.endTime.IsZero() {
		duration = time.Since(m.startTime)
	}

	return map[string]any{
		"processed_items": processed,
		"error_rate":      float64(atomic.LoadUint64(&m.errorCount)) / float64(processed),
		"retry_rate":      float64(atomic.LoadUint64(&m.retryCount)) / float64(processed),
		"drop_rate":       float64(atomic.LoadUint64(&m.droppedItems)) / float64(processed),
		"avg_processing":  m.totalProcessingTime / time.Duration(processed),
		"min_processing":  m.minProcessingTime,
		"max_processing":  m.maxProcessingTime,
		"throughput":      float64(processed) / duration.Seconds(),
	}
}
