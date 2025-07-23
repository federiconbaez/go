package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrKeyExpired  = errors.New("key expired")
	ErrCacheFull   = errors.New("cache is full")
)

type CacheEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	ExpiresAt  time.Time   `json:"expires_at"`
	CreatedAt  time.Time   `json:"created_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	AccessCount int64      `json:"access_count"`
	Size       int         `json:"size"`
}

func (e *CacheEntry) IsExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt)
}

func (e *CacheEntry) Touch() {
	e.AccessedAt = time.Now()
	e.AccessCount++
}

type CacheStats struct {
	TotalKeys     int           `json:"total_keys"`
	HitCount      int64         `json:"hit_count"`
	MissCount     int64         `json:"miss_count"`
	EvictionCount int64         `json:"eviction_count"`
	TotalSize     int           `json:"total_size"`
	HitRatio      float64       `json:"hit_ratio"`
	AvgAccessTime time.Duration `json:"avg_access_time"`
}

type EvictionPolicy string

const (
	LRU   EvictionPolicy = "lru"   // Least Recently Used
	LFU   EvictionPolicy = "lfu"   // Least Frequently Used
	FIFO  EvictionPolicy = "fifo"  // First In First Out
	TTL   EvictionPolicy = "ttl"   // Time To Live based
)

type CacheConfig struct {
	MaxSize        int            `json:"max_size"`
	MaxMemory      int            `json:"max_memory"` // bytes
	DefaultTTL     time.Duration  `json:"default_ttl"`
	EvictionPolicy EvictionPolicy `json:"eviction_policy"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	EnableMetrics  bool           `json:"enable_metrics"`
}

type DistributedCache struct {
	entries  map[string]*CacheEntry
	mu       sync.RWMutex
	config   CacheConfig
	stats    CacheStats
	stopCh   chan struct{}
	
	// Event handlers
	onSet    func(key string, value interface{})
	onGet    func(key string, hit bool)
	onDelete func(key string)
	onEvict  func(key string, reason string)
}

func NewDistributedCache(config CacheConfig) *DistributedCache {
	if config.MaxSize <= 0 {
		config.MaxSize = 10000
	}
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = time.Hour
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.EvictionPolicy == "" {
		config.EvictionPolicy = LRU
	}
	
	cache := &DistributedCache{
		entries: make(map[string]*CacheEntry),
		config:  config,
		stopCh:  make(chan struct{}),
	}
	
	cache.startCleanupRoutine()
	return cache
}

func (dc *DistributedCache) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	if dc.needsEviction() {
		if err := dc.evictEntries(1); err != nil {
			return err
		}
	}
	
	expiration := time.Time{}
	if len(ttl) > 0 && ttl[0] > 0 {
		expiration = time.Now().Add(ttl[0])
	} else if dc.config.DefaultTTL > 0 {
		expiration = time.Now().Add(dc.config.DefaultTTL)
	}
	
	serialized, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}
	
	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		ExpiresAt:   expiration,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		Size:        len(serialized),
	}
	
	dc.entries[key] = entry
	
	if dc.onSet != nil {
		dc.onSet(key, value)
	}
	
	return nil
}

func (dc *DistributedCache) Get(ctx context.Context, key string) (interface{}, error) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	entry, exists := dc.entries[key]
	if !exists {
		dc.stats.MissCount++
		if dc.onGet != nil {
			dc.onGet(key, false)
		}
		return nil, ErrKeyNotFound
	}
	
	if entry.IsExpired() {
		delete(dc.entries, key)
		dc.stats.MissCount++
		if dc.onGet != nil {
			dc.onGet(key, false)
		}
		return nil, ErrKeyExpired
	}
	
	entry.Touch()
	dc.stats.HitCount++
	
	if dc.onGet != nil {
		dc.onGet(key, true)
	}
	
	return entry.Value, nil
}

func (dc *DistributedCache) GetWithInfo(ctx context.Context, key string) (*CacheEntry, error) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	entry, exists := dc.entries[key]
	if !exists {
		return nil, ErrKeyNotFound
	}
	
	if entry.IsExpired() {
		return nil, ErrKeyExpired
	}
	
	entryCopy := *entry
	return &entryCopy, nil
}

func (dc *DistributedCache) Delete(ctx context.Context, key string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	if _, exists := dc.entries[key]; !exists {
		return ErrKeyNotFound
	}
	
	delete(dc.entries, key)
	
	if dc.onDelete != nil {
		dc.onDelete(key)
	}
	
	return nil
}

func (dc *DistributedCache) SetWithCallback(
	ctx context.Context,
	key string,
	value interface{},
	ttl time.Duration,
	callback func(key string, expired bool),
) error {
	if err := dc.Set(ctx, key, value, ttl); err != nil {
		return err
	}
	
	if callback != nil && ttl > 0 {
		go func() {
			timer := time.NewTimer(ttl)
			defer timer.Stop()
			
			select {
			case <-timer.C:
				callback(key, true)
			case <-ctx.Done():
				callback(key, false)
			}
		}()
	}
	
	return nil
}

func (dc *DistributedCache) GetOrSet(
	ctx context.Context,
	key string,
	setter func() (interface{}, error),
	ttl ...time.Duration,
) (interface{}, error) {
	value, err := dc.Get(ctx, key)
	if err == nil {
		return value, nil
	}
	
	if err != ErrKeyNotFound && err != ErrKeyExpired {
		return nil, err
	}
	
	newValue, err := setter()
	if err != nil {
		return nil, fmt.Errorf("setter function failed: %w", err)
	}
	
	if err := dc.Set(ctx, key, newValue, ttl...); err != nil {
		return newValue, err
	}
	
	return newValue, nil
}

func (dc *DistributedCache) Keys(pattern string) []string {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	var keys []string
	for key := range dc.entries {
		if pattern == "" || dc.matchPattern(key, pattern) {
			keys = append(keys, key)
		}
	}
	return keys
}

func (dc *DistributedCache) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.entries = make(map[string]*CacheEntry)
	dc.stats = CacheStats{}
}

func (dc *DistributedCache) Size() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.entries)
}

func (dc *DistributedCache) Stats() CacheStats {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	stats := dc.stats
	stats.TotalKeys = len(dc.entries)
	
	if stats.HitCount+stats.MissCount > 0 {
		stats.HitRatio = float64(stats.HitCount) / float64(stats.HitCount+stats.MissCount)
	}
	
	totalSize := 0
	for _, entry := range dc.entries {
		totalSize += entry.Size
	}
	stats.TotalSize = totalSize
	
	return stats
}

func (dc *DistributedCache) needsEviction() bool {
	if dc.config.MaxSize > 0 && len(dc.entries) >= dc.config.MaxSize {
		return true
	}
	
	if dc.config.MaxMemory > 0 {
		totalSize := 0
		for _, entry := range dc.entries {
			totalSize += entry.Size
		}
		return totalSize >= dc.config.MaxMemory
	}
	
	return false
}

func (dc *DistributedCache) evictEntries(count int) error {
	if len(dc.entries) == 0 {
		return nil
	}
	
	var keysToEvict []string
	
	switch dc.config.EvictionPolicy {
	case LRU:
		keysToEvict = dc.getLRUKeys(count)
	case LFU:
		keysToEvict = dc.getLFUKeys(count)
	case FIFO:
		keysToEvict = dc.getFIFOKeys(count)
	case TTL:
		keysToEvict = dc.getExpiredKeys()
		if len(keysToEvict) < count {
			keysToEvict = append(keysToEvict, dc.getLRUKeys(count-len(keysToEvict))...)
		}
	default:
		keysToEvict = dc.getLRUKeys(count)
	}
	
	for _, key := range keysToEvict {
		delete(dc.entries, key)
		dc.stats.EvictionCount++
		if dc.onEvict != nil {
			dc.onEvict(key, string(dc.config.EvictionPolicy))
		}
	}
	
	return nil
}

func (dc *DistributedCache) getLRUKeys(count int) []string {
	type keyTime struct {
		key  string
		time time.Time
	}
	
	var items []keyTime
	for key, entry := range dc.entries {
		items = append(items, keyTime{key: key, time: entry.AccessedAt})
	}
	
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].time.After(items[j].time) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	
	var keys []string
	for i := 0; i < count && i < len(items); i++ {
		keys = append(keys, items[i].key)
	}
	
	return keys
}

func (dc *DistributedCache) getLFUKeys(count int) []string {
	type keyCount struct {
		key   string
		count int64
	}
	
	var items []keyCount
	for key, entry := range dc.entries {
		items = append(items, keyCount{key: key, count: entry.AccessCount})
	}
	
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].count > items[j].count {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	
	var keys []string
	for i := 0; i < count && i < len(items); i++ {
		keys = append(keys, items[i].key)
	}
	
	return keys
}

func (dc *DistributedCache) getFIFOKeys(count int) []string {
	type keyTime struct {
		key  string
		time time.Time
	}
	
	var items []keyTime
	for key, entry := range dc.entries {
		items = append(items, keyTime{key: key, time: entry.CreatedAt})
	}
	
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].time.After(items[j].time) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	
	var keys []string
	for i := 0; i < count && i < len(items); i++ {
		keys = append(keys, items[i].key)
	}
	
	return keys
}

func (dc *DistributedCache) getExpiredKeys() []string {
	var keys []string
	now := time.Now()
	
	for key, entry := range dc.entries {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			keys = append(keys, key)
		}
	}
	
	return keys
}

func (dc *DistributedCache) startCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(dc.config.CleanupInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				dc.cleanup()
			case <-dc.stopCh:
				return
			}
		}
	}()
}

func (dc *DistributedCache) cleanup() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	expiredKeys := dc.getExpiredKeys()
	for _, key := range expiredKeys {
		delete(dc.entries, key)
		if dc.onEvict != nil {
			dc.onEvict(key, "expired")
		}
	}
}

func (dc *DistributedCache) matchPattern(key, pattern string) bool {
	return key == pattern
}

func (dc *DistributedCache) Hash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (dc *DistributedCache) OnSet(handler func(key string, value interface{})) {
	dc.onSet = handler
}

func (dc *DistributedCache) OnGet(handler func(key string, hit bool)) {
	dc.onGet = handler
}

func (dc *DistributedCache) OnDelete(handler func(key string)) {
	dc.onDelete = handler
}

func (dc *DistributedCache) OnEvict(handler func(key string, reason string)) {
	dc.onEvict = handler
}

func (dc *DistributedCache) Stop() {
	close(dc.stopCh)
}