package com.example.notebook.data.local

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.*
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

private val Context.userPreferencesDataStore: DataStore<Preferences> by preferencesDataStore(
    name = "user_preferences"
)

@Singleton
class UserPreferencesDataStore @Inject constructor(
    @ApplicationContext private val context: Context
) {
    private val dataStore = context.userPreferencesDataStore

    // Keys
    companion object {
        private val USER_ID_KEY = stringPreferencesKey("user_id")
        private val USER_NAME_KEY = stringPreferencesKey("user_name")
        private val SERVER_HOST_KEY = stringPreferencesKey("server_host")
        private val SERVER_PORT_KEY = intPreferencesKey("server_port")
        private val USE_TLS_KEY = booleanPreferencesKey("use_tls")
        private val THEME_MODE_KEY = stringPreferencesKey("theme_mode")
        private val LANGUAGE_KEY = stringPreferencesKey("language")
        private val NOTIFICATIONS_ENABLED_KEY = booleanPreferencesKey("notifications_enabled")
        private val SYNC_ENABLED_KEY = booleanPreferencesKey("sync_enabled")
        private val LAST_SYNC_TIME_KEY = longPreferencesKey("last_sync_time")
    }

    // User ID
    fun getUserId(): Flow<String> = dataStore.data.map { preferences ->
        preferences[USER_ID_KEY] ?: generateDefaultUserId()
    }

    suspend fun setUserId(userId: String) {
        dataStore.edit { preferences ->
            preferences[USER_ID_KEY] = userId
        }
    }

    // User Name
    fun getUserName(): Flow<String> = dataStore.data.map { preferences ->
        preferences[USER_NAME_KEY] ?: "Usuario"
    }

    suspend fun setUserName(userName: String) {
        dataStore.edit { preferences ->
            preferences[USER_NAME_KEY] = userName
        }
    }

    // Server Configuration
    fun getServerHost(): Flow<String> = dataStore.data.map { preferences ->
        preferences[SERVER_HOST_KEY] ?: "10.0.2.2" // Default for Android emulator
    }

    suspend fun setServerHost(host: String) {
        dataStore.edit { preferences ->
            preferences[SERVER_HOST_KEY] = host
        }
    }

    fun getServerPort(): Flow<Int> = dataStore.data.map { preferences ->
        preferences[SERVER_PORT_KEY] ?: 50051
    }

    suspend fun setServerPort(port: Int) {
        dataStore.edit { preferences ->
            preferences[SERVER_PORT_KEY] = port
        }
    }

    fun getUseTls(): Flow<Boolean> = dataStore.data.map { preferences ->
        preferences[USE_TLS_KEY] ?: false
    }

    suspend fun setUseTls(useTls: Boolean) {
        dataStore.edit { preferences ->
            preferences[USE_TLS_KEY] = useTls
        }
    }

    // Theme and UI
    fun getThemeMode(): Flow<String> = dataStore.data.map { preferences ->
        preferences[THEME_MODE_KEY] ?: "system"
    }

    suspend fun setThemeMode(themeMode: String) {
        dataStore.edit { preferences ->
            preferences[THEME_MODE_KEY] = themeMode
        }
    }

    fun getLanguage(): Flow<String> = dataStore.data.map { preferences ->
        preferences[LANGUAGE_KEY] ?: "es"
    }

    suspend fun setLanguage(language: String) {
        dataStore.edit { preferences ->
            preferences[LANGUAGE_KEY] = language
        }
    }

    // Notifications
    fun getNotificationsEnabled(): Flow<Boolean> = dataStore.data.map { preferences ->
        preferences[NOTIFICATIONS_ENABLED_KEY] ?: true
    }

    suspend fun setNotificationsEnabled(enabled: Boolean) {
        dataStore.edit { preferences ->
            preferences[NOTIFICATIONS_ENABLED_KEY] = enabled
        }
    }

    // Sync
    fun getSyncEnabled(): Flow<Boolean> = dataStore.data.map { preferences ->
        preferences[SYNC_ENABLED_KEY] ?: true
    }

    suspend fun setSyncEnabled(enabled: Boolean) {
        dataStore.edit { preferences ->
            preferences[SYNC_ENABLED_KEY] = enabled
        }
    }

    fun getLastSyncTime(): Flow<Long> = dataStore.data.map { preferences ->
        preferences[LAST_SYNC_TIME_KEY] ?: 0L
    }

    suspend fun setLastSyncTime(timestamp: Long) {
        dataStore.edit { preferences ->
            preferences[LAST_SYNC_TIME_KEY] = timestamp
        }
    }

    // Composite flows for complex data
    data class UserPreferences(
        val userId: String,
        val userName: String,
        val serverHost: String,
        val serverPort: Int,
        val useTls: Boolean,
        val themeMode: String,
        val language: String,
        val notificationsEnabled: Boolean,
        val syncEnabled: Boolean,
        val lastSyncTime: Long
    )

    fun getAllPreferences(): Flow<UserPreferences> = dataStore.data.map { preferences ->
        UserPreferences(
            userId = preferences[USER_ID_KEY] ?: generateDefaultUserId(),
            userName = preferences[USER_NAME_KEY] ?: "Usuario",
            serverHost = preferences[SERVER_HOST_KEY] ?: "10.0.2.2",
            serverPort = preferences[SERVER_PORT_KEY] ?: 50051,
            useTls = preferences[USE_TLS_KEY] ?: false,
            themeMode = preferences[THEME_MODE_KEY] ?: "system",
            language = preferences[LANGUAGE_KEY] ?: "es",
            notificationsEnabled = preferences[NOTIFICATIONS_ENABLED_KEY] ?: true,
            syncEnabled = preferences[SYNC_ENABLED_KEY] ?: true,
            lastSyncTime = preferences[LAST_SYNC_TIME_KEY] ?: 0L
        )
    }

    suspend fun updateAllPreferences(preferences: UserPreferences) {
        dataStore.edit { prefs ->
            prefs[USER_ID_KEY] = preferences.userId
            prefs[USER_NAME_KEY] = preferences.userName
            prefs[SERVER_HOST_KEY] = preferences.serverHost
            prefs[SERVER_PORT_KEY] = preferences.serverPort
            prefs[USE_TLS_KEY] = preferences.useTls
            prefs[THEME_MODE_KEY] = preferences.themeMode
            prefs[LANGUAGE_KEY] = preferences.language
            prefs[NOTIFICATIONS_ENABLED_KEY] = preferences.notificationsEnabled
            prefs[SYNC_ENABLED_KEY] = preferences.syncEnabled
            prefs[LAST_SYNC_TIME_KEY] = preferences.lastSyncTime
        }
    }

    suspend fun clearAllPreferences() {
        dataStore.edit { preferences ->
            preferences.clear()
        }
    }

    private fun generateDefaultUserId(): String {
        return "user_${System.currentTimeMillis()}"
    }
}

// Generic DataStore wrapper for type safety
abstract class TypedDataStore<T> @Inject constructor(
    @ApplicationContext private val context: Context,
    private val dataStoreName: String
) {
    protected abstract val dataStore: DataStore<Preferences>
    
    abstract fun getData(): Flow<T>
    abstract suspend fun updateData(data: T)
    abstract suspend fun clearData()
    
    protected suspend fun <V> updatePreference(key: Preferences.Key<V>, value: V) {
        dataStore.edit { preferences ->
            preferences[key] = value
        }
    }
    
    protected fun <V> getPreference(key: Preferences.Key<V>, defaultValue: V): Flow<V> {
        return dataStore.data.map { preferences ->
            preferences[key] ?: defaultValue
        }
    }
}