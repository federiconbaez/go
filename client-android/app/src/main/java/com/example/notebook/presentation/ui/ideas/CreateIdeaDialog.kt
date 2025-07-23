package com.example.notebook.presentation.ui.ideas

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.example.notebook.domain.model.CreateIdeaRequest
import com.example.notebook.domain.model.IdeaCategory

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreateIdeaDialog(
    onDismiss: () -> Unit,
    onConfirm: (CreateIdeaRequest) -> Unit,
    isCreating: Boolean = false
) {
    var title by remember { mutableStateOf("") }
    var content by remember { mutableStateOf("") }
    var tags by remember { mutableStateOf("") }
    var selectedCategory by remember { mutableStateOf(IdeaCategory.UNSPECIFIED) }
    var priority by remember { mutableStateOf("5") }
    var showCategoryDropdown by remember { mutableStateOf(false) }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(
            dismissOnBackPress = !isCreating,
            dismissOnClickOutside = !isCreating
        )
    ) {
        Card(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            elevation = CardDefaults.cardElevation(defaultElevation = 8.dp)
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(24.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Text(
                    text = "Crear Nueva Idea",
                    style = MaterialTheme.typography.headlineSmall,
                    color = MaterialTheme.colorScheme.onSurface
                )

                // Title Field
                OutlinedTextField(
                    value = title,
                    onValueChange = { title = it },
                    label = { Text("Título *") },
                    enabled = !isCreating,
                    isError = title.isBlank(),
                    supportingText = {
                        if (title.isBlank()) {
                            Text("El título es requerido")
                        }
                    },
                    modifier = Modifier.fillMaxWidth()
                )

                // Content Field
                OutlinedTextField(
                    value = content,
                    onValueChange = { content = it },
                    label = { Text("Contenido *") },
                    enabled = !isCreating,
                    isError = content.isBlank(),
                    supportingText = {
                        if (content.isBlank()) {
                            Text("El contenido es requerido")
                        }
                    },
                    minLines = 3,
                    maxLines = 6,
                    modifier = Modifier.fillMaxWidth()
                )

                // Category Dropdown
                ExposedDropdownMenuBox(
                    expanded = showCategoryDropdown,
                    onExpandedChange = { showCategoryDropdown = !isCreating && it }
                ) {
                    OutlinedTextField(
                        value = selectedCategory.displayName,
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("Categoría") },
                        trailingIcon = {
                            ExposedDropdownMenuDefaults.TrailingIcon(
                                expanded = showCategoryDropdown
                            )
                        },
                        enabled = !isCreating,
                        modifier = Modifier
                            .fillMaxWidth()
                            .menuAnchor()
                    )

                    ExposedDropdownMenu(
                        expanded = showCategoryDropdown,
                        onDismissRequest = { showCategoryDropdown = false }
                    ) {
                        IdeaCategory.values().forEach { category ->
                            DropdownMenuItem(
                                text = { Text(category.displayName) },
                                onClick = {
                                    selectedCategory = category
                                    showCategoryDropdown = false
                                }
                            )
                        }
                    }
                }

                // Tags Field
                OutlinedTextField(
                    value = tags,
                    onValueChange = { tags = it },
                    label = { Text("Etiquetas") },
                    enabled = !isCreating,
                    supportingText = {
                        Text("Separa las etiquetas con comas")
                    },
                    modifier = Modifier.fillMaxWidth()
                )

                // Priority Field
                OutlinedTextField(
                    value = priority,
                    onValueChange = { newValue ->
                        if (newValue.isEmpty() || (newValue.toIntOrNull()?.let { it in 1..10 } == true)) {
                            priority = newValue
                        }
                    },
                    label = { Text("Prioridad (1-10)") },
                    enabled = !isCreating,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    isError = priority.toIntOrNull()?.let { it !in 1..10 } ?: true,
                    supportingText = {
                        if (priority.toIntOrNull()?.let { it !in 1..10 } != false) {
                            Text("Debe ser un número entre 1 y 10")
                        }
                    },
                    modifier = Modifier.fillMaxWidth()
                )

                // Action Buttons
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp, Arrangement.End)
                ) {
                    TextButton(
                        onClick = onDismiss,
                        enabled = !isCreating
                    ) {
                        Text("Cancelar")
                    }

                    Button(
                        onClick = {
                            if (title.isNotBlank() && content.isNotBlank()) {
                                val tagsList = tags.split(",")
                                    .map { it.trim() }
                                    .filter { it.isNotEmpty() }

                                val request = CreateIdeaRequest(
                                    title = title,
                                    content = content,
                                    tags = tagsList,
                                    category = selectedCategory,
                                    priority = priority.toIntOrNull() ?: 5
                                )
                                onConfirm(request)
                            }
                        },
                        enabled = !isCreating && title.isNotBlank() && content.isNotBlank() && 
                                 (priority.toIntOrNull()?.let { it in 1..10 } == true)
                    ) {
                        if (isCreating) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(16.dp),
                                strokeWidth = 2.dp
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                        }
                        Text("Crear")
                    }
                }
            }
        }
    }
}