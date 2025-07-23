package com.example.notebook.presentation.ui.ideas

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.example.notebook.domain.model.*
import com.example.notebook.domain.repository.IdeaRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class IdeasViewModel @Inject constructor(
    private val ideaRepository: IdeaRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(IdeasUiState())
    val uiState: StateFlow<IdeasUiState> = _uiState.asStateFlow()

    private val _filters = MutableStateFlow(IdeaFilters())
    val filters: StateFlow<IdeaFilters> = _filters.asStateFlow()

    init {
        loadIdeas()
        observeIdeas()
    }

    fun loadIdeas() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            
            ideaRepository.listIdeas(_filters.value)
                .onSuccess { ideas ->
                    _uiState.update { 
                        it.copy(
                            ideas = ideas,
                            isLoading = false,
                            error = null
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update { 
                        it.copy(
                            isLoading = false,
                            error = error.message ?: "Error desconocido"
                        )
                    }
                }
        }
    }

    fun createIdea(request: CreateIdeaRequest) {
        viewModelScope.launch {
            _uiState.update { it.copy(isCreating = true) }
            
            ideaRepository.createIdea(request)
                .onSuccess { newIdea ->
                    _uiState.update { state ->
                        state.copy(
                            ideas = listOf(newIdea) + state.ideas,
                            isCreating = false,
                            showCreateDialog = false
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update { 
                        it.copy(
                            isCreating = false,
                            error = error.message ?: "Error al crear la idea"
                        )
                    }
                }
        }
    }

    fun updateIdea(request: UpdateIdeaRequest) {
        viewModelScope.launch {
            ideaRepository.updateIdea(request)
                .onSuccess { updatedIdea ->
                    _uiState.update { state ->
                        state.copy(
                            ideas = state.ideas.map { idea ->
                                if (idea.id == updatedIdea.id) updatedIdea else idea
                            }
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update { 
                        it.copy(error = error.message ?: "Error al actualizar la idea")
                    }
                }
        }
    }

    fun deleteIdea(ideaId: String) {
        viewModelScope.launch {
            ideaRepository.deleteIdea(ideaId)
                .onSuccess {
                    _uiState.update { state ->
                        state.copy(
                            ideas = state.ideas.filter { it.id != ideaId }
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update { 
                        it.copy(error = error.message ?: "Error al eliminar la idea")
                    }
                }
        }
    }

    fun searchIdeas(query: String) {
        if (query.isBlank()) {
            loadIdeas()
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            
            ideaRepository.searchIdeas(query)
                .onSuccess { ideas ->
                    _uiState.update { 
                        it.copy(
                            ideas = ideas,
                            isLoading = false,
                            searchQuery = query
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update { 
                        it.copy(
                            isLoading = false,
                            error = error.message ?: "Error en la búsqueda"
                        )
                    }
                }
        }
    }

    fun updateFilters(newFilters: IdeaFilters) {
        _filters.value = newFilters
        loadIdeas()
    }

    fun showCreateDialog() {
        _uiState.update { it.copy(showCreateDialog = true) }
    }

    fun hideCreateDialog() {
        _uiState.update { it.copy(showCreateDialog = false) }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }

    private fun observeIdeas() {
        viewModelScope.launch {
            _filters.flatMapLatest { filters ->
                ideaRepository.observeIdeas(filters)
            }.collect { ideas ->
                _uiState.update { it.copy(ideas = ideas) }
            }
        }
    }
}

data class IdeasUiState(
    val ideas: List<Idea> = emptyList(),
    val isLoading: Boolean = false,
    val isCreating: Boolean = false,
    val error: String? = null,
    val searchQuery: String = "",
    val showCreateDialog: Boolean = false,
    val selectedIdea: Idea? = null
)