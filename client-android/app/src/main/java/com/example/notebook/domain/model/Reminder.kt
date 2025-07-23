package com.example.notebook.domain.model

import kotlinx.datetime.Instant
import kotlinx.parcelize.Parcelize
import android.os.Parcelable

@Parcelize
data class Reminder(
    val id: String,
    val title: String,
    val description: String,
    val scheduledTime: Instant,
    val type: ReminderType,
    val status: ReminderStatus,
    val recurring: Boolean,
    val recurrencePattern: RecurrencePattern,
    val createdAt: Instant,
    val updatedAt: Instant,
    val userId: String,
    val notificationChannels: List<String>
) : Parcelable

enum class ReminderType(val displayName: String) {
    UNSPECIFIED("Sin especificar"),
    TASK("Tarea"),
    MEETING("Reunión"),
    DEADLINE("Fecha límite"),
    EVENT("Evento"),
    CALL("Llamada")
}

enum class ReminderStatus(val displayName: String) {
    UNSPECIFIED("Sin especificar"),
    PENDING("Pendiente"),
    ACTIVE("Activo"),
    COMPLETED("Completado"),
    CANCELLED("Cancelado"),
    OVERDUE("Vencido")
}

enum class RecurrencePattern(val displayName: String) {
    UNSPECIFIED("Sin especificar"),
    DAILY("Diario"),
    WEEKLY("Semanal"),
    MONTHLY("Mensual"),
    YEARLY("Anual"),
    CUSTOM("Personalizado")
}

data class CreateReminderRequest(
    val title: String,
    val description: String,
    val scheduledTime: Instant,
    val type: ReminderType,
    val recurring: Boolean,
    val recurrencePattern: RecurrencePattern,
    val notificationChannels: List<String>
)

data class UpdateReminderRequest(
    val id: String,
    val title: String?,
    val description: String?,
    val scheduledTime: Instant?,
    val type: ReminderType?,
    val status: ReminderStatus?,
    val recurring: Boolean?,
    val recurrencePattern: RecurrencePattern?
)

data class ReminderFilters(
    val type: ReminderType? = null,
    val status: ReminderStatus? = null,
    val fromDate: Instant? = null,
    val toDate: Instant? = null,
    val page: Int = 1,
    val pageSize: Int = 10
)