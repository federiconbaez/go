package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[LogLevel]string{
	TRACE: "TRACE",
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	CallerInfo  *CallerInfo            `json:"caller,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Error       *ErrorInfo             `json:"error,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	Component   string                 `json:"component,omitempty"`
	Operation   string                 `json:"operation,omitempty"`
	Environment string                 `json:"environment,omitempty"`
}

type CallerInfo struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

type ErrorInfo struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	StackTrace string `json:"stack_trace,omitempty"`
	Code       string `json:"code,omitempty"`
}

type LoggerConfig struct {
	Level            LogLevel          `json:"level"`
	Format           string            `json:"format"` // "json" or "text"
	Output           io.Writer         `json:"-"`
	EnableCaller     bool              `json:"enable_caller"`
	EnableStackTrace bool              `json:"enable_stack_trace"`
	TimeFormat       string            `json:"time_format"`
	BufferSize       int               `json:"buffer_size"`
	FlushInterval    time.Duration     `json:"flush_interval"`
	DefaultFields    map[string]interface{} `json:"default_fields"`
	Hooks            []LogHook         `json:"-"`
	Async            bool              `json:"async"`
	MaxFileSize      int64             `json:"max_file_size"`
	MaxBackups       int               `json:"max_backups"`
	MaxAge           int               `json:"max_age"`
	Compress         bool              `json:"compress"`
	Environment      string            `json:"environment"`
	ServiceName      string            `json:"service_name"`
	ServiceVersion   string            `json:"service_version"`
}

type LogHook interface {
	Fire(entry *LogEntry) error
	Levels() []LogLevel
}

type MetricsHook struct {
	counters map[LogLevel]*int64
	mu       sync.RWMutex
}

func NewMetricsHook() *MetricsHook {
	counters := make(map[LogLevel]*int64)
	for level := TRACE; level <= FATAL; level++ {
		var counter int64
		counters[level] = &counter
	}
	
	return &MetricsHook{
		counters: counters,
	}
}

func (m *MetricsHook) Fire(entry *LogEntry) error {
	for level, levelName := range levelNames {
		if entry.Level == levelName {
			if counter, exists := m.counters[level]; exists {
				atomic.AddInt64(counter, 1)
			}
			break
		}
	}
	return nil
}

func (m *MetricsHook) Levels() []LogLevel {
	return []LogLevel{TRACE, DEBUG, INFO, WARN, ERROR, FATAL}
}

func (m *MetricsHook) GetCounts() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	counts := make(map[string]int64)
	for level, counter := range m.counters {
		counts[levelNames[level]] = atomic.LoadInt64(counter)
	}
	return counts
}

type FileRotationHook struct {
	filePath    string
	maxSize     int64
	maxBackups  int
	maxAge      int
	compress    bool
	currentFile *os.File
	mu          sync.Mutex
}

func NewFileRotationHook(filePath string, maxSize int64, maxBackups int, maxAge int, compress bool) *FileRotationHook {
	return &FileRotationHook{
		filePath:   filePath,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		compress:   compress,
	}
}

func (f *FileRotationHook) Fire(entry *LogEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.currentFile == nil {
		file, err := os.OpenFile(f.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		f.currentFile = file
	}
	
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	
	_, err = f.currentFile.Write(append(data, '\n'))
	if err != nil {
		return err
	}
	
	return f.rotateIfNeeded()
}

func (f *FileRotationHook) rotateIfNeeded() error {
	if f.currentFile == nil {
		return nil
	}
	
	info, err := f.currentFile.Stat()
	if err != nil {
		return err
	}
	
	if info.Size() >= f.maxSize {
		f.currentFile.Close()
		
		backupName := fmt.Sprintf("%s.%d", f.filePath, time.Now().Unix())
		if err := os.Rename(f.filePath, backupName); err != nil {
			return err
		}
		
		f.currentFile = nil
		
		go f.cleanupOldBackups()
	}
	
	return nil
}

func (f *FileRotationHook) cleanupOldBackups() {
}

func (f *FileRotationHook) Levels() []LogLevel {
	return []LogLevel{TRACE, DEBUG, INFO, WARN, ERROR, FATAL}
}

type StructuredLogger struct {
	config     LoggerConfig
	output     io.Writer
	buffer     chan *LogEntry
	mu         sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
	hooks      []LogHook
	contextual map[string]interface{}
}

func NewStructuredLogger(config LoggerConfig) *StructuredLogger {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.TimeFormat == "" {
		config.TimeFormat = time.RFC3339
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}
	if config.DefaultFields == nil {
		config.DefaultFields = make(map[string]interface{})
	}
	
	logger := &StructuredLogger{
		config:     config,
		output:     config.Output,
		hooks:      config.Hooks,
		contextual: make(map[string]interface{}),
		stopCh:     make(chan struct{}),
	}
	
	if config.Async {
		logger.buffer = make(chan *LogEntry, config.BufferSize)
		logger.startAsyncProcessor()
	}
	
	return logger
}

func (sl *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	
	newLogger := &StructuredLogger{
		config:     sl.config,
		output:     sl.output,
		hooks:      sl.hooks,
		buffer:     sl.buffer,
		stopCh:     sl.stopCh,
		contextual: make(map[string]interface{}),
	}
	
	for k, v := range sl.contextual {
		newLogger.contextual[k] = v
	}
	newLogger.contextual[key] = value
	
	return newLogger
}

func (sl *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	
	newLogger := &StructuredLogger{
		config:     sl.config,
		output:     sl.output,
		hooks:      sl.hooks,
		buffer:     sl.buffer,
		stopCh:     sl.stopCh,
		contextual: make(map[string]interface{}),
	}
	
	for k, v := range sl.contextual {
		newLogger.contextual[k] = v
	}
	for k, v := range fields {
		newLogger.contextual[k] = v
	}
	
	return newLogger
}

func (sl *StructuredLogger) WithContext(ctx context.Context) *StructuredLogger {
	fields := make(map[string]interface{})
	
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields["trace_id"] = traceID
	}
	if spanID := ctx.Value("span_id"); spanID != nil {
		fields["span_id"] = spanID
	}
	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		fields["user_id"] = userID
	}
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		fields["session_id"] = sessionID
	}
	
	return sl.WithFields(fields)
}

func (sl *StructuredLogger) Trace(message string, fields ...map[string]interface{}) {
	sl.log(TRACE, message, fields...)
}

func (sl *StructuredLogger) Debug(message string, fields ...map[string]interface{}) {
	sl.log(DEBUG, message, fields...)
}

func (sl *StructuredLogger) Info(message string, fields ...map[string]interface{}) {
	sl.log(INFO, message, fields...)
}

func (sl *StructuredLogger) Warn(message string, fields ...map[string]interface{}) {
	sl.log(WARN, message, fields...)
}

func (sl *StructuredLogger) Error(message string, err error, fields ...map[string]interface{}) {
	mergedFields := sl.mergeFields(fields...)
	if err != nil {
		if mergedFields == nil {
			mergedFields = make(map[string]interface{})
		}
		mergedFields["error"] = err.Error()
	}
	sl.logWithError(ERROR, message, err, mergedFields)
}

func (sl *StructuredLogger) Fatal(message string, err error, fields ...map[string]interface{}) {
	mergedFields := sl.mergeFields(fields...)
	if err != nil {
		if mergedFields == nil {
			mergedFields = make(map[string]interface{})
		}
		mergedFields["error"] = err.Error()
	}
	sl.logWithError(FATAL, message, err, mergedFields)
	os.Exit(1)
}

func (sl *StructuredLogger) LogWithDuration(level LogLevel, message string, duration time.Duration, fields ...map[string]interface{}) {
	mergedFields := sl.mergeFields(fields...)
	if mergedFields == nil {
		mergedFields = make(map[string]interface{})
	}
	mergedFields["duration_ms"] = duration.Milliseconds()
	sl.log(level, message, mergedFields)
}

func (sl *StructuredLogger) TimeOperation(level LogLevel, message string, operation func()) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		sl.LogWithDuration(level, message, duration)
	}()
	operation()
}

func (sl *StructuredLogger) log(level LogLevel, message string, fields ...map[string]interface{}) {
	sl.logWithError(level, message, nil, sl.mergeFields(fields...))
}

func (sl *StructuredLogger) logWithError(level LogLevel, message string, err error, fields map[string]interface{}) {
	if level < sl.config.Level {
		return
	}
	
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     levelNames[level],
		Message:   message,
		Fields:    sl.buildFields(fields),
	}
	
	if sl.config.EnableCaller {
		entry.CallerInfo = sl.getCallerInfo(3)
	}
	
	if err != nil {
		entry.Error = &ErrorInfo{
			Type:    fmt.Sprintf("%T", err),
			Message: err.Error(),
		}
		
		if sl.config.EnableStackTrace {
			entry.Error.StackTrace = sl.getStackTrace()
		}
	}
	
	sl.writeEntry(entry)
}

func (sl *StructuredLogger) buildFields(fields map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for k, v := range sl.config.DefaultFields {
		result[k] = v
	}
	
	sl.mu.RLock()
	for k, v := range sl.contextual {
		result[k] = v
	}
	sl.mu.RUnlock()
	
	if fields != nil {
		for k, v := range fields {
			result[k] = v
		}
	}
	
	if sl.config.Environment != "" {
		result["environment"] = sl.config.Environment
	}
	if sl.config.ServiceName != "" {
		result["service"] = sl.config.ServiceName
	}
	if sl.config.ServiceVersion != "" {
		result["version"] = sl.config.ServiceVersion
	}
	
	return result
}

func (sl *StructuredLogger) mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	
	result := make(map[string]interface{})
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			result[k] = v
		}
	}
	return result
}

func (sl *StructuredLogger) getCallerInfo(skip int) *CallerInfo {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return nil
	}
	
	fn := runtime.FuncForPC(pc)
	var funcName string
	if fn != nil {
		funcName = fn.Name()
		if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
			funcName = funcName[lastSlash+1:]
		}
	}
	
	if lastSlash := strings.LastIndex(file, "/"); lastSlash >= 0 {
		file = file[lastSlash+1:]
	}
	
	return &CallerInfo{
		File:     file,
		Line:     line,
		Function: funcName,
	}
}

func (sl *StructuredLogger) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (sl *StructuredLogger) writeEntry(entry *LogEntry) {
	if sl.config.Async && sl.buffer != nil {
		select {
		case sl.buffer <- entry:
		default:
		}
		return
	}
	
	sl.processEntry(entry)
}

func (sl *StructuredLogger) processEntry(entry *LogEntry) {
	for _, hook := range sl.hooks {
		if sl.shouldFireHook(hook, entry) {
			if err := hook.Fire(entry); err != nil {
				fmt.Fprintf(os.Stderr, "Hook error: %v\n", err)
			}
		}
	}
	
	var output []byte
	var err error
	
	if sl.config.Format == "json" {
		output, err = json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
			return
		}
		output = append(output, '\n')
	} else {
		output = []byte(sl.formatText(entry))
	}
	
	sl.output.Write(output)
}

func (sl *StructuredLogger) shouldFireHook(hook LogHook, entry *LogEntry) bool {
	hookLevels := hook.Levels()
	for _, level := range hookLevels {
		if levelNames[level] == entry.Level {
			return true
		}
	}
	return false
}

func (sl *StructuredLogger) formatText(entry *LogEntry) string {
	var builder strings.Builder
	
	builder.WriteString(entry.Timestamp.Format(sl.config.TimeFormat))
	builder.WriteString(" [")
	builder.WriteString(entry.Level)
	builder.WriteString("] ")
	builder.WriteString(entry.Message)
	
	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			builder.WriteString(" ")
			builder.WriteString(k)
			builder.WriteString("=")
			builder.WriteString(fmt.Sprintf("%v", v))
		}
	}
	
	if entry.CallerInfo != nil {
		builder.WriteString(" (")
		builder.WriteString(entry.CallerInfo.File)
		builder.WriteString(":")
		builder.WriteString(fmt.Sprintf("%d", entry.CallerInfo.Line))
		builder.WriteString(")")
	}
	
	builder.WriteString("\n")
	return builder.String()
}

func (sl *StructuredLogger) startAsyncProcessor() {
	sl.wg.Add(1)
	
	go func() {
		defer sl.wg.Done()
		
		ticker := time.NewTicker(sl.config.FlushInterval)
		defer ticker.Stop()
		
		batch := make([]*LogEntry, 0, 100)
		
		for {
			select {
			case entry := <-sl.buffer:
				batch = append(batch, entry)
				if len(batch) >= 100 {
					sl.processBatch(batch)
					batch = batch[:0]
				}
				
			case <-ticker.C:
				if len(batch) > 0 {
					sl.processBatch(batch)
					batch = batch[:0]
				}
				
			case <-sl.stopCh:
				if len(batch) > 0 {
					sl.processBatch(batch)
				}
				return
			}
		}
	}()
}

func (sl *StructuredLogger) processBatch(batch []*LogEntry) {
	for _, entry := range batch {
		sl.processEntry(entry)
	}
}

func (sl *StructuredLogger) Flush() {
	if sl.config.Async && sl.buffer != nil {
		timeout := time.After(5 * time.Second)
		for {
			select {
			case <-timeout:
				return
			default:
				if len(sl.buffer) == 0 {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

func (sl *StructuredLogger) Close() {
	if sl.config.Async {
		close(sl.stopCh)
		sl.wg.Wait()
		close(sl.buffer)
	}
}

func (sl *StructuredLogger) SetLevel(level LogLevel) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.config.Level = level
}

func (sl *StructuredLogger) GetLevel() LogLevel {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.config.Level
}

func (sl *StructuredLogger) AddHook(hook LogHook) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.hooks = append(sl.hooks, hook)
}