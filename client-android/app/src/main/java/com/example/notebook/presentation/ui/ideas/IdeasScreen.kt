package com.example.notebook.presentation.ui.ideas

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.example.notebook.domain.model.Idea
import com.example.notebook.domain.model.IdeaCategory
import com.example.notebook.domain.model.IdeaStatus
import com.example.notebook.presentation.components.ErrorMessage
import com.example.notebook.presentation.components.LoadingIndicator
import com.example.notebook.presentation.components.SearchBar
import com.example.notebook.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun IdeasScreen(
    onNavigateToIdeaDetail: (String) -> Unit,
    modifier: Modifier = Modifier,
    viewModel: IdeasViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    var searchQuery by remember { mutableStateOf("") }

    Column(
        modifier = modifier.fillMaxSize()
    ) {
        // Search Bar
        SearchBar(
            query = searchQuery,
            onQueryChange = { searchQuery = it },
            onSearch = { viewModel.searchIdeas(it) },
            placeholder = "Buscar ideas...",
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        )

        when {
            uiState.isLoading -> {
                LoadingIndicator(
                    modifier = Modifier.fillMaxSize()
                )
            }
            
            uiState.error != null -> {
                ErrorMessage(
                    message = uiState.error!!,
                    onRetry = { viewModel.loadIdeas() },
                    modifier = Modifier.fillMaxSize()
                )
            }
            
            uiState.ideas.isEmpty() -> {
                EmptyIdeasState(
                    onCreateIdea = { viewModel.showCreateDialog() },
                    modifier = Modifier.fillMaxSize()
                )
            }
            
            else -> {
                LazyColumn(
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.weight(1f)
                ) {
                    items(uiState.ideas) { idea ->
                        IdeaCard(
                            idea = idea,
                            onClick = { onNavigateToIdeaDetail(idea.id) },
                            onDelete = { viewModel.deleteIdea(idea.id) }
                        )
                    }
                }
            }
        }

        // FAB para crear nueva idea
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            contentAlignment = Alignment.BottomEnd
        ) {
            FloatingActionButton(
                onClick = { viewModel.showCreateDialog() }
            ) {
                Icon(
                    imageVector = Icons.Default.Add,
                    contentDescription = "Crear idea"
                )
            }
        }
    }

    // Dialog para crear idea
    if (uiState.showCreateDialog) {
        CreateIdeaDialog(
            onDismiss = { viewModel.hideCreateDialog() },
            onConfirm = { request ->
                viewModel.createIdea(request)
            },
            isCreating = uiState.isCreating
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun IdeaCard(
    idea: Idea,
    onClick: () -> Unit,
    onDelete: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        onClick = onClick,
        modifier = modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier.padding(16.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.Top
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = idea.title,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis
                    )
                    
                    Spacer(modifier = Modifier.height(4.dp))
                    
                    Text(
                        text = idea.content,
                        style = MaterialTheme.typography.bodyMedium,
                        maxLines = 3,
                        overflow = TextOverflow.Ellipsis,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                
                Column(
                    horizontalAlignment = Alignment.End
                ) {
                    CategoryChip(category = idea.category)
                    Spacer(modifier = Modifier.height(4.dp))
                    StatusChip(status = idea.status)
                }
            }
            
            if (idea.tags.isNotEmpty()) {
                Spacer(modifier = Modifier.height(8.dp))
                LazyRow(
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    items(idea.tags) { tag ->
                        AssistChip(
                            onClick = { },
                            label = { Text(text = "#$tag") }
                        )
                    }
                }
            }
            
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Prioridad: ${idea.priority}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                
                TextButton(onClick = onDelete) {
                    Text(
                        text = "Eliminar",
                        color = MaterialTheme.colorScheme.error
                    )
                }
            }
        }
    }
}

@Composable
private fun CategoryChip(
    category: IdeaCategory,
    modifier: Modifier = Modifier
) {
    SuggestionChip(
        onClick = { },
        label = { Text(text = category.displayName) },
        modifier = modifier
    )
}

@Composable
private fun StatusChip(
    status: IdeaStatus,
    modifier: Modifier = Modifier
) {
    val colors = when (status) {
        IdeaStatus.DRAFT -> SuggestionChipDefaults.suggestionChipColors()
        IdeaStatus.ACTIVE -> SuggestionChipDefaults.suggestionChipColors()
        IdeaStatus.COMPLETED -> SuggestionChipDefaults.suggestionChipColors()
        IdeaStatus.ARCHIVED -> SuggestionChipDefaults.suggestionChipColors()
        else -> SuggestionChipDefaults.suggestionChipColors()
    }
    
    SuggestionChip(
        onClick = { },
        label = { Text(text = status.displayName) },
        colors = colors,
        modifier = modifier
    )
}

@Composable
private fun EmptyIdeasState(
    onCreateIdea: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier,
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text(
            text = "No tienes ideas guardadas",
            style = MaterialTheme.typography.headlineSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        
        Spacer(modifier = Modifier.height(8.dp))
        
        Text(
            text = "Crea tu primera idea para comenzar",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        
        Spacer(modifier = Modifier.height(24.dp))
        
        Button(onClick = onCreateIdea) {
            Icon(
                imageVector = Icons.Default.Add,
                contentDescription = null
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(text = "Crear idea")
        }
    }
}