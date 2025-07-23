package com.example.notebook.data.repository

import com.example.notebook.data.local.UserPreferencesDataStore
import com.example.notebook.data.mapper.IdeaMapper
import com.example.notebook.data.remote.GrpcClient
import com.example.notebook.domain.model.*
import com.example.notebook.domain.repository.IdeaRepository
import com.example.notebook.grpc.*
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.flow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class IdeaRepositoryImpl @Inject constructor(
    private val grpcClient: GrpcClient,
    private val userPrefs: UserPreferencesDataStore,
    private val mapper: IdeaMapper
) : IdeaRepository {

    override suspend fun createIdea(request: CreateIdeaRequest): Result<Idea> {
        return try {
            val userId = userPrefs.getUserId().first()
            val grpcRequest = createIdeaRequest {
                title = request.title
                content = request.content
                tags.addAll(request.tags)
                category = mapper.mapCategoryToProto(request.category)
                priority = request.priority
                this.userId = userId
            }
            
            val response = grpcClient.stub.createIdea(grpcRequest)
            
            if (response.success) {
                Result.success(mapper.mapIdeaFromProto(response.idea))
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getIdea(id: String): Result<Idea> {
        return try {
            val userId = userPrefs.getUserId().first()
            val grpcRequest = getIdeaRequest {
                this.id = id
                this.userId = userId
            }
            
            val response = grpcClient.stub.getIdea(grpcRequest)
            
            if (response.success) {
                Result.success(mapper.mapIdeaFromProto(response.idea))
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun listIdeas(filters: IdeaFilters): Result<List<Idea>> {
        return try {
            val userId = userPrefs.getUserId().first()
            val grpcRequest = listIdeasRequest {
                this.userId = userId
                filters.category?.let { category = mapper.mapCategoryToProto(it) }
                filters.status?.let { status = mapper.mapStatusToProto(it) }
                filters.tags?.let { tags.addAll(it) }
                page = filters.page
                pageSize = filters.pageSize
                sortBy = filters.sortBy
                sortDesc = filters.sortDesc
            }
            
            val response = grpcClient.stub.listIdeas(grpcRequest)
            
            if (response.success) {
                val ideas = response.ideasList.map { mapper.mapIdeaFromProto(it) }
                Result.success(ideas)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun updateIdea(request: UpdateIdeaRequest): Result<Idea> {
        return try {
            val userId = userPrefs.getUserId().first()
            val grpcRequest = updateIdeaRequest {
                id = request.id
                this.userId = userId
                request.title?.let { title = it }
                request.content?.let { content = it }
                request.tags?.let { tags.addAll(it) }
                request.category?.let { category = mapper.mapCategoryToProto(it) }
                request.status?.let { status = mapper.mapStatusToProto(it) }
                request.priority?.let { priority = it }
            }
            
            val response = grpcClient.stub.updateIdea(grpcRequest)
            
            if (response.success) {
                Result.success(mapper.mapIdeaFromProto(response.idea))
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun deleteIdea(id: String): Result<Unit> {
        return try {
            val userId = userPrefs.getUserId().first()
            val grpcRequest = deleteIdeaRequest {
                this.id = id
                this.userId = userId
            }
            
            val response = grpcClient.stub.deleteIdea(grpcRequest)
            
            if (response.success) {
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override fun observeIdeas(filters: IdeaFilters): Flow<List<Idea>> = flow {
        // Para observar ideas en tiempo real, podrías implementar una cache local
        // y actualizarla periódicamente o usar WebSockets/Server-Sent Events
        while (true) {
            listIdeas(filters).getOrNull()?.let { emit(it) }
            kotlinx.coroutines.delay(30000) // Actualizar cada 30 segundos
        }
    }

    override suspend fun searchIdeas(query: String): Result<List<Idea>> {
        // Implementar búsqueda usando filtros de texto
        val filters = IdeaFilters()
        return listIdeas(filters).map { ideas ->
            ideas.filter { idea ->
                idea.title.contains(query, ignoreCase = true) ||
                idea.content.contains(query, ignoreCase = true) ||
                idea.tags.any { it.contains(query, ignoreCase = true) }
            }
        }
    }
}