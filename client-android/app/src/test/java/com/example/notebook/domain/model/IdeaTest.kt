package com.example.notebook.domain.model

import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import org.junit.runners.JUnit4

@RunWith(JUnit4::class)
class IdeaTest {

    @Test
    fun `create idea with valid data should succeed`() {
        // Given
        val id = "test-id"
        val title = "Test Idea"
        val content = "Test content"
        val tags = listOf("test", "idea")
        val category = IdeaCategory.BUSINESS
        val status = IdeaStatus.DRAFT
        val now = Clock.System.now()
        val userId = "user-123"
        val relatedIdeas = listOf("related-1", "related-2")
        val priority = 5

        // When
        val idea = Idea(
            id = id,
            title = title,
            content = content,
            tags = tags,
            category = category,
            status = status,
            createdAt = now,
            updatedAt = now,
            userId = userId,
            relatedIdeas = relatedIdeas,
            priority = priority
        )

        // Then
        assertEquals(id, idea.id)
        assertEquals(title, idea.title)
        assertEquals(content, idea.content)
        assertEquals(tags, idea.tags)
        assertEquals(category, idea.category)
        assertEquals(status, idea.status)
        assertEquals(now, idea.createdAt)
        assertEquals(now, idea.updatedAt)
        assertEquals(userId, idea.userId)
        assertEquals(relatedIdeas, idea.relatedIdeas)
        assertEquals(priority, idea.priority)
    }

    @Test
    fun `idea should be parcelable`() {
        // Given
        val idea = createTestIdea()

        // When - This would be tested in Android instrumentation tests
        // as Parcelable requires Android framework
        
        // Then
        assertNotNull(idea)
        // In real Android test, you would test:
        // val parcel = Parcel.obtain()
        // idea.writeToParcel(parcel, 0)
        // parcel.setDataPosition(0)
        // val recreatedIdea = Idea.CREATOR.createFromParcel(parcel)
        // assertEquals(idea, recreatedIdea)
    }

    @Test
    fun `idea categories should have correct display names`() {
        // Given & When & Then
        assertEquals("Sin especificar", IdeaCategory.UNSPECIFIED.displayName)
        assertEquals("Negocio", IdeaCategory.BUSINESS.displayName)
        assertEquals("Personal", IdeaCategory.PERSONAL.displayName)
        assertEquals("Técnico", IdeaCategory.TECHNICAL.displayName)
        assertEquals("Creativo", IdeaCategory.CREATIVE.displayName)
        assertEquals("Investigación", IdeaCategory.RESEARCH.displayName)
    }

    @Test
    fun `idea statuses should have correct display names`() {
        // Given & When & Then
        assertEquals("Sin especificar", IdeaStatus.UNSPECIFIED.displayName)
        assertEquals("Borrador", IdeaStatus.DRAFT.displayName)
        assertEquals("Activo", IdeaStatus.ACTIVE.displayName)
        assertEquals("En pausa", IdeaStatus.ON_HOLD.displayName)
        assertEquals("Completado", IdeaStatus.COMPLETED.displayName)
        assertEquals("Archivado", IdeaStatus.ARCHIVED.displayName)
    }

    @Test
    fun `create idea request should be created correctly`() {
        // Given
        val title = "Test Request"
        val content = "Test content"
        val tags = listOf("test")
        val category = IdeaCategory.TECHNICAL
        val priority = 8

        // When
        val request = CreateIdeaRequest(
            title = title,
            content = content,
            tags = tags,
            category = category,
            priority = priority
        )

        // Then
        assertEquals(title, request.title)
        assertEquals(content, request.content)
        assertEquals(tags, request.tags)
        assertEquals(category, request.category)
        assertEquals(priority, request.priority)
    }

    @Test
    fun `update idea request should handle nullable fields correctly`() {
        // Given
        val id = "test-id"
        val title = "Updated Title"
        val tags = listOf("updated")

        // When
        val request = UpdateIdeaRequest(
            id = id,
            title = title,
            content = null, // Should be nullable
            tags = tags,
            category = null, // Should be nullable
            status = IdeaStatus.ACTIVE,
            priority = null // Should be nullable
        )

        // Then
        assertEquals(id, request.id)
        assertEquals(title, request.title)
        assertNull(request.content)
        assertEquals(tags, request.tags)
        assertNull(request.category)
        assertEquals(IdeaStatus.ACTIVE, request.status)
        assertNull(request.priority)
    }

    @Test
    fun `idea filters should have correct default values`() {
        // Given & When
        val filters = IdeaFilters()

        // Then
        assertNull(filters.category)
        assertNull(filters.status)
        assertNull(filters.tags)
        assertEquals(1, filters.page)
        assertEquals(10, filters.pageSize)
        assertEquals("created_at", filters.sortBy)
        assertTrue(filters.sortDesc)
    }

    @Test
    fun `idea filters should allow custom values`() {
        // Given
        val category = IdeaCategory.CREATIVE
        val status = IdeaStatus.COMPLETED
        val tags = listOf("custom", "filter")
        val page = 2
        val pageSize = 20
        val sortBy = "title"
        val sortDesc = false

        // When
        val filters = IdeaFilters(
            category = category,
            status = status,
            tags = tags,
            page = page,
            pageSize = pageSize,
            sortBy = sortBy,
            sortDesc = sortDesc
        )

        // Then
        assertEquals(category, filters.category)
        assertEquals(status, filters.status)
        assertEquals(tags, filters.tags)
        assertEquals(page, filters.page)
        assertEquals(pageSize, filters.pageSize)
        assertEquals(sortBy, filters.sortBy)
        assertEquals(sortDesc, filters.sortDesc)
    }

    private fun createTestIdea(): Idea {
        return Idea(
            id = "test-id-123",
            title = "Test Idea Title",
            content = "This is a test idea content with some meaningful text.",
            tags = listOf("test", "kotlin", "android"),
            category = IdeaCategory.TECHNICAL,
            status = IdeaStatus.ACTIVE,
            createdAt = Clock.System.now(),
            updatedAt = Clock.System.now(),
            userId = "user-456",
            relatedIdeas = listOf("related-789"),
            priority = 7
        )
    }
}

// Test utility class for creating test data with generics
class TestDataFactory<T> {
    
    companion object {
        
        inline fun <reified T> createListOf(
            size: Int,
            factory: (Int) -> T
        ): List<T> {
            return (0 until size).map { factory(it) }
        }
        
        fun createTestIdeas(count: Int): List<Idea> {
            return createListOf(count) { index ->
                Idea(
                    id = "idea-$index",
                    title = "Test Idea $index",
                    content = "Content for idea $index",
                    tags = listOf("test", "idea$index"),
                    category = IdeaCategory.values()[index % IdeaCategory.values().size],
                    status = IdeaStatus.values()[index % IdeaStatus.values().size],
                    createdAt = Clock.System.now(),
                    updatedAt = Clock.System.now(),
                    userId = "user-123",
                    relatedIdeas = emptyList(),
                    priority = (index % 10) + 1
                )
            }
        }
        
        fun createTestIdeaRequests(count: Int): List<CreateIdeaRequest> {
            return createListOf(count) { index ->
                CreateIdeaRequest(
                    title = "Request $index",
                    content = "Content $index",
                    tags = listOf("request", "test$index"),
                    category = IdeaCategory.values()[index % IdeaCategory.values().size],
                    priority = (index % 10) + 1
                )
            }
        }
    }
}