package com.example.notebook.data.mapper

import com.example.notebook.domain.model.*
import com.example.notebook.grpc.*
import kotlinx.datetime.Instant
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class IdeaMapper @Inject constructor() {

    // Domain to Proto mappings
    fun mapCategoryToProto(category: IdeaCategory): com.example.notebook.grpc.IdeaCategory {
        return when (category) {
            IdeaCategory.UNSPECIFIED -> com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_UNSPECIFIED
            IdeaCategory.BUSINESS -> com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_BUSINESS
            IdeaCategory.PERSONAL -> com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_PERSONAL
            IdeaCategory.TECHNICAL -> com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_TECHNICAL
            IdeaCategory.CREATIVE -> com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_CREATIVE
            IdeaCategory.RESEARCH -> com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_RESEARCH
        }
    }

    fun mapStatusToProto(status: IdeaStatus): com.example.notebook.grpc.IdeaStatus {
        return when (status) {
            IdeaStatus.UNSPECIFIED -> com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_UNSPECIFIED
            IdeaStatus.DRAFT -> com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_DRAFT
            IdeaStatus.ACTIVE -> com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_ACTIVE
            IdeaStatus.ON_HOLD -> com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_ON_HOLD
            IdeaStatus.COMPLETED -> com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_COMPLETED
            IdeaStatus.ARCHIVED -> com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_ARCHIVED
        }
    }

    // Proto to Domain mappings
    fun mapCategoryFromProto(category: com.example.notebook.grpc.IdeaCategory): IdeaCategory {
        return when (category) {
            com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_UNSPECIFIED -> IdeaCategory.UNSPECIFIED
            com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_BUSINESS -> IdeaCategory.BUSINESS
            com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_PERSONAL -> IdeaCategory.PERSONAL
            com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_TECHNICAL -> IdeaCategory.TECHNICAL
            com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_CREATIVE -> IdeaCategory.CREATIVE
            com.example.notebook.grpc.IdeaCategory.IDEA_CATEGORY_RESEARCH -> IdeaCategory.RESEARCH
            else -> IdeaCategory.UNSPECIFIED
        }
    }

    fun mapStatusFromProto(status: com.example.notebook.grpc.IdeaStatus): IdeaStatus {
        return when (status) {
            com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_UNSPECIFIED -> IdeaStatus.UNSPECIFIED
            com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_DRAFT -> IdeaStatus.DRAFT
            com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_ACTIVE -> IdeaStatus.ACTIVE
            com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_ON_HOLD -> IdeaStatus.ON_HOLD
            com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_COMPLETED -> IdeaStatus.COMPLETED
            com.example.notebook.grpc.IdeaStatus.IDEA_STATUS_ARCHIVED -> IdeaStatus.ARCHIVED
            else -> IdeaStatus.UNSPECIFIED
        }
    }

    // Complete Idea mapping from Proto
    fun mapIdeaFromProto(protoIdea: com.example.notebook.grpc.Idea): Idea {
        return Idea(
            id = protoIdea.id,
            title = protoIdea.title,
            content = protoIdea.content,
            tags = protoIdea.tagsList.toList(),
            category = mapCategoryFromProto(protoIdea.category),
            status = mapStatusFromProto(protoIdea.status),
            createdAt = Instant.fromEpochSeconds(
                protoIdea.createdAt.seconds,
                protoIdea.createdAt.nanos
            ),
            updatedAt = Instant.fromEpochSeconds(
                protoIdea.updatedAt.seconds,
                protoIdea.updatedAt.nanos
            ),
            userId = protoIdea.userId,
            relatedIdeas = protoIdea.relatedIdeasList.toList(),
            priority = protoIdea.priority
        )
    }

    // Complete Idea mapping to Proto
    fun mapIdeaToProto(idea: Idea): com.example.notebook.grpc.Idea {
        return com.example.notebook.grpc.idea {
            id = idea.id
            title = idea.title
            content = idea.content
            tags.addAll(idea.tags)
            category = mapCategoryToProto(idea.category)
            status = mapStatusToProto(idea.status)
            createdAt = com.google.protobuf.timestamp {
                seconds = idea.createdAt.epochSeconds
                nanos = idea.createdAt.nanosecondsOfSecond
            }
            updatedAt = com.google.protobuf.timestamp {
                seconds = idea.updatedAt.epochSeconds
                nanos = idea.updatedAt.nanosecondsOfSecond
            }
            userId = idea.userId
            relatedIdeas.addAll(idea.relatedIdeas)
            priority = idea.priority
        }
    }

    // Create request mapping
    fun mapCreateRequestToProto(
        request: CreateIdeaRequest,
        userId: String
    ): com.example.notebook.grpc.CreateIdeaRequest {
        return com.example.notebook.grpc.createIdeaRequest {
            title = request.title
            content = request.content
            tags.addAll(request.tags)
            category = mapCategoryToProto(request.category)
            priority = request.priority
            this.userId = userId
        }
    }

    // Update request mapping
    fun mapUpdateRequestToProto(
        request: UpdateIdeaRequest,
        userId: String
    ): com.example.notebook.grpc.UpdateIdeaRequest {
        return com.example.notebook.grpc.updateIdeaRequest {
            id = request.id
            this.userId = userId
            request.title?.let { title = it }
            request.content?.let { content = it }
            request.tags?.let { tags.addAll(it) }
            request.category?.let { category = mapCategoryToProto(it) }
            request.status?.let { status = mapStatusToProto(it) }
            request.priority?.let { priority = it }
        }
    }

    // List filters mapping
    fun mapFiltersToProto(
        filters: IdeaFilters,
        userId: String
    ): com.example.notebook.grpc.ListIdeasRequest {
        return com.example.notebook.grpc.listIdeasRequest {
            this.userId = userId
            filters.category?.let { category = mapCategoryToProto(it) }
            filters.status?.let { status = mapStatusToProto(it) }
            filters.tags?.let { tags.addAll(it) }
            page = filters.page
            pageSize = filters.pageSize
            sortBy = filters.sortBy
            sortDesc = filters.sortDesc
        }
    }
}

// Generic mapper interface for type safety
interface Mapper<Domain, Proto> {
    fun toDomain(proto: Proto): Domain
    fun toProto(domain: Domain): Proto
}

// Generic bidirectional mapper
abstract class BidirectionalMapper<Domain, Proto> : Mapper<Domain, Proto> {
    abstract override fun toDomain(proto: Proto): Domain
    abstract override fun toProto(domain: Domain): Proto
    
    fun toDomainList(protoList: List<Proto>): List<Domain> {
        return protoList.map { toDomain(it) }
    }
    
    fun toProtoList(domainList: List<Domain>): List<Proto> {
        return domainList.map { toProto(it) }
    }
}

// Extension functions for easier conversion
inline fun <Domain, Proto> Mapper<Domain, Proto>.mapSafely(
    proto: Proto?,
    default: Domain
): Domain {
    return proto?.let { toDomain(it) } ?: default
}

inline fun <Domain, Proto> List<Proto>.mapToDomain(
    mapper: Mapper<Domain, Proto>
): List<Domain> {
    return this.map { mapper.toDomain(it) }
}

inline fun <Domain, Proto> List<Domain>.mapToProto(
    mapper: Mapper<Domain, Proto>
): List<Proto> {
    return this.map { mapper.toProto(it) }
}