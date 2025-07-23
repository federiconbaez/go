package com.example.notebook.domain.model

import kotlinx.datetime.Instant
import kotlinx.parcelize.Parcelize
import android.os.Parcelable

@Parcelize
data class Idea(
    val id: String,
    val title: String,
    val content: String,
    val tags: List<String>,
    val category: IdeaCategory,
    val status: IdeaStatus,
    val createdAt: Instant,
    val updatedAt: Instant,
    val userId: String,
    val relatedIdeas: List<String>,
    val priority: Int
) : Parcelable

enum class IdeaCategory(val displayName: String) {
    UNSPECIFIED("Sin especificar"),
    BUSINESS("Negocio"),
    PERSONAL("Personal"),
    TECHNICAL("Técnico"),
    CREATIVE("Creativo"),
    RESEARCH("Investigación")
}

enum class IdeaStatus(val displayName: String) {
    UNSPECIFIED("Sin especificar"),
    DRAFT("Borrador"),
    ACTIVE("Activo"),
    ON_HOLD("En pausa"),
    COMPLETED("Completado"),
    ARCHIVED("Archivado")
}

data class CreateIdeaRequest(
    val title: String,
    val content: String,
    val tags: List<String>,
    val category: IdeaCategory,
    val priority: Int
)

data class UpdateIdeaRequest(
    val id: String,
    val title: String?,
    val content: String?,
    val tags: List<String>?,
    val category: IdeaCategory?,
    val status: IdeaStatus?,
    val priority: Int?
)

data class IdeaFilters(
    val category: IdeaCategory? = null,
    val status: IdeaStatus? = null,
    val tags: List<String>? = null,
    val page: Int = 1,
    val pageSize: Int = 10,
    val sortBy: String = "created_at",
    val sortDesc: Boolean = true
)