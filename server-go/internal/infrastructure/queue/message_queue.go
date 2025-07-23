package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrQueueFull       = errors.New("queue is full")
	ErrConsumerStopped = errors.New("consumer stopped")
	ErrInvalidMessage  = errors.New("invalid message")
	ErrRetryExceeded   = errors.New("max retries exceeded")
)

type MessagePriority int

const (
	PriorityLow    MessagePriority = 0
	PriorityNormal MessagePriority = 1
	PriorityHigh   MessagePriority = 2
	PriorityCritical MessagePriority = 3
)

type MessageStatus string

const (
	StatusPending    MessageStatus = "pending"
	StatusProcessing MessageStatus = "processing"
	StatusCompleted  MessageStatus = "completed"
	StatusFailed     MessageStatus = "failed"
	StatusRetrying   MessageStatus = "retrying"
	StatusDead       MessageStatus = "dead"
)

type Message struct {
	ID          string                 `json:"id"`
	Topic       string                 `json:"topic"`
	Payload     interface{}            `json:"payload"`
	Headers     map[string]string      `json:"headers"`
	Priority    MessagePriority        `json:"priority"`
	Status      MessageStatus          `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	DelayUntil  *time.Time             `json:"delay_until,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func (m *Message) IsExpired(ttl time.Duration) bool {
	return time.Since(m.CreatedAt) > ttl
}

func (m *Message) CanRetry() bool {
	return m.RetryCount < m.MaxRetries
}

func (m *Message) ShouldProcess() bool {
	if m.DelayUntil != nil && time.Now().Before(*m.DelayUntil) {
		return false
	}
	return m.Status == StatusPending || m.Status == StatusRetrying
}

type MessageHandler func(ctx context.Context, msg *Message) error

type RetryStrategy interface {
	NextDelay(attempt int) time.Duration
	ShouldRetry(err error, attempt int) bool
}

type ExponentialBackoffStrategy struct {
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	MaxRetries   int
	RetryOnError func(error) bool
}

func (ebs *ExponentialBackoffStrategy) NextDelay(attempt int) time.Duration {
	delay := time.Duration(float64(ebs.BaseDelay) * pow(ebs.Multiplier, float64(attempt)))
	if delay > ebs.MaxDelay {
		delay = ebs.MaxDelay
	}
	return delay
}

func (ebs *ExponentialBackoffStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= ebs.MaxRetries {
		return false
	}
	
	if ebs.RetryOnError != nil {
		return ebs.RetryOnError(err)
	}
	
	return true
}

func pow(base float64, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

type QueueConfig struct {
	MaxSize        int                    `json:"max_size"`
	Workers        int                    `json:"workers"`
	BatchSize      int                    `json:"batch_size"`
	PollInterval   time.Duration          `json:"poll_interval"`
	RetryStrategy  RetryStrategy          `json:"-"`
	DeadLetterTTL  time.Duration          `json:"dead_letter_ttl"`
	EnableMetrics  bool                   `json:"enable_metrics"`
}

type QueueMetrics struct {
	TotalMessages     int64 `json:"total_messages"`
	ProcessedMessages int64 `json:"processed_messages"`
	FailedMessages    int64 `json:"failed_messages"`
	RetryMessages     int64 `json:"retry_messages"`
	DeadMessages      int64 `json:"dead_messages"`
	CurrentSize       int64 `json:"current_size"`
	Workers           int   `json:"workers"`
	ActiveWorkers     int32 `json:"active_workers"`
}

type MessageQueue struct {
	config       QueueConfig
	messages     chan *Message
	dlq          chan *Message
	handlers     map[string]MessageHandler
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	metrics      QueueMetrics
	activeWorkers int32
	
	// Event callbacks
	onMessage    func(*Message)
	onProcessed  func(*Message, error)
	onRetry      func(*Message, error)
	onDead       func(*Message)
}

func NewMessageQueue(config QueueConfig) *MessageQueue {
	if config.MaxSize <= 0 {
		config.MaxSize = 1000
	}
	if config.Workers <= 0 {
		config.Workers = 5
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 10
	}
	if config.PollInterval <= 0 {
		config.PollInterval = time.Second
	}
	if config.RetryStrategy == nil {
		config.RetryStrategy = &ExponentialBackoffStrategy{
			BaseDelay:  time.Second,
			MaxDelay:   time.Minute * 5,
			Multiplier: 2.0,
			MaxRetries: 3,
		}
	}
	if config.DeadLetterTTL <= 0 {
		config.DeadLetterTTL = time.Hour * 24
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	mq := &MessageQueue{
		config:   config,
		messages: make(chan *Message, config.MaxSize),
		dlq:      make(chan *Message, config.MaxSize/10),
		handlers: make(map[string]MessageHandler),
		ctx:      ctx,
		cancel:   cancel,
		metrics:  QueueMetrics{Workers: config.Workers},
	}
	
	mq.startWorkers()
	mq.startDLQProcessor()
	
	return mq
}

func (mq *MessageQueue) Publish(ctx context.Context, topic string, payload interface{}, options ...PublishOption) error {
	msg := &Message{
		ID:         generateID(),
		Topic:      topic,
		Payload:    payload,
		Headers:    make(map[string]string),
		Priority:   PriorityNormal,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
		MaxRetries: 3,
		Metadata:   make(map[string]interface{}),
	}
	
	for _, option := range options {
		option(msg)
	}
	
	select {
	case mq.messages <- msg:
		atomic.AddInt64(&mq.metrics.TotalMessages, 1)
		atomic.AddInt64(&mq.metrics.CurrentSize, 1)
		
		if mq.onMessage != nil {
			mq.onMessage(msg)
		}
		
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

type PublishOption func(*Message)

func WithPriority(priority MessagePriority) PublishOption {
	return func(m *Message) {
		m.Priority = priority
	}
}

func WithHeaders(headers map[string]string) PublishOption {
	return func(m *Message) {
		for k, v := range headers {
			m.Headers[k] = v
		}
	}
}

func WithMaxRetries(retries int) PublishOption {
	return func(m *Message) {
		m.MaxRetries = retries
	}
}

func WithDelay(delay time.Duration) PublishOption {
	return func(m *Message) {
		delayUntil := time.Now().Add(delay)
		m.DelayUntil = &delayUntil
	}
}

func WithMetadata(metadata map[string]interface{}) PublishOption {
	return func(m *Message) {
		for k, v := range metadata {
			m.Metadata[k] = v
		}
	}
}

func (mq *MessageQueue) Subscribe(topic string, handler MessageHandler) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.handlers[topic] = handler
}

func (mq *MessageQueue) Unsubscribe(topic string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	delete(mq.handlers, topic)
}

func (mq *MessageQueue) startWorkers() {
	for i := 0; i < mq.config.Workers; i++ {
		mq.wg.Add(1)
		go mq.worker(i)
	}
}

func (mq *MessageQueue) worker(id int) {
	defer mq.wg.Done()
	
	batch := make([]*Message, 0, mq.config.BatchSize)
	ticker := time.NewTicker(mq.config.PollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mq.ctx.Done():
			return
			
		case msg := <-mq.messages:
			atomic.AddInt32(&mq.activeWorkers, 1)
			
			if msg.ShouldProcess() {
				batch = append(batch, msg)
				atomic.AddInt64(&mq.metrics.CurrentSize, -1)
			} else {
				mq.scheduleRetry(msg)
				atomic.AddInt32(&mq.activeWorkers, -1)
				continue
			}
			
			if len(batch) >= mq.config.BatchSize {
				mq.processBatch(batch)
				batch = batch[:0]
			}
			
			atomic.AddInt32(&mq.activeWorkers, -1)
			
		case <-ticker.C:
			if len(batch) > 0 {
				atomic.AddInt32(&mq.activeWorkers, 1)
				mq.processBatch(batch)
				batch = batch[:0]
				atomic.AddInt32(&mq.activeWorkers, -1)
			}
		}
	}
}

func (mq *MessageQueue) processBatch(batch []*Message) {
	for _, msg := range batch {
		mq.processMessage(msg)
	}
}

func (mq *MessageQueue) processMessage(msg *Message) {
	mq.mu.RLock()
	handler, exists := mq.handlers[msg.Topic]
	mq.mu.RUnlock()
	
	if !exists {
		msg.Status = StatusFailed
		atomic.AddInt64(&mq.metrics.FailedMessages, 1)
		
		if mq.onProcessed != nil {
			mq.onProcessed(msg, fmt.Errorf("no handler for topic: %s", msg.Topic))
		}
		return
	}
	
	msg.Status = StatusProcessing
	now := time.Now()
	msg.ProcessedAt = &now
	
	ctx, cancel := context.WithCancel(mq.ctx)
	defer cancel()
	
	err := handler(ctx, msg)
	
	if err != nil {
		mq.handleProcessingError(msg, err)
	} else {
		msg.Status = StatusCompleted
		atomic.AddInt64(&mq.metrics.ProcessedMessages, 1)
		
		if mq.onProcessed != nil {
			mq.onProcessed(msg, nil)
		}
	}
}

func (mq *MessageQueue) handleProcessingError(msg *Message, err error) {
	if mq.config.RetryStrategy.ShouldRetry(err, msg.RetryCount) && msg.CanRetry() {
		msg.RetryCount++
		msg.Status = StatusRetrying
		
		delay := mq.config.RetryStrategy.NextDelay(msg.RetryCount)
		retryTime := time.Now().Add(delay)
		msg.DelayUntil = &retryTime
		
		atomic.AddInt64(&mq.metrics.RetryMessages, 1)
		
		if mq.onRetry != nil {
			mq.onRetry(msg, err)
		}
		
		mq.scheduleRetry(msg)
	} else {
		msg.Status = StatusDead
		atomic.AddInt64(&mq.metrics.DeadMessages, 1)
		
		select {
		case mq.dlq <- msg:
			if mq.onDead != nil {
				mq.onDead(msg)
			}
		default:
		}
		
		atomic.AddInt64(&mq.metrics.FailedMessages, 1)
		
		if mq.onProcessed != nil {
			mq.onProcessed(msg, ErrRetryExceeded)
		}
	}
}

func (mq *MessageQueue) scheduleRetry(msg *Message) {
	if msg.DelayUntil == nil {
		select {
		case mq.messages <- msg:
			atomic.AddInt64(&mq.metrics.CurrentSize, 1)
		default:
		}
		return
	}
	
	delay := time.Until(*msg.DelayUntil)
	if delay <= 0 {
		select {
		case mq.messages <- msg:
			atomic.AddInt64(&mq.metrics.CurrentSize, 1)
		default:
		}
		return
	}
	
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		
		select {
		case <-timer.C:
			select {
			case mq.messages <- msg:
				atomic.AddInt64(&mq.metrics.CurrentSize, 1)
			case <-mq.ctx.Done():
			}
		case <-mq.ctx.Done():
		}
	}()
}

func (mq *MessageQueue) startDLQProcessor() {
	mq.wg.Add(1)
	
	go func() {
		defer mq.wg.Done()
		
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		
		for {
			select {
			case <-mq.ctx.Done():
				return
				
			case msg := <-mq.dlq:
				if msg.IsExpired(mq.config.DeadLetterTTL) {
					continue
				}
				
			case <-ticker.C:
				mq.cleanupExpiredDLQMessages()
			}
		}
	}()
}

func (mq *MessageQueue) cleanupExpiredDLQMessages() {
	for {
		select {
		case msg := <-mq.dlq:
			if !msg.IsExpired(mq.config.DeadLetterTTL) {
				mq.dlq <- msg
				return
			}
		default:
			return
		}
	}
}

func (mq *MessageQueue) GetMetrics() QueueMetrics {
	mq.metrics.ActiveWorkers = atomic.LoadInt32(&mq.activeWorkers)
	return mq.metrics
}

func (mq *MessageQueue) GetSize() int {
	return int(atomic.LoadInt64(&mq.metrics.CurrentSize))
}

func (mq *MessageQueue) GetDLQSize() int {
	return len(mq.dlq)
}

func (mq *MessageQueue) OnMessage(callback func(*Message)) {
	mq.onMessage = callback
}

func (mq *MessageQueue) OnProcessed(callback func(*Message, error)) {
	mq.onProcessed = callback
}

func (mq *MessageQueue) OnRetry(callback func(*Message, error)) {
	mq.onRetry = callback
}

func (mq *MessageQueue) OnDead(callback func(*Message)) {
	mq.onDead = callback
}

func (mq *MessageQueue) Stop() {
	mq.cancel()
	mq.wg.Wait()
	close(mq.messages)
	close(mq.dlq)
}

func (mq *MessageQueue) GetHandlers() []string {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	
	var topics []string
	for topic := range mq.handlers {
		topics = append(topics, topic)
	}
	return topics
}

func generateID() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Int31())
}

func (mq *MessageQueue) DrainDLQ() []*Message {
	var messages []*Message
	
	for {
		select {
		case msg := <-mq.dlq:
			messages = append(messages, msg)
		default:
			return messages
		}
	}
}

func (mq *MessageQueue) RequeueFromDLQ(messageID string) error {
	messages := mq.DrainDLQ()
	var targetMessage *Message
	
	for _, msg := range messages {
		if msg.ID == messageID {
			targetMessage = msg
		} else {
			mq.dlq <- msg
		}
	}
	
	if targetMessage == nil {
		return ErrInvalidMessage
	}
	
	targetMessage.Status = StatusPending
	targetMessage.RetryCount = 0
	targetMessage.DelayUntil = nil
	
	select {
	case mq.messages <- targetMessage:
		atomic.AddInt64(&mq.metrics.CurrentSize, 1)
		atomic.AddInt64(&mq.metrics.DeadMessages, -1)
		return nil
	default:
		mq.dlq <- targetMessage
		return ErrQueueFull
	}
}