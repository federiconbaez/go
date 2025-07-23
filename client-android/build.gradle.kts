// Top-level build file where you can add configuration options common to all sub-projects/modules.
buildscript {
    extra["kotlin_version"] = "1.9.10"
    extra["compose_version"] = "1.5.4"
    extra["hilt_version"] = "2.48"
    
    dependencies {
        classpath("com.google.dagger:hilt-android-gradle-plugin:${extra["hilt_version"]}")
        classpath("com.google.protobuf:protobuf-gradle-plugin:0.9.4")
    }
}

plugins {
    id("com.android.application") version "8.1.2" apply false
    id("org.jetbrains.kotlin.android") version "1.9.10" apply false
    id("com.google.dagger.hilt.android") version "2.48" apply false
    id("com.google.protobuf") version "0.9.4" apply false
}

tasks.register("clean", Delete::class) {
    delete(rootProject.buildDir)
}