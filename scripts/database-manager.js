#!/usr/bin/env node
/**
 * Advanced Database Management System
 * Provides comprehensive database operations, migrations, backups, and monitoring
 * Author: Auto-generated Advanced Logic
 * Version: 1.0.0
 */

const fs = require('fs').promises;
const path = require('path');
const crypto = require('crypto');
const { execSync, spawn } = require('child_process');
const EventEmitter = require('events');

// Database clients (install with: npm install pg mysql2 mongodb redis sqlite3)
let pgClient, mysqlClient, mongoClient, redisClient, sqliteClient;

class DatabaseManager extends EventEmitter {
    constructor(config = {}) {
        super();
        this.config = {
            connections: {
                postgresql: {
                    host: process.env.POSTGRES_HOST || 'localhost',
                    port: process.env.POSTGRES_PORT || 5432,
                    database: process.env.POSTGRES_DB || 'notebook',
                    username: process.env.POSTGRES_USER || 'postgres',
                    password: process.env.POSTGRES_PASSWORD || 'password',
                    ssl: process.env.POSTGRES_SSL === 'true'
                },
                mysql: {
                    host: process.env.MYSQL_HOST || 'localhost',
                    port: process.env.MYSQL_PORT || 3306,
                    database: process.env.MYSQL_DB || 'notebook',
                    username: process.env.MYSQL_USER || 'root',
                    password: process.env.MYSQL_PASSWORD || 'password'
                },
                mongodb: {
                    url: process.env.MONGO_URL || 'mongodb://localhost:27017/notebook',
                    options: {
                        useNewUrlParser: true,
                        useUnifiedTopology: true
                    }
                },
                redis: {
                    host: process.env.REDIS_HOST || 'localhost',
                    port: process.env.REDIS_PORT || 6379,
                    password: process.env.REDIS_PASSWORD || null,
                    db: process.env.REDIS_DB || 0
                },
                sqlite: {
                    database: process.env.SQLITE_DB || './data/notebook.db'
                }
            },
            backup: {
                enabled: true,
                schedule: '0 2 * * *', // Daily at 2 AM
                retention: 30, // Keep 30 days
                compression: true,
                encryption: true,
                storage: {
                    local: './backups',
                    s3: {
                        bucket: process.env.S3_BACKUP_BUCKET,
                        region: process.env.S3_REGION || 'us-west-2'
                    }
                }
            },
            monitoring: {
                enabled: true,
                metrics: {
                    connections: true,
                    queries: true,
                    performance: true,
                    errors: true
                },
                alerts: {
                    slowQueries: 1000, // ms
                    highConnections: 80, // % of max
                    diskSpace: 90, // % usage
                    replicationLag: 5000 // ms
                }
            },
            security: {
                encryption: {
                    enabled: true,
                    algorithm: 'aes-256-gcm',
                    keyRotation: 90 // days
                },
                audit: {
                    enabled: true,
                    logQueries: true,
                    logConnections: true
                }
            },
            ...config
        };

        this.connections = new Map();
        this.metrics = {
            connections: 0,
            queries: 0,
            errors: 0,
            slowQueries: 0,
            lastBackup: null,
            uptime: Date.now()
        };
        
        this.migrationHistory = [];
        this.backupQueue = [];
        this.isRunning = false;
        
        this.setupLogging();
        this.initializeClients();
    }

    setupLogging() {
        this.log = {
            info: (msg, ...args) => console.log(`[INFO] ${new Date().toISOString()} - ${msg}`, ...args),
            warn: (msg, ...args) => console.warn(`[WARN] ${new Date().toISOString()} - ${msg}`, ...args),
            error: (msg, ...args) => console.error(`[ERROR] ${new Date().toISOString()} - ${msg}`, ...args),
            debug: (msg, ...args) => process.env.DEBUG && console.log(`[DEBUG] ${new Date().toISOString()} - ${msg}`, ...args)
        };
    }

    async initializeClients() {
        try {
            // PostgreSQL
            if (this.config.connections.postgresql) {
                try {
                    const { Client } = require('pg');
                    pgClient = new Client(this.config.connections.postgresql);
                    await pgClient.connect();
                    this.connections.set('postgresql', pgClient);
                    this.log.info('PostgreSQL connection established');
                } catch (error) {
                    this.log.warn('PostgreSQL connection failed:', error.message);
                }
            }

            // MySQL
            if (this.config.connections.mysql) {
                try {
                    const mysql = require('mysql2/promise');
                    mysqlClient = await mysql.createConnection(this.config.connections.mysql);
                    this.connections.set('mysql', mysqlClient);
                    this.log.info('MySQL connection established');
                } catch (error) {
                    this.log.warn('MySQL connection failed:', error.message);
                }
            }

            // MongoDB
            if (this.config.connections.mongodb) {
                try {
                    const { MongoClient } = require('mongodb');
                    mongoClient = new MongoClient(this.config.connections.mongodb.url, this.config.connections.mongodb.options);
                    await mongoClient.connect();
                    this.connections.set('mongodb', mongoClient);
                    this.log.info('MongoDB connection established');
                } catch (error) {
                    this.log.warn('MongoDB connection failed:', error.message);
                }
            }

            // Redis
            if (this.config.connections.redis) {
                try {
                    const redis = require('redis');
                    redisClient = redis.createClient(this.config.connections.redis);
                    await redisClient.connect();
                    this.connections.set('redis', redisClient);
                    this.log.info('Redis connection established');
                } catch (error) {
                    this.log.warn('Redis connection failed:', error.message);
                }
            }

            // SQLite
            if (this.config.connections.sqlite) {
                try {
                    const sqlite3 = require('sqlite3').verbose();
                    const { promisify } = require('util');
                    
                    sqliteClient = new sqlite3.Database(this.config.connections.sqlite.database);
                    sqliteClient.runAsync = promisify(sqliteClient.run.bind(sqliteClient));
                    sqliteClient.getAsync = promisify(sqliteClient.get.bind(sqliteClient));
                    sqliteClient.allAsync = promisify(sqliteClient.all.bind(sqliteClient));
                    
                    this.connections.set('sqlite', sqliteClient);
                    this.log.info('SQLite connection established');
                } catch (error) {
                    this.log.warn('SQLite connection failed:', error.message);
                }
            }

        } catch (error) {
            this.log.error('Error initializing database clients:', error);
            throw error;
        }
    }

    // Migration Management
    async runMigrations(database = 'postgresql', migrationsPath = './migrations') {
        this.log.info(`Running migrations for ${database}...`);
        
        try {
            const client = this.connections.get(database);
            if (!client) {
                throw new Error(`Database ${database} not connected`);
            }

            // Ensure migrations table exists
            await this.createMigrationsTable(database);

            // Get executed migrations
            const executedMigrations = await this.getExecutedMigrations(database);
            
            // Read migration files
            const migrationFiles = await this.getMigrationFiles(migrationsPath);
            
            // Execute pending migrations
            for (const file of migrationFiles) {
                if (!executedMigrations.includes(file.name)) {
                    await this.executeMigration(database, file);
                    await this.recordMigration(database, file);
                    this.log.info(`Migration executed: ${file.name}`);
                }
            }

            this.log.info('All migrations completed successfully');
            
        } catch (error) {
            this.log.error('Migration failed:', error);
            throw error;
        }
    }

    async createMigrationsTable(database) {
        const client = this.connections.get(database);
        
        const createTableQueries = {
            postgresql: `
                CREATE TABLE IF NOT EXISTS migrations (
                    id SERIAL PRIMARY KEY,
                    name VARCHAR(255) NOT NULL UNIQUE,
                    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    checksum VARCHAR(64)
                )
            `,
            mysql: `
                CREATE TABLE IF NOT EXISTS migrations (
                    id INT AUTO_INCREMENT PRIMARY KEY,
                    name VARCHAR(255) NOT NULL UNIQUE,
                    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    checksum VARCHAR(64)
                )
            `,
            sqlite: `
                CREATE TABLE IF NOT EXISTS migrations (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    name TEXT NOT NULL UNIQUE,
                    executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                    checksum TEXT
                )
            `
        };

        if (database === 'postgresql') {
            await client.query(createTableQueries.postgresql);
        } else if (database === 'mysql') {
            await client.execute(createTableQueries.mysql);
        } else if (database === 'sqlite') {
            await client.runAsync(createTableQueries.sqlite);
        }
    }

    async getExecutedMigrations(database) {
        const client = this.connections.get(database);
        let result;

        if (database === 'postgresql') {
            result = await client.query('SELECT name FROM migrations ORDER BY executed_at');
            return result.rows.map(row => row.name);
        } else if (database === 'mysql') {
            const [rows] = await client.execute('SELECT name FROM migrations ORDER BY executed_at');
            return rows.map(row => row.name);
        } else if (database === 'sqlite') {
            const rows = await client.allAsync('SELECT name FROM migrations ORDER BY executed_at');
            return rows.map(row => row.name);
        }

        return [];
    }

    async getMigrationFiles(migrationsPath) {
        try {
            const files = await fs.readdir(migrationsPath);
            const migrationFiles = [];

            for (const file of files.sort()) {
                if (file.endsWith('.sql')) {
                    const filePath = path.join(migrationsPath, file);
                    const content = await fs.readFile(filePath, 'utf8');
                    const checksum = crypto.createHash('sha256').update(content).digest('hex');
                    
                    migrationFiles.push({
                        name: file,
                        path: filePath,
                        content,
                        checksum
                    });
                }
            }

            return migrationFiles;
        } catch (error) {
            this.log.warn(`Migrations directory not found: ${migrationsPath}`);
            return [];
        }
    }

    async executeMigration(database, migration) {
        const client = this.connections.get(database);
        const startTime = Date.now();

        try {
            if (database === 'postgresql') {
                await client.query(migration.content);
            } else if (database === 'mysql') {
                await client.execute(migration.content);
            } else if (database === 'sqlite') {
                await client.runAsync(migration.content);
            }

            const duration = Date.now() - startTime;
            this.migrationHistory.push({
                name: migration.name,
                executed_at: new Date(),
                duration,
                success: true
            });

        } catch (error) {
            const duration = Date.now() - startTime;
            this.migrationHistory.push({
                name: migration.name,
                executed_at: new Date(),
                duration,
                success: false,
                error: error.message
            });
            throw error;
        }
    }

    async recordMigration(database, migration) {
        const client = this.connections.get(database);
        
        if (database === 'postgresql') {
            await client.query(
                'INSERT INTO migrations (name, checksum) VALUES ($1, $2)',
                [migration.name, migration.checksum]
            );
        } else if (database === 'mysql') {
            await client.execute(
                'INSERT INTO migrations (name, checksum) VALUES (?, ?)',
                [migration.name, migration.checksum]
            );
        } else if (database === 'sqlite') {
            await client.runAsync(
                'INSERT INTO migrations (name, checksum) VALUES (?, ?)',
                [migration.name, migration.checksum]
            );
        }
    }

    // Backup Management
    async createBackup(databases = ['all'], options = {}) {
        const backupId = crypto.randomUUID();
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
        const backupDir = path.join(this.config.backup.storage.local, `backup-${timestamp}-${backupId}`);

        this.log.info(`Creating backup: ${backupId}`);

        try {
            await fs.mkdir(backupDir, { recursive: true });

            const backupManifest = {
                id: backupId,
                timestamp: new Date(),
                databases: [],
                size: 0,
                compressed: this.config.backup.compression,
                encrypted: this.config.backup.encryption
            };

            // Backup each database
            for (const [dbType, client] of this.connections.entries()) {
                if (databases.includes('all') || databases.includes(dbType)) {
                    await this.backupDatabase(dbType, backupDir, backupManifest);
                }
            }

            // Create manifest file
            await fs.writeFile(
                path.join(backupDir, 'manifest.json'),
                JSON.stringify(backupManifest, null, 2)
            );

            // Compress if enabled
            if (this.config.backup.compression) {
                await this.compressBackup(backupDir);
            }

            // Encrypt if enabled
            if (this.config.backup.encryption) {
                await this.encryptBackup(backupDir);
            }

            // Upload to S3 if configured
            if (this.config.backup.storage.s3?.bucket) {
                await this.uploadBackupToS3(backupDir, backupId);
            }

            this.metrics.lastBackup = new Date();
            this.log.info(`Backup completed: ${backupId}`);

            return {
                id: backupId,
                path: backupDir,
                manifest: backupManifest
            };

        } catch (error) {
            this.log.error(`Backup failed: ${error.message}`);
            throw error;
        }
    }

    async backupDatabase(dbType, backupDir, manifest) {
        const backupFile = path.join(backupDir, `${dbType}.sql`);
        
        try {
            switch (dbType) {
                case 'postgresql':
                    await this.backupPostgreSQL(backupFile);
                    break;
                case 'mysql':
                    await this.backupMySQL(backupFile);
                    break;
                case 'mongodb':
                    await this.backupMongoDB(backupDir);
                    break;
                case 'sqlite':
                    await this.backupSQLite(backupFile);
                    break;
                default:
                    this.log.warn(`Backup not implemented for ${dbType}`);
                    return;
            }

            const stats = await fs.stat(backupFile);
            manifest.databases.push({
                type: dbType,
                file: path.basename(backupFile),
                size: stats.size,
                timestamp: new Date()
            });
            manifest.size += stats.size;

        } catch (error) {
            this.log.error(`Failed to backup ${dbType}:`, error.message);
            throw error;
        }
    }

    async backupPostgreSQL(backupFile) {
        const config = this.config.connections.postgresql;
        const command = `pg_dump -h ${config.host} -p ${config.port} -U ${config.username} -d ${config.database} -f ${backupFile}`;
        
        execSync(command, {
            env: { ...process.env, PGPASSWORD: config.password },
            stdio: 'inherit'
        });
    }

    async backupMySQL(backupFile) {
        const config = this.config.connections.mysql;
        const command = `mysqldump -h ${config.host} -P ${config.port} -u ${config.username} -p${config.password} ${config.database} > ${backupFile}`;
        
        execSync(command, { stdio: 'inherit' });
    }

    async backupMongoDB(backupDir) {
        const config = this.config.connections.mongodb;
        const mongoDir = path.join(backupDir, 'mongodb');
        const command = `mongodump --uri="${config.url}" --out=${mongoDir}`;
        
        execSync(command, { stdio: 'inherit' });
    }

    async backupSQLite(backupFile) {
        const config = this.config.connections.sqlite;
        const command = `sqlite3 ${config.database} ".backup ${backupFile}"`;
        
        execSync(command, { stdio: 'inherit' });
    }

    async restoreBackup(backupId, databases = ['all']) {
        this.log.info(`Restoring backup: ${backupId}`);

        try {
            const backupDir = await this.findBackupDir(backupId);
            const manifest = await this.loadBackupManifest(backupDir);

            // Decrypt if needed
            if (manifest.encrypted) {
                await this.decryptBackup(backupDir);
            }

            // Decompress if needed
            if (manifest.compressed) {
                await this.decompressBackup(backupDir);
            }

            // Restore each database
            for (const dbInfo of manifest.databases) {
                if (databases.includes('all') || databases.includes(dbInfo.type)) {
                    await this.restoreDatabase(dbInfo.type, backupDir, dbInfo.file);
                }
            }

            this.log.info(`Backup restored successfully: ${backupId}`);

        } catch (error) {
            this.log.error(`Backup restore failed: ${error.message}`);
            throw error;
        }
    }

    // Performance Monitoring
    async getPerformanceMetrics(database) {
        const client = this.connections.get(database);
        if (!client) {
            throw new Error(`Database ${database} not connected`);
        }

        const metrics = {
            database,
            timestamp: new Date(),
            connections: 0,
            activeQueries: 0,
            slowQueries: 0,
            cacheHitRatio: 0,
            diskUsage: 0,
            indexUsage: {}
        };

        try {
            switch (database) {
                case 'postgresql':
                    return await this.getPostgreSQLMetrics(client, metrics);
                case 'mysql':
                    return await this.getMySQLMetrics(client, metrics);
                case 'mongodb':
                    return await this.getMongoDBMetrics(client, metrics);
                case 'redis':
                    return await this.getRedisMetrics(client, metrics);
                default:
                    return metrics;
            }
        } catch (error) {
            this.log.error(`Failed to get metrics for ${database}:`, error.message);
            return metrics;
        }
    }

    async getPostgreSQLMetrics(client, metrics) {
        // Connection count
        const connResult = await client.query(`
            SELECT count(*) as connections 
            FROM pg_stat_activity 
            WHERE state = 'active'
        `);
        metrics.connections = parseInt(connResult.rows[0].connections);

        // Database size
        const sizeResult = await client.query(`
            SELECT pg_size_pretty(pg_database_size(current_database())) as size,
                   pg_database_size(current_database()) as size_bytes
        `);
        metrics.diskUsage = parseInt(sizeResult.rows[0].size_bytes);

        // Cache hit ratio
        const cacheResult = await client.query(`
            SELECT 
                round(blks_hit*100.0/(blks_hit+blks_read), 2) as cache_hit_ratio
            FROM pg_stat_database 
            WHERE datname = current_database()
        `);
        metrics.cacheHitRatio = parseFloat(cacheResult.rows[0]?.cache_hit_ratio || 0);

        // Index usage
        const indexResult = await client.query(`
            SELECT 
                schemaname,
                tablename,
                indexname,
                idx_scan
            FROM pg_stat_user_indexes
            ORDER BY idx_scan DESC
            LIMIT 10
        `);
        
        metrics.indexUsage = indexResult.rows.reduce((acc, row) => {
            acc[`${row.schemaname}.${row.tablename}.${row.indexname}`] = row.idx_scan;
            return acc;
        }, {});

        return metrics;
    }

    async getMySQLMetrics(client, metrics) {
        // Connection count
        const [connRows] = await client.execute('SHOW STATUS LIKE "Threads_connected"');
        metrics.connections = parseInt(connRows[0].Value);

        // Cache hit ratio
        const [cacheRows] = await client.execute(`
            SHOW STATUS WHERE Variable_name IN 
            ('Qcache_hits', 'Qcache_inserts', 'Qcache_not_cached')
        `);
        
        const cacheStats = cacheRows.reduce((acc, row) => {
            acc[row.Variable_name] = parseInt(row.Value);
            return acc;
        }, {});
        
        const totalQueries = cacheStats.Qcache_hits + cacheStats.Qcache_inserts + cacheStats.Qcache_not_cached;
        metrics.cacheHitRatio = totalQueries > 0 ? (cacheStats.Qcache_hits / totalQueries) * 100 : 0;

        return metrics;
    }

    async getMongoDBMetrics(client, metrics) {
        const db = client.db();
        const admin = db.admin();
        
        // Server status
        const serverStatus = await admin.serverStatus();
        metrics.connections = serverStatus.connections.current;
        metrics.activeQueries = serverStatus.globalLock.activeClients.total;

        // Database stats
        const dbStats = await db.stats();
        metrics.diskUsage = dbStats.dataSize;

        return metrics;
    }

    async getRedisMetrics(client, metrics) {
        const info = await client.info();
        const lines = info.split('\r\n');
        
        for (const line of lines) {
            if (line.startsWith('connected_clients:')) {
                metrics.connections = parseInt(line.split(':')[1]);
            }
            if (line.startsWith('used_memory:')) {
                metrics.diskUsage = parseInt(line.split(':')[1]);
            }
        }

        return metrics;
    }

    // Query Analytics
    async analyzeSlowQueries(database, limit = 10) {
        const client = this.connections.get(database);
        if (!client) {
            throw new Error(`Database ${database} not connected`);
        }

        let slowQueries = [];

        try {
            switch (database) {
                case 'postgresql':
                    // Requires pg_stat_statements extension
                    const pgResult = await client.query(`
                        SELECT 
                            query,
                            calls,
                            total_time,
                            mean_time,
                            rows
                        FROM pg_stat_statements 
                        ORDER BY mean_time DESC 
                        LIMIT $1
                    `, [limit]);
                    
                    slowQueries = pgResult.rows.map(row => ({
                        query: row.query.substring(0, 200) + '...',
                        calls: row.calls,
                        totalTime: row.total_time,
                        averageTime: row.mean_time,
                        rowsAffected: row.rows
                    }));
                    break;

                case 'mysql':
                    // Check if slow query log is enabled
                    const [slowLogRows] = await client.execute(`
                        SELECT 
                            sql_text,
                            start_time,
                            query_time,
                            rows_sent,
                            rows_examined
                        FROM mysql.slow_log 
                        ORDER BY query_time DESC 
                        LIMIT ?
                    `, [limit]);
                    
                    slowQueries = slowLogRows.map(row => ({
                        query: row.sql_text.substring(0, 200) + '...',
                        startTime: row.start_time,
                        queryTime: row.query_time,
                        rowsSent: row.rows_sent,
                        rowsExamined: row.rows_examined
                    }));
                    break;

                default:
                    this.log.warn(`Slow query analysis not implemented for ${database}`);
                    break;
            }

            return slowQueries;

        } catch (error) {
            this.log.error(`Failed to analyze slow queries for ${database}:`, error.message);
            return [];
        }
    }

    // Index Optimization
    async suggestIndexes(database, table = null) {
        const client = this.connections.get(database);
        if (!client) {
            throw new Error(`Database ${database} not connected`);
        }

        const suggestions = [];

        try {
            switch (database) {
                case 'postgresql':
                    const pgQuery = `
                        SELECT 
                            schemaname,
                            tablename,
                            seq_scan,
                            seq_tup_read,
                            idx_scan,
                            idx_tup_read
                        FROM pg_stat_user_tables
                        ${table ? 'WHERE tablename = $1' : ''}
                        ORDER BY seq_scan DESC
                    `;
                    
                    const pgResult = table 
                        ? await client.query(pgQuery, [table])
                        : await client.query(pgQuery);
                    
                    for (const row of pgResult.rows) {
                        const ratio = row.seq_scan > 0 ? row.idx_scan / row.seq_scan : 0;
                        if (ratio < 0.1 && row.seq_scan > 100) {
                            suggestions.push({
                                table: `${row.schemaname}.${row.tablename}`,
                                issue: 'High sequential scans, low index usage',
                                suggestion: 'Consider adding indexes on frequently queried columns',
                                seqScans: row.seq_scan,
                                indexScans: row.idx_scan,
                                priority: 'high'
                            });
                        }
                    }
                    break;

                case 'mysql':
                    const mysqlQuery = `
                        SELECT 
                            table_schema,
                            table_name,
                            index_name,
                            cardinality,
                            non_unique
                        FROM information_schema.statistics
                        WHERE table_schema = DATABASE()
                        ${table ? 'AND table_name = ?' : ''}
                        ORDER BY cardinality DESC
                    `;
                    
                    const [mysqlRows] = table
                        ? await client.execute(mysqlQuery, [table])
                        : await client.execute(mysqlQuery);
                    
                    // Simple heuristic: tables without indexes or with low cardinality indexes
                    const tableIndexes = mysqlRows.reduce((acc, row) => {
                        const key = `${row.table_schema}.${row.table_name}`;
                        if (!acc[key]) acc[key] = [];
                        acc[key].push(row);
                        return acc;
                    }, {});
                    
                    for (const [tableName, indexes] of Object.entries(tableIndexes)) {
                        if (indexes.length === 0) {
                            suggestions.push({
                                table: tableName,
                                issue: 'No indexes found',
                                suggestion: 'Consider adding primary key and indexes on foreign keys',
                                priority: 'high'
                            });
                        } else {
                            const lowCardinalityIndexes = indexes.filter(idx => idx.cardinality < 10);
                            if (lowCardinalityIndexes.length > 0) {
                                suggestions.push({
                                    table: tableName,
                                    issue: 'Low cardinality indexes detected',
                                    suggestion: 'Review index effectiveness and consider composite indexes',
                                    indexes: lowCardinalityIndexes.map(idx => idx.index_name),
                                    priority: 'medium'
                                });
                            }
                        }
                    }
                    break;

                default:
                    this.log.warn(`Index suggestions not implemented for ${database}`);
                    break;
            }

            return suggestions;

        } catch (error) {
            this.log.error(`Failed to suggest indexes for ${database}:`, error.message);
            return [];
        }
    }

    // Security Audit
    async performSecurityAudit(database) {
        const client = this.connections.get(database);
        if (!client) {
            throw new Error(`Database ${database} not connected`);
        }

        const audit = {
            database,
            timestamp: new Date(),
            findings: [],
            score: 100,
            recommendations: []
        };

        try {
            switch (database) {
                case 'postgresql':
                    await this.auditPostgreSQL(client, audit);
                    break;
                case 'mysql':
                    await this.auditMySQL(client, audit);
                    break;
                default:
                    this.log.warn(`Security audit not implemented for ${database}`);
                    break;
            }

            // Calculate overall security score
            const severityWeights = { high: 20, medium: 10, low: 5 };
            const totalDeduction = audit.findings.reduce((sum, finding) => {
                return sum + (severityWeights[finding.severity] || 0);
            }, 0);
            
            audit.score = Math.max(0, 100 - totalDeduction);

            return audit;

        } catch (error) {
            this.log.error(`Security audit failed for ${database}:`, error.message);
            audit.findings.push({
                category: 'audit_error',
                severity: 'high',
                message: `Security audit failed: ${error.message}`,
                recommendation: 'Investigate audit failure and ensure proper permissions'
            });
            return audit;
        }
    }

    async auditPostgreSQL(client, audit) {
        // Check for default passwords
        const userResult = await client.query(`
            SELECT usename, usesuper 
            FROM pg_user 
            WHERE usename IN ('postgres', 'admin', 'root')
        `);
        
        for (const user of userResult.rows) {
            if (user.usesuper) {
                audit.findings.push({
                    category: 'user_privileges',
                    severity: 'medium',
                    message: `Superuser account detected: ${user.usename}`,
                    recommendation: 'Limit superuser privileges and use role-based access'
                });
            }
        }

        // Check SSL configuration
        const sslResult = await client.query('SHOW ssl');
        if (sslResult.rows[0].ssl !== 'on') {
            audit.findings.push({
                category: 'encryption',
                severity: 'high',
                message: 'SSL is not enabled',
                recommendation: 'Enable SSL for encrypted connections'
            });
        }

        // Check for weak authentication methods
        const authResult = await client.query(`
            SELECT name, setting 
            FROM pg_settings 
            WHERE name IN ('password_encryption', 'log_connections', 'log_disconnections')
        `);
        
        for (const setting of authResult.rows) {
            if (setting.name === 'password_encryption' && setting.setting !== 'scram-sha-256') {
                audit.findings.push({
                    category: 'authentication',
                    severity: 'medium',
                    message: 'Weak password encryption method',
                    recommendation: 'Use SCRAM-SHA-256 for password encryption'
                });
            }
        }
    }

    async auditMySQL(client, audit) {
        // Check for users without passwords
        const [userRows] = await client.execute(`
            SELECT user, host, authentication_string 
            FROM mysql.user 
            WHERE authentication_string = ''
        `);
        
        if (userRows.length > 0) {
            audit.findings.push({
                category: 'authentication',
                severity: 'high',
                message: `${userRows.length} users without passwords found`,
                recommendation: 'Set strong passwords for all database users'
            });
        }

        // Check SSL configuration
        const [sslRows] = await client.execute("SHOW VARIABLES LIKE 'have_ssl'");
        if (sslRows.length === 0 || sslRows[0].Value !== 'YES') {
            audit.findings.push({
                category: 'encryption',
                severity: 'high',
                message: 'SSL is not available or enabled',
                recommendation: 'Configure and enable SSL for secure connections'
            });
        }

        // Check for default test database
        const [dbRows] = await client.execute("SHOW DATABASES LIKE 'test'");
        if (dbRows.length > 0) {
            audit.findings.push({
                category: 'configuration',
                severity: 'low',
                message: 'Test database exists',
                recommendation: 'Remove test database in production environments'
            });
        }
    }

    // Health Check
    async performHealthCheck() {
        const health = {
            timestamp: new Date(),
            overall: 'healthy',
            databases: {},
            metrics: this.metrics,
            alerts: []
        };

        for (const [dbType, client] of this.connections.entries()) {
            try {
                const dbHealth = await this.checkDatabaseHealth(dbType, client);
                health.databases[dbType] = dbHealth;
                
                if (dbHealth.status !== 'healthy') {
                    health.overall = 'degraded';
                    health.alerts.push(`${dbType}: ${dbHealth.message}`);
                }
            } catch (error) {
                health.databases[dbType] = {
                    status: 'unhealthy',
                    message: error.message,
                    lastCheck: new Date()
                };
                health.overall = 'unhealthy';
                health.alerts.push(`${dbType}: ${error.message}`);
            }
        }

        return health;
    }

    async checkDatabaseHealth(dbType, client) {
        const health = {
            status: 'healthy',
            responseTime: 0,
            message: 'OK',
            lastCheck: new Date()
        };

        const startTime = Date.now();

        try {
            switch (dbType) {
                case 'postgresql':
                    await client.query('SELECT 1');
                    break;
                case 'mysql':
                    await client.execute('SELECT 1');
                    break;
                case 'mongodb':
                    await client.db().admin().ping();
                    break;
                case 'redis':
                    await client.ping();
                    break;
                case 'sqlite':
                    await client.getAsync('SELECT 1');
                    break;
            }

            health.responseTime = Date.now() - startTime;

            if (health.responseTime > 1000) {
                health.status = 'degraded';
                health.message = `High response time: ${health.responseTime}ms`;
            }

        } catch (error) {
            health.status = 'unhealthy';
            health.message = error.message;
            health.responseTime = Date.now() - startTime;
        }

        return health;
    }

    // Cleanup and shutdown
    async shutdown() {
        this.log.info('Shutting down database manager...');
        
        for (const [dbType, client] of this.connections.entries()) {
            try {
                switch (dbType) {
                    case 'postgresql':
                    case 'mysql':
                        await client.end();
                        break;
                    case 'mongodb':
                        await client.close();
                        break;
                    case 'redis':
                        await client.disconnect();
                        break;
                    case 'sqlite':
                        client.close();
                        break;
                }
                this.log.info(`${dbType} connection closed`);
            } catch (error) {
                this.log.error(`Error closing ${dbType} connection:`, error.message);
            }
        }

        this.connections.clear();
        this.log.info('Database manager shutdown complete');
    }
}

// CLI Interface
async function main() {
    const args = process.argv.slice(2);
    const command = args[0];
    
    const dbManager = new DatabaseManager();

    try {
        switch (command) {
            case 'migrate':
                const database = args[1] || 'postgresql';
                const migrationsPath = args[2] || './migrations';
                await dbManager.runMigrations(database, migrationsPath);
                break;

            case 'backup':
                const databases = args[1] ? args[1].split(',') : ['all'];
                const backup = await dbManager.createBackup(databases);
                console.log('Backup created:', backup.id);
                break;

            case 'restore':
                const backupId = args[1];
                if (!backupId) {
                    console.error('Backup ID required');
                    process.exit(1);
                }
                const restoreDatabases = args[2] ? args[2].split(',') : ['all'];
                await dbManager.restoreBackup(backupId, restoreDatabases);
                break;

            case 'metrics':
                const targetDb = args[1] || 'postgresql';
                const metrics = await dbManager.getPerformanceMetrics(targetDb);
                console.log(JSON.stringify(metrics, null, 2));
                break;

            case 'slow-queries':
                const queryDb = args[1] || 'postgresql';
                const limit = parseInt(args[2]) || 10;
                const slowQueries = await dbManager.analyzeSlowQueries(queryDb, limit);
                console.log(JSON.stringify(slowQueries, null, 2));
                break;

            case 'suggest-indexes':
                const indexDb = args[1] || 'postgresql';
                const table = args[2] || null;
                const suggestions = await dbManager.suggestIndexes(indexDb, table);
                console.log(JSON.stringify(suggestions, null, 2));
                break;

            case 'security-audit':
                const auditDb = args[1] || 'postgresql';
                const audit = await dbManager.performSecurityAudit(auditDb);
                console.log(JSON.stringify(audit, null, 2));
                break;

            case 'health-check':
                const health = await dbManager.performHealthCheck();
                console.log(JSON.stringify(health, null, 2));
                break;

            default:
                console.log(`
Database Manager Commands:
  migrate <database> [migrations_path]     - Run database migrations
  backup [databases]                       - Create database backup
  restore <backup_id> [databases]          - Restore from backup
  metrics <database>                       - Get performance metrics
  slow-queries <database> [limit]          - Analyze slow queries
  suggest-indexes <database> [table]       - Get index suggestions
  security-audit <database>                - Perform security audit
  health-check                             - Check database health

Examples:
  node database-manager.js migrate postgresql ./migrations
  node database-manager.js backup postgresql,redis
  node database-manager.js metrics postgresql
  node database-manager.js security-audit mysql
                `);
                break;
        }
    } catch (error) {
        console.error('Command failed:', error.message);
        process.exit(1);
    } finally {
        await dbManager.shutdown();
    }
}

if (require.main === module) {
    main().catch(console.error);
}

module.exports = DatabaseManager;