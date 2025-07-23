package com.example.notebook.data.local

import android.content.Context
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit

class CacheManager private constructor(private val context: Context) {
    
    companion object {
        @Volatile
        private var INSTANCE: CacheManager? = null
        
        fun getInstance(context: Context): CacheManager {
            return INSTANCE ?: synchronized(this) {
                INSTANCE ?: CacheManager(context.applicationContext).also { INSTANCE = it }
            }
        }
        
        private const val DEFAULT_EXPIRY_MINUTES = 30
        private const val MAX_CACHE_SIZE = 100
    }
    
    private data class CacheEntry<T>(
        val data: T,
        val timestamp: Long,
        val expiryMillis: Long
    ) {
        fun isExpired(): Boolean = System.currentTimeMillis() > timestamp + expiryMillis
    }
    
    private val cache = ConcurrentHashMap<String, CacheEntry<*>>()
    private val mutex = Mutex()
    
    suspend fun <T> get(key: String): T? = mutex.withLock {
        val entry = cache[key] as? CacheEntry<T>
        when {
            entry == null -> null
            entry.isExpired() -> {
                cache.remove(key)
                null
            }
            else -> entry.data
        }
    }
    
    suspend fun <T> put(
        key: String, 
        data: T, 
        expiryMinutes: Int = DEFAULT_EXPIRY_MINUTES
    ) = mutex.withLock {
        if (cache.size >= MAX_CACHE_SIZE) {
            evictOldestEntries()
        }
        
        val expiryMillis = TimeUnit.MINUTES.toMillis(expiryMinutes.toLong())
        cache[key] = CacheEntry(data, System.currentTimeMillis(), expiryMillis)
    }
    
    suspend fun invalidate(key: String) = mutex.withLock {
        cache.remove(key)
    }
    
    suspend fun clear() = mutex.withLock {
        cache.clear()
    }
    
    private fun evictOldestEntries() {
        val sortedEntries = cache.entries.sortedBy { it.value.timestamp }
        val toRemove = sortedEntries.take(cache.size / 4)
        toRemove.forEach { cache.remove(it.key) }
    }
    
    suspend fun getMemoryStats(): CacheStats = mutex.withLock {
        val now = System.currentTimeMillis()
        val expiredCount = cache.values.count { it.isExpired() }
        
        CacheStats(
            totalEntries = cache.size,
            expiredEntries = expiredCount,
            validEntries = cache.size - expiredCount,
            memoryUsageKB = estimateMemoryUsage()
        )
    }
    
    private fun estimateMemoryUsage(): Long {
        return cache.size * 1024L
    }
    
    data class CacheStats(
        val totalEntries: Int,
        val expiredEntries: Int,
        val validEntries: Int,
        val memoryUsageKB: Long
    )
}