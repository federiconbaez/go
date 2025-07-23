package com.example.notebook.data.analytics

import android.content.Context
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import java.util.*
import java.util.concurrent.ConcurrentLinkedQueue
import kotlin.random.Random

class AnalyticsTracker private constructor(
    private val context: Context,
    private val scope: CoroutineScope
) {
    
    companion object {
        @Volatile
        private var INSTANCE: AnalyticsTracker? = null
        
        fun getInstance(context: Context, scope: CoroutineScope): AnalyticsTracker {
            return INSTANCE ?: synchronized(this) {
                INSTANCE ?: AnalyticsTracker(context.applicationContext, scope).also { 
                    INSTANCE = it 
                }
            }
        }
        
        private const val BATCH_SIZE = 10
        private const val FLUSH_INTERVAL_MS = 30000L
    }
    
    data class AnalyticsEvent(
        val id: String = UUID.randomUUID().toString(),
        val name: String,
        val category: String,
        val properties: Map<String, Any> = emptyMap(),
        val timestamp: Long = System.currentTimeMillis(),
        val sessionId: String,
        val userId: String? = null
    )
    
    data class UserSession(
        val sessionId: String = UUID.randomUUID().toString(),
        val startTime: Long = System.currentTimeMillis(),
        var lastActivity: Long = System.currentTimeMillis(),
        var eventCount: Int = 0
    )
    
    private val eventQueue = ConcurrentLinkedQueue<AnalyticsEvent>()
    private val _eventFlow = MutableSharedFlow<AnalyticsEvent>()
    val eventFlow: SharedFlow<AnalyticsEvent> = _eventFlow.asSharedFlow()
    
    private var currentSession = UserSession()
    private var isEnabled = true
    
    init {
        startPeriodicFlush()
    }
    
    fun trackEvent(
        name: String,
        category: String = "general",
        properties: Map<String, Any> = emptyMap(),
        userId: String? = null
    ) {
        if (!isEnabled) return
        
        updateSession()
        
        val event = AnalyticsEvent(
            name = name,
            category = category,
            properties = properties + getContextualProperties(),
            sessionId = currentSession.sessionId,
            userId = userId
        )
        
        eventQueue.offer(event)
        scope.launch {
            _eventFlow.emit(event)
        }
        
        if (eventQueue.size >= BATCH_SIZE) {
            scope.launch { flushEvents() }
        }
    }
    
    fun trackUserInteraction(action: String, target: String, metadata: Map<String, Any> = emptyMap()) {
        trackEvent(
            name = "user_interaction",
            category = "ui",
            properties = mapOf(
                "action" to action,
                "target" to target,
                "interaction_id" to Random.nextInt(10000, 99999)
            ) + metadata
        )
    }
    
    fun trackScreenView(screenName: String, timeSpent: Long? = null) {
        val properties = mutableMapOf<String, Any>(
            "screen_name" to screenName,
            "view_id" to Random.nextInt(10000, 99999)
        )
        
        timeSpent?.let { properties["time_spent_ms"] = it }
        
        trackEvent(
            name = "screen_view",
            category = "navigation",
            properties = properties
        )
    }
    
    fun trackError(
        errorType: String,
        errorMessage: String,
        stackTrace: String? = null,
        context: Map<String, Any> = emptyMap()
    ) {
        trackEvent(
            name = "error_occurred",
            category = "error",
            properties = mapOf(
                "error_type" to errorType,
                "error_message" to errorMessage,
                "stack_trace" to (stackTrace ?: "not_available"),
                "error_id" to Random.nextInt(10000, 99999)
            ) + context
        )
    }
    
    fun trackPerformance(
        operation: String,
        duration: Long,
        success: Boolean,
        additionalMetrics: Map<String, Any> = emptyMap()
    ) {
        trackEvent(
            name = "performance_metric",
            category = "performance",
            properties = mapOf(
                "operation" to operation,
                "duration_ms" to duration,
                "success" to success,
                "performance_id" to Random.nextInt(10000, 99999)
            ) + additionalMetrics
        )
    }
    
    private fun updateSession() {
        currentSession.lastActivity = System.currentTimeMillis()
        currentSession.eventCount++
        
        val sessionDuration = currentSession.lastActivity - currentSession.startTime
        if (sessionDuration > 1800000) { // 30 minutes
            startNewSession()
        }
    }
    
    private fun startNewSession() {
        currentSession = UserSession()
    }
    
    private fun getContextualProperties(): Map<String, Any> {
        return mapOf(
            "app_version" to "1.0.0",
            "platform" to "android",
            "session_duration" to (currentSession.lastActivity - currentSession.startTime),
            "session_event_count" to currentSession.eventCount,
            "device_id" to android.provider.Settings.Secure.getString(
                context.contentResolver,
                android.provider.Settings.Secure.ANDROID_ID
            )
        )
    }
    
    private fun startPeriodicFlush() {
        scope.launch {
            while (isActive) {
                delay(FLUSH_INTERVAL_MS)
                flushEvents()
            }
        }
    }
    
    private suspend fun flushEvents() {
        val events = mutableListOf<AnalyticsEvent>()
        while (events.size < BATCH_SIZE && eventQueue.isNotEmpty()) {
            eventQueue.poll()?.let { events.add(it) }
        }
        
        if (events.isNotEmpty()) {
            try {
                processEventBatch(events)
            } catch (e: Exception) {
                events.forEach { eventQueue.offer(it) }
            }
        }
    }
    
    private suspend fun processEventBatch(events: List<AnalyticsEvent>) {
        withContext(Dispatchers.IO) {
            events.forEach { event ->
                println("Analytics Event: ${event.name} | Category: ${event.category}")
            }
        }
    }
    
    fun getSessionInfo(): UserSession = currentSession
    
    fun setEnabled(enabled: Boolean) {
        isEnabled = enabled
    }
    
    suspend fun flush() {
        flushEvents()
    }
}