package com.example.notebook.data.repository

import com.example.notebook.data.local.UserPreferencesDataStore
import com.example.notebook.data.mapper.IdeaMapper
import com.example.notebook.data.remote.GrpcClient
import com.example.notebook.domain.model.*
import com.example.notebook.grpc.*
import io.mockk.*
import io.mockk.impl.annotations.MockK
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.junit.runners.JUnit4
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

@RunWith(JUnit4::class)
class IdeaRepositoryImplTest {

    @MockK
    private lateinit var grpcClient: GrpcClient

    @MockK
    private lateinit var userPrefs: UserPreferencesDataStore

    @MockK
    private lateinit var mapper: IdeaMapper

    @MockK
    private lateinit var stub: NotebookServiceGrpcKt.NotebookServiceCoroutineStub

    private lateinit var repository: IdeaRepositoryImpl

    @Before
    fun setup() {
        MockKAnnotations.init(this)
        
        every { grpcClient.stub } returns stub
        
        repository = IdeaRepositoryImpl(grpcClient, userPrefs, mapper)
    }

    @After
    fun tearDown() {
        clearAllMocks()
    }

    @Test
    fun `createIdea should return success when grpc call succeeds`() = runTest {
        // Given
        val userId = "user-123"
        val request = CreateIdeaRequest(
            title = "Test Idea",
            content = "Test content",
            tags = listOf("test"),
            category = IdeaCategory.BUSINESS,
            priority = 5
        )

        val expectedIdea = createTestIdea()
        val grpcRequest = mockk<CreateIdeaRequest>()
        val grpcResponse = mockk<CreateIdeaResponse> {
            every { success } returns true
            every { idea } returns mockk()
        }

        // Mock dependencies
        every { userPrefs.getUserId() } returns flowOf(userId)
        every { mapper.mapCategoryToProto(any()) } returns IdeaCategory.IDEA_CATEGORY_BUSINESS
        every { mapper.mapIdeaFromProto(any()) } returns expectedIdea
        
        // Create a slot to capture the gRPC request
        val grpcRequestSlot = slot<com.example.notebook.grpc.CreateIdeaRequest>()
        coEvery { stub.createIdea(capture(grpcRequestSlot)) } returns grpcResponse

        // When
        val result = repository.createIdea(request)

        // Then
        assertTrue(result.isSuccess)
        assertEquals(expectedIdea, result.getOrNull())
        
        // Verify gRPC request was built correctly
        coVerify { stub.createIdea(any()) }
        verify { mapper.mapCategoryToProto(IdeaCategory.BUSINESS) }
        verify { mapper.mapIdeaFromProto(any()) }
    }

    @Test
    fun `createIdea should return failure when grpc call fails`() = runTest {
        // Given
        val userId = "user-123"
        val request = CreateIdeaRequest(
            title = "Test Idea",
            content = "Test content",
            tags = listOf("test"),
            category = IdeaCategory.BUSINESS,
            priority = 5
        )

        val errorMessage = "Network error"
        val grpcResponse = mockk<CreateIdeaResponse> {
            every { success } returns false
            every { message } returns errorMessage
        }

        every { userPrefs.getUserId() } returns flowOf(userId)
        every { mapper.mapCategoryToProto(any()) } returns IdeaCategory.IDEA_CATEGORY_BUSINESS
        coEvery { stub.createIdea(any()) } returns grpcResponse

        // When
        val result = repository.createIdea(request)

        // Then
        assertTrue(result.isFailure)
        assertEquals(errorMessage, result.exceptionOrNull()?.message)
    }

    @Test
    fun `createIdea should handle exceptions`() = runTest {
        // Given
        val userId = "user-123"
        val request = CreateIdeaRequest(
            title = "Test Idea",
            content = "Test content",
            tags = listOf("test"),
            category = IdeaCategory.BUSINESS,
            priority = 5
        )

        val exception = RuntimeException("Connection failed")

        every { userPrefs.getUserId() } returns flowOf(userId)
        every { mapper.mapCategoryToProto(any()) } returns IdeaCategory.IDEA_CATEGORY_BUSINESS
        coEvery { stub.createIdea(any()) } throws exception

        // When
        val result = repository.createIdea(request)

        // Then
        assertTrue(result.isFailure)
        assertEquals(exception, result.exceptionOrNull())
    }

    @Test
    fun `getIdea should return success when idea exists and user is authorized`() = runTest {
        // Given
        val ideaId = "idea-123"
        val userId = "user-123"
        val expectedIdea = createTestIdea()
        
        val grpcResponse = mockk<GetIdeaResponse> {
            every { success } returns true
            every { idea } returns mockk()
        }

        every { userPrefs.getUserId() } returns flowOf(userId)
        every { mapper.mapIdeaFromProto(any()) } returns expectedIdea
        coEvery { stub.getIdea(any()) } returns grpcResponse

        // When
        val result = repository.getIdea(ideaId)

        // Then
        assertTrue(result.isSuccess)
        assertEquals(expectedIdea, result.getOrNull())
        
        coVerify { stub.getIdea(any()) }
        verify { mapper.mapIdeaFromProto(any()) }
    }

    @Test
    fun `listIdeas should return paginated list with correct mapping`() = runTest {
        // Given
        val userId = "user-123"
        val filters = IdeaFilters(
            category = IdeaCategory.BUSINESS,
            status = IdeaStatus.ACTIVE,
            page = 1,
            pageSize = 10
        )

        val expectedIdeas = listOf(createTestIdea(), createTestIdea())
        val grpcIdeas = listOf(mockk<com.example.notebook.grpc.Idea>(), mockk())
        val grpcResponse = mockk<ListIdeasResponse> {
            every { success } returns true
            every { ideasList } returns grpcIdeas
        }

        every { userPrefs.getUserId() } returns flowOf(userId)
        every { mapper.mapCategoryToProto(any()) } returns IdeaCategory.IDEA_CATEGORY_BUSINESS
        every { mapper.mapStatusToProto(any()) } returns IdeaStatus.IDEA_STATUS_ACTIVE
        every { mapper.mapIdeaFromProto(any()) } returnsMany expectedIdeas
        coEvery { stub.listIdeas(any()) } returns grpcResponse

        // When
        val result = repository.listIdeas(filters)

        // Then
        assertTrue(result.isSuccess)
        val ideas = result.getOrNull()!!
        assertEquals(expectedIdeas.size, ideas.size)
        
        coVerify { stub.listIdeas(any()) }
        verify { mapper.mapCategoryToProto(IdeaCategory.BUSINESS) }
        verify { mapper.mapStatusToProto(IdeaStatus.ACTIVE) }
        verify(exactly = expectedIdeas.size) { mapper.mapIdeaFromProto(any()) }
    }

    @Test
    fun `updateIdea should update successfully when authorized`() = runTest {
        // Given
        val updateRequest = UpdateIdeaRequest(
            id = "idea-123",
            title = "Updated Title",
            content = null,
            tags = listOf("updated"),
            category = IdeaCategory.TECHNICAL,
            status = IdeaStatus.ACTIVE,
            priority = 8
        )

        val expectedIdea = createTestIdea()
        val grpcResponse = mockk<UpdateIdeaResponse> {
            every { success } returns true
            every { idea } returns mockk()
        }

        every { userPrefs.getUserId() } returns flowOf("user-123")
        every { mapper.mapCategoryToProto(any()) } returns IdeaCategory.IDEA_CATEGORY_TECHNICAL
        every { mapper.mapStatusToProto(any()) } returns IdeaStatus.IDEA_STATUS_ACTIVE
        every { mapper.mapIdeaFromProto(any()) } returns expectedIdea
        coEvery { stub.updateIdea(any()) } returns grpcResponse

        // When
        val result = repository.updateIdea(updateRequest)

        // Then
        assertTrue(result.isSuccess)
        assertEquals(expectedIdea, result.getOrNull())
        
        coVerify { stub.updateIdea(any()) }
    }

    @Test
    fun `deleteIdea should succeed when authorized`() = runTest {
        // Given
        val ideaId = "idea-123"
        val userId = "user-123"
        
        val grpcResponse = mockk<DeleteIdeaResponse> {
            every { success } returns true
        }

        every { userPrefs.getUserId() } returns flowOf(userId)
        coEvery { stub.deleteIdea(any()) } returns grpcResponse

        // When
        val result = repository.deleteIdea(ideaId)

        // Then
        assertTrue(result.isSuccess)
        
        coVerify { stub.deleteIdea(any()) }
    }

    @Test
    fun `searchIdeas should filter ideas by query`() = runTest {
        // Given
        val query = "test"
        val allIdeas = listOf(
            createTestIdea().copy(title = "Test Idea"),
            createTestIdea().copy(title = "Another Idea", content = "test content"),
            createTestIdea().copy(title = "Different", content = "Something else", tags = listOf("test")),
            createTestIdea().copy(title = "Unrelated", content = "Random content", tags = listOf("other"))
        )

        val grpcIdeas = allIdeas.map { mockk<com.example.notebook.grpc.Idea>() }
        val grpcResponse = mockk<ListIdeasResponse> {
            every { success } returns true
            every { ideasList } returns grpcIdeas
        }

        every { userPrefs.getUserId() } returns flowOf("user-123")
        every { mapper.mapIdeaFromProto(any()) } returnsMany allIdeas
        coEvery { stub.listIdeas(any()) } returns grpcResponse

        // When
        val result = repository.searchIdeas(query)

        // Then
        assertTrue(result.isSuccess)
        val filteredIdeas = result.getOrNull()!!
        assertEquals(3, filteredIdeas.size) // Should match title, content, or tags
        
        // Verify all returned ideas contain the query
        filteredIdeas.forEach { idea ->
            val containsQuery = idea.title.contains(query, ignoreCase = true) ||
                               idea.content.contains(query, ignoreCase = true) ||
                               idea.tags.any { it.contains(query, ignoreCase = true) }
            assertTrue(containsQuery, "Idea should contain query: $query")
        }
    }

    private fun createTestIdea(): Idea {
        return Idea(
            id = "test-id",
            title = "Test Idea",
            content = "Test content",
            tags = listOf("test"),
            category = IdeaCategory.BUSINESS,
            status = IdeaStatus.DRAFT,
            createdAt = kotlinx.datetime.Clock.System.now(),
            updatedAt = kotlinx.datetime.Clock.System.now(),
            userId = "user-123",
            relatedIdeas = emptyList(),
            priority = 5
        )
    }
}

// Generic test utility class for repository testing
abstract class RepositoryTestBase<T, R> {
    
    abstract fun createTestEntity(): T
    abstract fun createTestRequest(): R
    abstract fun getRepository(): Any
    
    companion object {
        fun <T> assertResultSuccess(result: Result<T>, expectedValue: T? = null) {
            assertTrue(result.isSuccess, "Result should be successful")
            expectedValue?.let {
                assertEquals(it, result.getOrNull())
            }
        }
        
        fun <T> assertResultFailure(result: Result<T>, expectedErrorMessage: String? = null) {
            assertTrue(result.isFailure, "Result should be failure")
            expectedErrorMessage?.let {
                assertEquals(it, result.exceptionOrNull()?.message)
            }
        }
    }
}