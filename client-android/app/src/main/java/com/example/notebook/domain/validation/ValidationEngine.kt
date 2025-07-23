package com.example.notebook.domain.validation

import java.util.regex.Pattern
import kotlin.reflect.KProperty1
import kotlin.reflect.full.memberProperties

class ValidationEngine {
    
    sealed class ValidationResult {
        object Valid : ValidationResult()
        data class Invalid(val errors: List<ValidationError>) : ValidationResult()
        
        val isValid: Boolean get() = this is Valid
        val isInvalid: Boolean get() = this is Invalid
    }
    
    data class ValidationError(
        val field: String,
        val message: String,
        val code: String,
        val severity: Severity = Severity.ERROR
    ) {
        enum class Severity { ERROR, WARNING, INFO }
    }
    
    interface ValidationRule<T> {
        fun validate(value: T, context: ValidationContext): ValidationResult
        val errorCode: String
        val errorMessage: String
    }
    
    data class ValidationContext(
        val fieldName: String = "",
        val objectType: String = "",
        val customData: Map<String, Any> = emptyMap()
    )
    
    class RequiredRule<T> : ValidationRule<T?> {
        override val errorCode = "REQUIRED"
        override val errorMessage = "Field is required"
        
        override fun validate(value: T?, context: ValidationContext): ValidationResult {
            return when {
                value == null -> ValidationResult.Invalid(
                    listOf(ValidationError(context.fieldName, errorMessage, errorCode))
                )
                value is String && value.isBlank() -> ValidationResult.Invalid(
                    listOf(ValidationError(context.fieldName, "Field cannot be empty", errorCode))
                )
                value is Collection<*> && value.isEmpty() -> ValidationResult.Invalid(
                    listOf(ValidationError(context.fieldName, "Collection cannot be empty", errorCode))
                )
                else -> ValidationResult.Valid
            }
        }
    }
    
    class LengthRule(
        private val min: Int? = null,
        private val max: Int? = null
    ) : ValidationRule<String?> {
        override val errorCode = "LENGTH"
        override val errorMessage = "Invalid length"
        
        override fun validate(value: String?, context: ValidationContext): ValidationResult {
            if (value == null) return ValidationResult.Valid
            
            val errors = mutableListOf<ValidationError>()
            val length = value.length
            
            min?.let { minLen ->
                if (length < minLen) {
                    errors.add(ValidationError(
                        context.fieldName,
                        "Minimum length is $minLen characters",
                        "${errorCode}_MIN"
                    ))
                }
            }
            
            max?.let { maxLen ->
                if (length > maxLen) {
                    errors.add(ValidationError(
                        context.fieldName,
                        "Maximum length is $maxLen characters",
                        "${errorCode}_MAX"
                    ))
                }
            }
            
            return if (errors.isEmpty()) ValidationResult.Valid 
                   else ValidationResult.Invalid(errors)
        }
    }
    
    class PatternRule(
        private val pattern: String,
        private val patternName: String = "pattern"
    ) : ValidationRule<String?> {
        private val compiledPattern = Pattern.compile(pattern)
        override val errorCode = "PATTERN"
        override val errorMessage = "Invalid $patternName format"
        
        override fun validate(value: String?, context: ValidationContext): ValidationResult {
            if (value == null) return ValidationResult.Valid
            
            return if (compiledPattern.matcher(value).matches()) {
                ValidationResult.Valid
            } else {
                ValidationResult.Invalid(
                    listOf(ValidationError(context.fieldName, errorMessage, errorCode))
                )
            }
        }
    }
    
    class RangeRule<T : Comparable<T>>(
        private val min: T? = null,
        private val max: T? = null
    ) : ValidationRule<T?> {
        override val errorCode = "RANGE"
        override val errorMessage = "Value out of range"
        
        override fun validate(value: T?, context: ValidationContext): ValidationResult {
            if (value == null) return ValidationResult.Valid
            
            val errors = mutableListOf<ValidationError>()
            
            min?.let { minVal ->
                if (value < minVal) {
                    errors.add(ValidationError(
                        context.fieldName,
                        "Value must be at least $minVal",
                        "${errorCode}_MIN"
                    ))
                }
            }
            
            max?.let { maxVal ->
                if (value > maxVal) {
                    errors.add(ValidationError(
                        context.fieldName,
                        "Value must be at most $maxVal",
                        "${errorCode}_MAX"
                    ))
                }
            }
            
            return if (errors.isEmpty()) ValidationResult.Valid 
                   else ValidationResult.Invalid(errors)
        }
    }
    
    class EmailRule : ValidationRule<String?> {
        private val emailPattern = Pattern.compile(
            "^[A-Za-z0-9+_.-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$"
        )
        override val errorCode = "EMAIL"
        override val errorMessage = "Invalid email format"
        
        override fun validate(value: String?, context: ValidationContext): ValidationResult {
            if (value == null) return ValidationResult.Valid
            
            return if (emailPattern.matcher(value).matches()) {
                ValidationResult.Valid
            } else {
                ValidationResult.Invalid(
                    listOf(ValidationError(context.fieldName, errorMessage, errorCode))
                )
            }
        }
    }
    
    class CustomRule<T>(
        private val predicate: (T?) -> Boolean,
        override val errorCode: String,
        override val errorMessage: String
    ) : ValidationRule<T?> {
        override fun validate(value: T?, context: ValidationContext): ValidationResult {
            return if (predicate(value)) {
                ValidationResult.Valid
            } else {
                ValidationResult.Invalid(
                    listOf(ValidationError(context.fieldName, errorMessage, errorCode))
                )
            }
        }
    }
    
    class ValidationSchema<T : Any> {
        private val fieldRules = mutableMapOf<String, MutableList<ValidationRule<Any?>>>()
        
        fun <V> field(property: KProperty1<T, V>, vararg rules: ValidationRule<V>) {
            val fieldName = property.name
            fieldRules.getOrPut(fieldName) { mutableListOf() }
                .addAll(rules.map { rule -> rule as ValidationRule<Any?> })
        }
        
        fun validate(obj: T): ValidationResult {
            val allErrors = mutableListOf<ValidationError>()
            val objClass = obj::class
            
            fieldRules.forEach { (fieldName, rules) ->
                val property = objClass.memberProperties.find { it.name == fieldName }
                val fieldValue = property?.call(obj)
                val context = ValidationContext(
                    fieldName = fieldName,
                    objectType = objClass.simpleName ?: "Unknown"
                )
                
                rules.forEach { rule ->
                    val result = rule.validate(fieldValue, context)
                    if (result is ValidationResult.Invalid) {
                        allErrors.addAll(result.errors)
                    }
                }
            }
            
            return if (allErrors.isEmpty()) ValidationResult.Valid 
                   else ValidationResult.Invalid(allErrors)
        }
    }
    
    companion object {
        fun <T : Any> schema(block: ValidationSchema<T>.() -> Unit): ValidationSchema<T> {
            return ValidationSchema<T>().apply(block)
        }
        
        fun required() = RequiredRule<Any>()
        fun length(min: Int? = null, max: Int? = null) = LengthRule(min, max)
        fun pattern(regex: String, name: String = "pattern") = PatternRule(regex, name)
        fun <T : Comparable<T>> range(min: T? = null, max: T? = null) = RangeRule(min, max)
        fun email() = EmailRule()
        fun <T> custom(
            predicate: (T?) -> Boolean,
            errorCode: String,
            errorMessage: String
        ) = CustomRule(predicate, errorCode, errorMessage)
    }
}