package com.example.notebook.presentation.navigation

import kotlinx.coroutines.flow.*
import java.util.*

sealed class NavigationAction {
    data class Navigate(val destination: Destination, val options: NavigationOptions = NavigationOptions()) : NavigationAction()
    object NavigateBack : NavigationAction()
    data class NavigateBackTo(val destination: Destination) : NavigationAction()
    data class Replace(val destination: Destination, val options: NavigationOptions = NavigationOptions()) : NavigationAction()
    object ClearBackStack : NavigationAction()
    data class NavigateAndClearBackStack(val destination: Destination) : NavigationAction()
}

data class Destination(
    val route: String,
    val arguments: Map<String, Any> = emptyMap(),
    val id: String = UUID.randomUUID().toString()
) {
    companion object {
        fun create(route: String, vararg args: Pair<String, Any>): Destination {
            return Destination(route, args.toMap())
        }
    }
}

data class NavigationOptions(
    val clearBackStack: Boolean = false,
    val singleTop: Boolean = false,
    val animated: Boolean = true,
    val transitionType: TransitionType = TransitionType.DEFAULT
) {
    enum class TransitionType {
        DEFAULT, SLIDE_LEFT, SLIDE_RIGHT, FADE, SLIDE_UP, SLIDE_DOWN
    }
}

data class NavigationState(
    val currentDestination: Destination?,
    val backStack: List<Destination> = emptyList(),
    val isNavigating: Boolean = false,
    val canNavigateBack: Boolean = backStack.isNotEmpty()
) {
    companion object {
        val Initial = NavigationState(currentDestination = null)
    }
}

sealed class NavigationEvent {
    data class NavigationStarted(val from: Destination?, val to: Destination) : NavigationEvent()
    data class NavigationCompleted(val destination: Destination) : NavigationEvent()
    data class NavigationFailed(val destination: Destination, val error: Throwable) : NavigationEvent()
    data class BackNavigation(val from: Destination, val to: Destination?) : NavigationEvent()
}

interface NavigationInterceptor {
    suspend fun intercept(
        action: NavigationAction,
        currentState: NavigationState
    ): NavigationAction?
}

class AuthenticationInterceptor(
    private val isUserAuthenticated: () -> Boolean,
    private val loginDestination: Destination
) : NavigationInterceptor {
    
    private val protectedRoutes = setOf("profile", "settings", "dashboard")
    
    override suspend fun intercept(
        action: NavigationAction,
        currentState: NavigationState
    ): NavigationAction? {
        return when (action) {
            is NavigationAction.Navigate -> {
                if (protectedRoutes.contains(action.destination.route) && !isUserAuthenticated()) {
                    NavigationAction.Navigate(loginDestination)
                } else {
                    action
                }
            }
            else -> action
        }
    }
}

class DeepLinkInterceptor : NavigationInterceptor {
    override suspend fun intercept(
        action: NavigationAction,
        currentState: NavigationState
    ): NavigationAction? {
        return when (action) {
            is NavigationAction.Navigate -> {
                if (action.destination.route.startsWith("deeplink://")) {
                    val parsedDestination = parseDeepLink(action.destination.route)
                    NavigationAction.Navigate(parsedDestination, action.options)
                } else {
                    action
                }
            }
            else -> action
        }
    }
    
    private fun parseDeepLink(deepLink: String): Destination {
        val uri = android.net.Uri.parse(deepLink)
        val route = uri.path?.removePrefix("/") ?: "home"
        val arguments = mutableMapOf<String, Any>()
        
        uri.queryParameterNames.forEach { paramName ->
            uri.getQueryParameter(paramName)?.let { paramValue ->
                arguments[paramName] = paramValue
            }
        }
        
        return Destination(route, arguments)
    }
}

class NavigationController(
    private val interceptors: List<NavigationInterceptor> = emptyList()
) {
    
    private val _state = MutableStateFlow(NavigationState.Initial)
    val state: StateFlow<NavigationState> = _state.asStateFlow()
    
    private val _events = MutableSharedFlow<NavigationEvent>()
    val events: SharedFlow<NavigationEvent> = _events.asSharedFlow()
    
    private val navigationHistory = mutableListOf<Pair<Long, NavigationAction>>()
    private val maxHistorySize = 100
    
    suspend fun dispatch(action: NavigationAction) {
        recordAction(action)
        
        val processedAction = processWithInterceptors(action, _state.value)
        if (processedAction != null) {
            executeNavigation(processedAction)
        }
    }
    
    private suspend fun processWithInterceptors(
        action: NavigationAction,
        currentState: NavigationState
    ): NavigationAction? {
        var processedAction: NavigationAction? = action
        
        for (interceptor in interceptors) {
            processedAction = processedAction?.let { 
                interceptor.intercept(it, currentState) 
            }
            if (processedAction == null) break
        }
        
        return processedAction
    }
    
    private suspend fun executeNavigation(action: NavigationAction) {
        val currentState = _state.value
        
        try {
            _state.value = currentState.copy(isNavigating = true)
            
            val newState = when (action) {
                is NavigationAction.Navigate -> handleNavigate(action, currentState)
                is NavigationAction.NavigateBack -> handleNavigateBack(currentState)
                is NavigationAction.NavigateBackTo -> handleNavigateBackTo(action, currentState)
                is NavigationAction.Replace -> handleReplace(action, currentState)
                is NavigationAction.ClearBackStack -> handleClearBackStack(currentState)
                is NavigationAction.NavigateAndClearBackStack -> handleNavigateAndClearBackStack(action, currentState)
            }
            
            _state.value = newState.copy(isNavigating = false)
            
            newState.currentDestination?.let { destination ->
                _events.emit(NavigationEvent.NavigationCompleted(destination))
            }
            
        } catch (e: Exception) {
            _state.value = currentState.copy(isNavigating = false)
            currentState.currentDestination?.let { destination ->
                _events.emit(NavigationEvent.NavigationFailed(destination, e))
            }
        }
    }
    
    private suspend fun handleNavigate(
        action: NavigationAction.Navigate,
        currentState: NavigationState
    ): NavigationState {
        _events.emit(NavigationEvent.NavigationStarted(currentState.currentDestination, action.destination))
        
        val newBackStack = if (action.options.clearBackStack) {
            emptyList()
        } else {
            val updatedBackStack = currentState.currentDestination?.let { current ->
                if (action.options.singleTop && currentState.backStack.any { it.route == action.destination.route }) {
                    currentState.backStack.filter { it.route != action.destination.route }
                } else {
                    currentState.backStack + current
                }
            } ?: currentState.backStack
            
            updatedBackStack.takeLast(20)
        }
        
        return currentState.copy(
            currentDestination = action.destination,
            backStack = newBackStack,
            canNavigateBack = newBackStack.isNotEmpty()
        )
    }
    
    private suspend fun handleNavigateBack(currentState: NavigationState): NavigationState {
        if (currentState.backStack.isEmpty()) {
            return currentState
        }
        
        val previousDestination = currentState.backStack.last()
        val newBackStack = currentState.backStack.dropLast(1)
        
        _events.emit(NavigationEvent.BackNavigation(
            currentState.currentDestination ?: return currentState,
            previousDestination
        ))
        
        return currentState.copy(
            currentDestination = previousDestination,
            backStack = newBackStack,
            canNavigateBack = newBackStack.isNotEmpty()
        )
    }
    
    private suspend fun handleNavigateBackTo(
        action: NavigationAction.NavigateBackTo,
        currentState: NavigationState
    ): NavigationState {
        val targetIndex = currentState.backStack.indexOfLast { it.route == action.destination.route }
        
        if (targetIndex == -1) {
            return handleNavigate(
                NavigationAction.Navigate(action.destination),
                currentState
            )
        }
        
        val targetDestination = currentState.backStack[targetIndex]
        val newBackStack = currentState.backStack.take(targetIndex)
        
        return currentState.copy(
            currentDestination = targetDestination,
            backStack = newBackStack,
            canNavigateBack = newBackStack.isNotEmpty()
        )
    }
    
    private suspend fun handleReplace(
        action: NavigationAction.Replace,
        currentState: NavigationState
    ): NavigationState {
        return currentState.copy(
            currentDestination = action.destination
        )
    }
    
    private suspend fun handleClearBackStack(currentState: NavigationState): NavigationState {
        return currentState.copy(
            backStack = emptyList(),
            canNavigateBack = false
        )
    }
    
    private suspend fun handleNavigateAndClearBackStack(
        action: NavigationAction.NavigateAndClearBackStack,
        currentState: NavigationState
    ): NavigationState {
        return handleNavigate(
            NavigationAction.Navigate(action.destination, NavigationOptions(clearBackStack = true)),
            currentState
        )
    }
    
    private fun recordAction(action: NavigationAction) {
        navigationHistory.add(System.currentTimeMillis() to action)
        if (navigationHistory.size > maxHistorySize) {
            navigationHistory.removeAt(0)
        }
    }
    
    fun getCurrentDestination(): Destination? = _state.value.currentDestination
    
    fun canNavigateBack(): Boolean = _state.value.canNavigateBack
    
    fun getNavigationHistory(): List<Pair<Long, NavigationAction>> = navigationHistory.toList()
    
    fun getBackStackSize(): Int = _state.value.backStack.size
}