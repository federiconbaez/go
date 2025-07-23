package com.example.notebook.data.remote

import com.example.notebook.grpc.NotebookServiceGrpcKt
import io.grpc.ManagedChannel
import io.grpc.ManagedChannelBuilder
import io.grpc.android.AndroidChannelBuilder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.asExecutor
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class GrpcClient @Inject constructor() {
    
    private var _channel: ManagedChannel? = null
    private var _stub: NotebookServiceGrpcKt.NotebookServiceCoroutineStub? = null
    
    private val channel: ManagedChannel
        get() = _channel ?: throw IllegalStateException("gRPC client not initialized")
    
    val stub: NotebookServiceGrpcKt.NotebookServiceCoroutineStub
        get() = _stub ?: throw IllegalStateException("gRPC client not initialized")
    
    fun initialize(host: String, port: Int, useTls: Boolean = false) {
        if (_channel != null) {
            shutdown()
        }
        
        val channelBuilder = if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.N) {
            AndroidChannelBuilder.forAddress(host, port)
        } else {
            ManagedChannelBuilder.forAddress(host, port)
        }
        
        if (!useTls) {
            channelBuilder.usePlaintext()
        }
        
        _channel = channelBuilder
            .executor(Dispatchers.IO.asExecutor())
            .build()
        
        _stub = NotebookServiceGrpcKt.NotebookServiceCoroutineStub(_channel!!)
    }
    
    fun shutdown() {
        _channel?.shutdown()
        _channel = null
        _stub = null
    }
    
    fun isInitialized(): Boolean = _channel != null && _stub != null
}