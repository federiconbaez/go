package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
)

type Metric struct {
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type MetricsCollector struct {
	metrics     sync.Map
	aggregates  sync.Map
	mu          sync.RWMutex
	collectors  []func() []Metric
	enabled     int32
	flushTicker *time.Ticker
	stopCh      chan struct{}
}

type CounterMetric struct {
	value  int64
	labels map[string]string
}

type GaugeMetric struct {
	value  float64
	labels map[string]string
	mu     sync.RWMutex
}

type HistogramMetric struct {
	buckets map[float64]int64
	sum     float64
	count   int64
	labels  map[string]string
	mu      sync.RWMutex
}

func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		enabled: 1,
		stopCh:  make(chan struct{}),
	}
	
	mc.registerDefaultCollectors()
	mc.startPeriodicFlush()
	
	return mc
}

func (mc *MetricsCollector) registerDefaultCollectors() {
	mc.collectors = append(mc.collectors, mc.collectSystemMetrics)
	mc.collectors = append(mc.collectors, mc.collectRuntimeMetrics)
}

func (mc *MetricsCollector) IncrementCounter(name string, labels map[string]string) {
	if !mc.isEnabled() {
		return
	}
	
	key := mc.buildKey(name, labels)
	
	if existing, ok := mc.metrics.Load(key); ok {
		if counter, ok := existing.(*CounterMetric); ok {
			atomic.AddInt64(&counter.value, 1)
			return
		}
	}
	
	counter := &CounterMetric{
		value:  1,
		labels: mc.copyLabels(labels),
	}
	mc.metrics.Store(key, counter)
}

func (mc *MetricsCollector) AddToCounter(name string, value float64, labels map[string]string) {
	if !mc.isEnabled() {
		return
	}
	
	key := mc.buildKey(name, labels)
	intValue := int64(value)
	
	if existing, ok := mc.metrics.Load(key); ok {
		if counter, ok := existing.(*CounterMetric); ok {
			atomic.AddInt64(&counter.value, intValue)
			return
		}
	}
	
	counter := &CounterMetric{
		value:  intValue,
		labels: mc.copyLabels(labels),
	}
	mc.metrics.Store(key, counter)
}

func (mc *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	if !mc.isEnabled() {
		return
	}
	
	key := mc.buildKey(name, labels)
	
	if existing, ok := mc.metrics.Load(key); ok {
		if gauge, ok := existing.(*GaugeMetric); ok {
			gauge.mu.Lock()
			gauge.value = value
			gauge.mu.Unlock()
			return
		}
	}
	
	gauge := &GaugeMetric{
		value:  value,
		labels: mc.copyLabels(labels),
	}
	mc.metrics.Store(key, gauge)
}

func (mc *MetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	if !mc.isEnabled() {
		return
	}
	
	key := mc.buildKey(name, labels)
	
	if existing, ok := mc.metrics.Load(key); ok {
		if hist, ok := existing.(*HistogramMetric); ok {
			hist.mu.Lock()
			hist.observe(value)
			hist.mu.Unlock()
			return
		}
	}
	
	hist := newHistogramMetric(labels)
	hist.observe(value)
	mc.metrics.Store(key, hist)
}

func newHistogramMetric(labels map[string]string) *HistogramMetric {
	buckets := map[float64]int64{
		0.005: 0, 0.01: 0, 0.025: 0, 0.05: 0, 0.1: 0,
		0.25: 0, 0.5: 0, 1: 0, 2.5: 0, 5: 0, 10: 0,
	}
	
	return &HistogramMetric{
		buckets: buckets,
		labels:  labels,
	}
}

func (h *HistogramMetric) observe(value float64) {
	h.sum += value
	h.count++
	
	for bucket := range h.buckets {
		if value <= bucket {
			h.buckets[bucket]++
		}
	}
}

func (mc *MetricsCollector) TimeDuration(name string, labels map[string]string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start).Seconds()
		mc.ObserveHistogram(name+"_duration_seconds", duration, labels)
	}
}

func (mc *MetricsCollector) collectSystemMetrics() []Metric {
	var metrics []Metric
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	metrics = append(metrics, Metric{
		Name:      "system_memory_alloc_bytes",
		Type:      Gauge,
		Value:     float64(m.Alloc),
		Timestamp: time.Now(),
	})
	
	metrics = append(metrics, Metric{
		Name:      "system_memory_sys_bytes",
		Type:      Gauge,
		Value:     float64(m.Sys),
		Timestamp: time.Now(),
	})
	
	metrics = append(metrics, Metric{
		Name:      "system_gc_total",
		Type:      Counter,
		Value:     float64(m.NumGC),
		Timestamp: time.Now(),
	})
	
	return metrics
}

func (mc *MetricsCollector) collectRuntimeMetrics() []Metric {
	var metrics []Metric
	
	metrics = append(metrics, Metric{
		Name:      "runtime_goroutines_total",
		Type:      Gauge,
		Value:     float64(runtime.NumGoroutine()),
		Timestamp: time.Now(),
	})
	
	metrics = append(metrics, Metric{
		Name:      "runtime_cpu_cores",
		Type:      Gauge,
		Value:     float64(runtime.NumCPU()),
		Timestamp: time.Now(),
	})
	
	return metrics
}

func (mc *MetricsCollector) GetAllMetrics() []Metric {
	var allMetrics []Metric
	now := time.Now()
	
	mc.metrics.Range(func(key, value interface{}) bool {
		switch metric := value.(type) {
		case *CounterMetric:
			allMetrics = append(allMetrics, Metric{
				Name:      mc.extractNameFromKey(key.(string)),
				Type:      Counter,
				Value:     float64(atomic.LoadInt64(&metric.value)),
				Labels:    metric.labels,
				Timestamp: now,
			})
		case *GaugeMetric:
			metric.mu.RLock()
			allMetrics = append(allMetrics, Metric{
				Name:      mc.extractNameFromKey(key.(string)),
				Type:      Gauge,
				Value:     metric.value,
				Labels:    metric.labels,
				Timestamp: now,
			})
			metric.mu.RUnlock()
		case *HistogramMetric:
			metric.mu.RLock()
			allMetrics = append(allMetrics, Metric{
				Name:      mc.extractNameFromKey(key.(string)),
				Type:      Histogram,
				Value:     metric.sum,
				Labels:    metric.labels,
				Timestamp: now,
				Metadata: map[string]interface{}{
					"count":   metric.count,
					"buckets": metric.buckets,
				},
			})
			metric.mu.RUnlock()
		}
		return true
	})
	
	for _, collector := range mc.collectors {
		allMetrics = append(allMetrics, collector()...)
	}
	
	return allMetrics
}

func (mc *MetricsCollector) GetMetricsJSON() ([]byte, error) {
	metrics := mc.GetAllMetrics()
	return json.MarshalIndent(metrics, "", "  ")
}

func (mc *MetricsCollector) RegisterCollector(collector func() []Metric) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.collectors = append(mc.collectors, collector)
}

func (mc *MetricsCollector) startPeriodicFlush() {
	mc.flushTicker = time.NewTicker(30 * time.Second)
	
	go func() {
		for {
			select {
			case <-mc.flushTicker.C:
				mc.flushOldMetrics()
			case <-mc.stopCh:
				return
			}
		}
	}()
}

func (mc *MetricsCollector) flushOldMetrics() {
	cutoff := time.Now().Add(-5 * time.Minute)
	
	mc.aggregates.Range(func(key, value interface{}) bool {
		if timestamp, ok := value.(time.Time); ok && timestamp.Before(cutoff) {
			mc.aggregates.Delete(key)
		}
		return true
	})
}

func (mc *MetricsCollector) buildKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	
	key := name
	for k, v := range labels {
		key += fmt.Sprintf("_%s_%s", k, v)
	}
	return key
}

func (mc *MetricsCollector) extractNameFromKey(key string) string {
	return key
}

func (mc *MetricsCollector) copyLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return make(map[string]string)
	}
	
	copied := make(map[string]string, len(labels))
	for k, v := range labels {
		copied[k] = v
	}
	return copied
}

func (mc *MetricsCollector) Enable() {
	atomic.StoreInt32(&mc.enabled, 1)
}

func (mc *MetricsCollector) Disable() {
	atomic.StoreInt32(&mc.enabled, 0)
}

func (mc *MetricsCollector) isEnabled() bool {
	return atomic.LoadInt32(&mc.enabled) == 1
}

func (mc *MetricsCollector) Stop() {
	if mc.flushTicker != nil {
		mc.flushTicker.Stop()
	}
	close(mc.stopCh)
}

func (mc *MetricsCollector) Reset() {
	mc.metrics = sync.Map{}
	mc.aggregates = sync.Map{}
}