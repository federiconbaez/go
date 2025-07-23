package com.example.notebook.presentation.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Error
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp

// Generic Loading Component
@Composable
fun <T> LoadingState(
    isLoading: Boolean,
    content: @Composable () -> Unit,
    loadingContent: @Composable () -> Unit = { DefaultLoadingContent() }
) {
    if (isLoading) {
        loadingContent()
    } else {
        content()
    }
}

@Composable
fun DefaultLoadingContent() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            CircularProgressIndicator()
            Spacer(modifier = Modifier.height(16.dp))
            Text(
                text = "Cargando...",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

// Generic Error Component
@Composable
fun <T> ErrorState(
    error: String?,
    onRetry: (() -> Unit)? = null,
    content: @Composable () -> Unit
) {
    if (error != null) {
        ErrorContent(
            message = error,
            onRetry = onRetry
        )
    } else {
        content()
    }
}

@Composable
fun ErrorContent(
    message: String,
    onRetry: (() -> Unit)? = null,
    icon: ImageVector = Icons.Default.Error
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.error,
            modifier = Modifier.size(48.dp)
        )
        
        Spacer(modifier = Modifier.height(16.dp))
        
        Text(
            text = message,
            style = MaterialTheme.typography.titleMedium,
            color = MaterialTheme.colorScheme.error,
            textAlign = TextAlign.Center
        )
        
        if (onRetry != null) {
            Spacer(modifier = Modifier.height(24.dp))
            
            Button(
                onClick = onRetry,
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.error
                )
            ) {
                Icon(
                    imageVector = Icons.Default.Refresh,
                    contentDescription = null
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Reintentar")
            }
        }
    }
}

// Generic Empty State Component
@Composable
fun <T> EmptyState(
    isEmpty: Boolean,
    emptyMessage: String,
    emptyAction: (@Composable () -> Unit)? = null,
    content: @Composable () -> Unit
) {
    if (isEmpty) {
        EmptyContent(
            message = emptyMessage,
            action = emptyAction
        )
    } else {
        content()
    }
}

@Composable
fun EmptyContent(
    message: String,
    action: (@Composable () -> Unit)? = null
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text(
            text = message,
            style = MaterialTheme.typography.titleMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center
        )
        
        if (action != null) {
            Spacer(modifier = Modifier.height(24.dp))
            action()
        }
    }
}

// Generic List Component with pagination support
@Composable
fun <T> GenericList(
    items: List<T>,
    listState: LazyListState = rememberLazyListState(),
    onLoadMore: (() -> Unit)? = null,
    loadMoreThreshold: Int = 5,
    itemContent: @Composable (T) -> Unit
) {
    LazyColumn(
        state = listState,
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        items(items) { item ->
            itemContent(item)
        }
    }
    
    // Handle infinite scroll
    if (onLoadMore != null) {
        val shouldLoadMore by remember {
            derivedStateOf {
                val layoutInfo = listState.layoutInfo
                val totalItemsNumber = layoutInfo.totalItemsCount
                val lastVisibleItemIndex = (layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: 0) + 1
                
                lastVisibleItemIndex > (totalItemsNumber - loadMoreThreshold)
            }
        }
        
        LaunchedEffect(shouldLoadMore) {
            if (shouldLoadMore) {
                onLoadMore()
            }
        }
    }
}

// Generic Card Component
@Composable
fun <T> GenericCard(
    item: T,
    onClick: ((T) -> Unit)? = null,
    modifier: Modifier = Modifier,
    content: @Composable (T) -> Unit
) {
    Card(
        onClick = { onClick?.invoke(item) },
        modifier = modifier.fillMaxWidth()
    ) {
        content(item)
    }
}

// Generic Dialog Component
@Composable
fun <T> GenericDialog(
    isVisible: Boolean,
    title: String,
    onDismiss: () -> Unit,
    onConfirm: (T) -> Unit,
    confirmButtonText: String = "Confirmar",
    dismissButtonText: String = "Cancelar",
    content: @Composable (onValueChange: (T) -> Unit) -> Unit
) {
    if (isVisible) {
        var currentValue by remember { mutableStateOf<T?>(null) }
        
        AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text(title) },
            text = {
                content { value ->
                    currentValue = value
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        currentValue?.let { onConfirm(it) }
                    }
                ) {
                    Text(confirmButtonText)
                }
            },
            dismissButton = {
                TextButton(onClick = onDismiss) {
                    Text(dismissButtonText)
                }
            }
        )
    }
}

// Generic Search Component
@Composable
fun <T> SearchableList(
    items: List<T>,
    searchQuery: String,
    onSearchQueryChange: (String) -> Unit,
    searchPlaceholder: String = "Buscar...",
    searchPredicate: (T, String) -> Boolean,
    listContent: @Composable (List<T>) -> Unit
) {
    Column {
        SearchBar(
            query = searchQuery,
            onQueryChange = onSearchQueryChange,
            placeholder = searchPlaceholder,
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        )
        
        val filteredItems = remember(items, searchQuery) {
            if (searchQuery.isBlank()) {
                items
            } else {
                items.filter { searchPredicate(it, searchQuery) }
            }
        }
        
        listContent(filteredItems)
    }
}

// Generic State Handler Component
@Composable
fun <T> StateHandler(
    uiState: UiState<T>,
    onRetry: (() -> Unit)? = null,
    loadingContent: @Composable () -> Unit = { DefaultLoadingContent() },
    errorContent: @Composable (String) -> Unit = { error ->
        ErrorContent(message = error, onRetry = onRetry)
    },
    emptyContent: @Composable () -> Unit = {
        EmptyContent(message = "No hay datos disponibles")
    },
    content: @Composable (T) -> Unit
) {
    when (uiState) {
        is UiState.Loading -> loadingContent()
        is UiState.Error -> errorContent(uiState.message)
        is UiState.Empty -> emptyContent()
        is UiState.Success -> content(uiState.data)
    }
}

// Generic UI State sealed class
sealed class UiState<out T> {
    object Loading : UiState<Nothing>()
    data class Error(val message: String) : UiState<Nothing>()
    object Empty : UiState<Nothing>()
    data class Success<T>(val data: T) : UiState<T>()
}

// Extension functions for UiState
fun <T> UiState<T>.isLoading(): Boolean = this is UiState.Loading
fun <T> UiState<T>.isError(): Boolean = this is UiState.Error
fun <T> UiState<T>.isSuccess(): Boolean = this is UiState.Success
fun <T> UiState<T>.isEmpty(): Boolean = this is UiState.Empty

fun <T> UiState<T>.getDataOrNull(): T? = when (this) {
    is UiState.Success -> data
    else -> null
}

fun <T> UiState<T>.getErrorOrNull(): String? = when (this) {
    is UiState.Error -> message
    else -> null
}

// Generic Repository Result Handler
@Composable
fun <T> HandleRepositoryResult(
    result: Result<T>?,
    onSuccess: @Composable (T) -> Unit,
    onError: @Composable (String) -> Unit = { error ->
        ErrorContent(message = error)
    },
    onLoading: @Composable () -> Unit = { DefaultLoadingContent() }
) {
    when {
        result == null -> onLoading()
        result.isSuccess -> onSuccess(result.getOrThrow())
        result.isFailure -> onError(
            result.exceptionOrNull()?.message ?: "Error desconocido"
        )
    }
}

// Generic Confirmation Dialog
@Composable
fun ConfirmationDialog(
    isVisible: Boolean,
    title: String,
    message: String,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
    confirmButtonText: String = "Confirmar",
    dismissButtonText: String = "Cancelar"
) {
    if (isVisible) {
        AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text(title) },
            text = { Text(message) },
            confirmButton = {
                TextButton(onClick = onConfirm) {
                    Text(confirmButtonText)
                }
            },
            dismissButton = {
                TextButton(onClick = onDismiss) {
                    Text(dismissButtonText)
                }
            }
        )
    }
}