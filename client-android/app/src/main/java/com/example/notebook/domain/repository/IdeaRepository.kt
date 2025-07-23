package com.example.notebook.domain.repository

import com.example.notebook.domain.model.*
import kotlinx.coroutines.flow.Flow

interface IdeaRepository {
    suspend fun createIdea(request: CreateIdeaRequest): Result<Idea>
    
    suspend fun getIdea(id: String): Result<Idea>
    
    suspend fun listIdeas(filters: IdeaFilters): Result<List<Idea>>
    
    suspend fun updateIdea(request: UpdateIdeaRequest): Result<Idea>
    
    suspend fun deleteIdea(id: String): Result<Unit>
    
    fun observeIdeas(filters: IdeaFilters): Flow<List<Idea>>
    
    suspend fun searchIdeas(query: String): Result<List<Idea>>
}

interface ReminderRepository {
    suspend fun createReminder(request: CreateReminderRequest): Result<Reminder>
    
    suspend fun getReminder(id: String): Result<Reminder>
    
    suspend fun listReminders(filters: ReminderFilters): Result<List<Reminder>>
    
    suspend fun updateReminder(request: UpdateReminderRequest): Result<Reminder>
    
    suspend fun deleteReminder(id: String): Result<Unit>
    
    fun observeReminders(filters: ReminderFilters): Flow<List<Reminder>>
    
    suspend fun getUpcomingReminders(): Result<List<Reminder>>
}

interface FileRepository {
    suspend fun uploadFile(
        filename: String,
        contentType: String,
        data: ByteArray,
        compress: Boolean = true,
        compressionType: String = "gzip"
    ): Result<FileInfo>
    
    suspend fun downloadFile(fileId: String): Result<Pair<FileInfo, ByteArray>>
    
    suspend fun deleteFile(fileId: String): Result<Unit>
    
    suspend fun listFiles(filters: FileFilters): Result<List<FileInfo>>
    
    fun observeFiles(): Flow<List<FileInfo>>
}

interface ProgressRepository {
    suspend fun updateProgress(request: UpdateProgressRequest): Result<Progress>
    
    suspend fun getProgress(id: String): Result<Progress>
    
    suspend fun listUserProgress(): Result<List<Progress>>
    
    fun observeProgress(): Flow<List<Progress>>
}

data class FileInfo(
    val id: String,
    val filename: String,
    val contentType: String,
    val size: Long,
    val checksum: String,
    val createdAt: kotlinx.datetime.Instant,
    val userId: String,
    val compressed: Boolean,
    val compressionType: String,
    val path: String
)

data class FileFilters(
    val contentTypeFilter: String? = null,
    val page: Int = 1,
    val pageSize: Int = 10,
    val sortBy: String = "created_at",
    val sortDesc: Boolean = true
)

data class Progress(
    val id: String,
    val userId: String,
    val projectName: String,
    val description: String,
    val completionPercentage: Float,
    val milestones: List<ProgressMilestone>,
    val createdAt: kotlinx.datetime.Instant,
    val updatedAt: kotlinx.datetime.Instant
)

data class ProgressMilestone(
    val id: String,
    val name: String,
    val description: String,
    val completed: Boolean,
    val dueDate: kotlinx.datetime.Instant,
    val completedAt: kotlinx.datetime.Instant?
)

data class UpdateProgressRequest(
    val id: String?,
    val projectName: String,
    val description: String,
    val completionPercentage: Float,
    val milestones: List<ProgressMilestone>
)