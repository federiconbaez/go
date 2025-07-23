package com.example.notebook.presentation.state

import kotlinx.coroutines.flow.*
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import java.util.*
import java.util.concurrent.ConcurrentHashMap

interface StateManager<S : Any, A : Any> {
    val state: StateFlow<S>
    suspend fun dispatch(action: A)
    fun getCurrentState(): S
}

abstract class BaseStateManager<S : Any, A : Any>(
    initialState: S
) : StateManager<S, A> {
    
    private val _state = MutableStateFlow(initialState)
    override val state: StateFlow<S> = _state.asStateFlow()
    
    private val mutex = Mutex()
    private val actionHistory = mutableListOf<TimestampedAction<A>>()
    private val stateHistory = mutableListOf<TimestampedState<S>>()
    private val maxHistorySize = 50
    
    data class TimestampedAction<A>(
        val action: A,
        val timestamp: Long = System.currentTimeMillis(),
        val id: String = UUID.randomUUID().toString()
    )
    
    data class TimestampedState<S>(
        val state: S,
        val timestamp: Long = System.currentTimeMillis(),
        val actionId: String? = null
    )
    
    abstract suspend fun reduce(currentState: S, action: A): S
    
    open suspend fun onBeforeStateChange(currentState: S, action: A, newState: S) {}
    open suspend fun onAfterStateChange(previousState: S, action: A, newState: S) {}
    
    override suspend fun dispatch(action: A) = mutex.withLock {
        val currentState = _state.value
        val timestampedAction = TimestampedAction(action)
        
        try {
            val newState = reduce(currentState, action)
            
            onBeforeStateChange(currentState, action, newState)
            
            _state.value = newState
            
            recordHistory(timestampedAction, currentState, newState)
            
            onAfterStateChange(currentState, action, newState)
            
        } catch (e: Exception) {
            handleStateError(currentState, action, e)
        }
    }
    
    override fun getCurrentState(): S = _state.value
    
    private fun recordHistory(
        action: TimestampedAction<A>,
        previousState: S,
        newState: S
    ) {
        actionHistory.add(action)
        stateHistory.add(TimestampedState(newState, actionId = action.id))
        
        if (actionHistory.size > maxHistorySize) {
            actionHistory.removeAt(0)
        }
        if (stateHistory.size > maxHistorySize) {
            stateHistory.removeAt(0)
        }
    }
    
    protected open suspend fun handleStateError(
        currentState: S,
        action: A,
        error: Exception
    ) {
        println("State transition error: ${error.message}")
    }
    
    fun getActionHistory(): List<TimestampedAction<A>> = actionHistory.toList()
    fun getStateHistory(): List<TimestampedState<S>> = stateHistory.toList()
    
    suspend fun rollback(steps: Int = 1): Boolean = mutex.withLock {
        if (stateHistory.size < steps + 1) return false
        
        val targetStateIndex = stateHistory.size - steps - 1
        val targetState = stateHistory[targetStateIndex].state
        _state.value = targetState
        
        repeat(steps) {
            if (stateHistory.isNotEmpty()) stateHistory.removeAt(stateHistory.size - 1)
            if (actionHistory.isNotEmpty()) actionHistory.removeAt(actionHistory.size - 1)
        }
        
        return true
    }
}

class CompositeStateManager<S : Any, A : Any>(
    initialState: S,
    private val middlewares: List<StateMiddleware<S, A>> = emptyList()
) : BaseStateManager<S, A>(initialState) {
    
    interface StateMiddleware<S, A> {
        suspend fun process(
            currentState: S,
            action: A,
            next: suspend (A) -> S
        ): S
    }
    
    private val reducers = ConcurrentHashMap<Class<out A>, suspend (S, A) -> S>()
    
    fun <T : A> registerReducer(
        actionClass: Class<T>,
        reducer: suspend (S, T) -> S
    ) {
        reducers[actionClass] = { state, action ->
            @Suppress("UNCHECKED_CAST")
            reducer(state, action as T)
        }
    }
    
    inline fun <reified T : A> registerReducer(
        noinline reducer: suspend (S, T) -> S
    ) {
        registerReducer(T::class.java, reducer)
    }
    
    override suspend fun reduce(currentState: S, action: A): S {
        val reducer = reducers[action::class.java]
            ?: throw IllegalArgumentException("No reducer found for action: ${action::class.simpleName}")
        
        return processWithMiddlewares(currentState, action) { processedAction ->
            reducer(currentState, processedAction)
        }
    }
    
    private suspend fun processWithMiddlewares(
        currentState: S,
        action: A,
        finalReducer: suspend (A) -> S
    ): S {
        if (middlewares.isEmpty()) {
            return finalReducer(action)
        }
        
        var index = 0
        
        suspend fun next(processedAction: A): S {
            return if (index < middlewares.size) {
                val middleware = middlewares[index++]
                middleware.process(currentState, processedAction, ::next)
            } else {
                finalReducer(processedAction)
            }
        }
        
        return next(action)
    }
}

class AsyncStateManager<S : Any, A : Any>(
    initialState: S,
    private val asyncReducer: suspend (S, A) -> S
) : BaseStateManager<S, A>(initialState) {
    
    private val pendingActions = MutableStateFlow<Set<String>>(emptySet())
    val isProcessing: StateFlow<Boolean> = pendingActions.map { it.isNotEmpty() }.stateIn(
        scope = kotlinx.coroutines.GlobalScope,
        started = SharingStarted.WhileSubscribed(),
        initialValue = false
    )
    
    override suspend fun dispatch(action: A) {
        val actionId = UUID.randomUUID().toString()
        addPendingAction(actionId)
        
        try {
            super.dispatch(action)
        } finally {
            removePendingAction(actionId)
        }
    }
    
    override suspend fun reduce(currentState: S, action: A): S {
        return asyncReducer(currentState, action)
    }
    
    private suspend fun addPendingAction(actionId: String) {
        pendingActions.value = pendingActions.value + actionId
    }
    
    private suspend fun removePendingAction(actionId: String) {
        pendingActions.value = pendingActions.value - actionId
    }
}

class LoggingMiddleware<S, A> : CompositeStateManager.StateMiddleware<S, A> {
    override suspend fun process(currentState: S, action: A, next: suspend (A) -> S): S {
        val startTime = System.currentTimeMillis()
        
        println("Action dispatched: ${action::class.simpleName}")
        
        val newState = next(action)
        val duration = System.currentTimeMillis() - startTime
        
        println("State transition completed in ${duration}ms")
        
        return newState
    }
}

class ValidationMiddleware<S, A>(
    private val validator: (S, A) -> Boolean
) : CompositeStateManager.StateMiddleware<S, A> {
    override suspend fun process(currentState: S, action: A, next: suspend (A) -> S): S {
        if (!validator(currentState, action)) {
            throw IllegalArgumentException("Action validation failed: ${action::class.simpleName}")
        }
        return next(action)
    }
}