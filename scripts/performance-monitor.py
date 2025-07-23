#!/usr/bin/env python3
"""
Advanced Performance Monitoring System
Monitors system resources, application metrics, and generates intelligent alerts
Author: Auto-generated Advanced Logic
Version: 1.0.0
"""

import asyncio
import json
import logging
import os
import psutil
import subprocess
import sys
import time
import threading
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, List, Optional, Any, Callable
import statistics
import re
import socket
import smtplib
from email.mime.text import MimeText
from email.mime.multipart import MimeMultipart

# Third-party imports (install with: pip install requests prometheus_client influxdb-client)
try:
    import requests
    from prometheus_client import CollectorRegistry, Gauge, Counter, Histogram, push_to_gateway
    from influxdb_client import InfluxDBClient, Point
    from influxdb_client.client.write_api import SYNCHRONOUS
except ImportError as e:
    print(f"Warning: Some optional dependencies are missing: {e}")
    print("Install with: pip install requests prometheus_client influxdb-client")

# Configuration
CONFIG_FILE = os.path.join(os.path.dirname(__file__), 'monitor.config.json')
LOG_FILE = '/var/log/performance-monitor.log'
METRICS_STORAGE_PATH = '/tmp/metrics_storage'
ALERT_HISTORY_FILE = '/tmp/alert_history.json'

# Default configuration
DEFAULT_CONFIG = {
    "monitoring": {
        "interval_seconds": 10,
        "cpu_threshold": 80.0,
        "memory_threshold": 85.0,
        "disk_threshold": 90.0,
        "network_threshold_mbps": 100.0,
        "response_time_threshold_ms": 1000,
        "enable_predictive_alerts": True,
        "historical_data_points": 100
    },
    "targets": {
        "grpc_server": {
            "host": "localhost",
            "port": 50051,
            "health_endpoint": "/health"
        },
        "android_app": {
            "package_name": "com.example.notebook",
            "test_endpoints": ["http://localhost:8080/api/health"]
        }
    },
    "notifications": {
        "slack_webhook": "",
        "email": {
            "enabled": False,
            "smtp_server": "smtp.gmail.com",
            "smtp_port": 587,
            "username": "",
            "password": "",
            "recipients": []
        },
        "prometheus": {
            "enabled": False,
            "gateway_url": "http://localhost:9091",
            "job_name": "performance_monitor"
        },
        "influxdb": {
            "enabled": False,
            "url": "http://localhost:8086",
            "token": "",
            "org": "my-org",
            "bucket": "monitoring"
        }
    },
    "advanced": {
        "anomaly_detection": True,
        "machine_learning": False,
        "auto_scaling": False,
        "log_analysis": True,
        "network_monitoring": True
    }
}

@dataclass
class SystemMetrics:
    timestamp: datetime
    cpu_percent: float
    memory_percent: float
    memory_available_mb: float
    disk_usage_percent: float
    disk_io_read_mb: float
    disk_io_write_mb: float
    network_sent_mb: float
    network_recv_mb: float
    load_average: List[float]
    active_connections: int
    process_count: int

@dataclass
class ApplicationMetrics:
    timestamp: datetime
    response_time_ms: float
    error_rate: float
    throughput_rps: float
    active_users: int
    database_connections: int
    queue_size: int
    memory_usage_mb: float
    cpu_usage_percent: float

@dataclass
class Alert:
    timestamp: datetime
    severity: str  # INFO, WARNING, CRITICAL
    category: str
    message: str
    value: float
    threshold: float
    duration_minutes: int
    resolved: bool = False
    resolution_time: Optional[datetime] = None

class AnomalyDetector:
    """Statistical anomaly detection using z-score and trend analysis"""
    
    def __init__(self, window_size: int = 50, threshold: float = 2.5):
        self.window_size = window_size
        self.threshold = threshold
        self.data_history: Dict[str, List[float]] = {}
    
    def add_data_point(self, metric_name: str, value: float):
        if metric_name not in self.data_history:
            self.data_history[metric_name] = []
        
        self.data_history[metric_name].append(value)
        
        # Keep only the last N data points
        if len(self.data_history[metric_name]) > self.window_size:
            self.data_history[metric_name] = self.data_history[metric_name][-self.window_size:]
    
    def is_anomaly(self, metric_name: str, value: float) -> tuple[bool, float]:
        if metric_name not in self.data_history or len(self.data_history[metric_name]) < 10:
            return False, 0.0
        
        history = self.data_history[metric_name]
        mean = statistics.mean(history)
        std_dev = statistics.stdev(history) if len(history) > 1 else 0
        
        if std_dev == 0:
            return False, 0.0
        
        z_score = abs((value - mean) / std_dev)
        return z_score > self.threshold, z_score
    
    def detect_trend(self, metric_name: str, points: int = 10) -> str:
        if metric_name not in self.data_history or len(self.data_history[metric_name]) < points:
            return "insufficient_data"
        
        recent_data = self.data_history[metric_name][-points:]
        
        # Simple trend detection using linear regression slope
        x = list(range(len(recent_data)))
        n = len(recent_data)
        
        sum_x = sum(x)
        sum_y = sum(recent_data)
        sum_xy = sum(x[i] * recent_data[i] for i in range(n))
        sum_x2 = sum(xi * xi for xi in x)
        
        slope = (n * sum_xy - sum_x * sum_y) / (n * sum_x2 - sum_x * sum_x)
        
        if slope > 0.1:
            return "increasing"
        elif slope < -0.1:
            return "decreasing"
        else:
            return "stable"

class PerformanceMonitor:
    def __init__(self, config_path: str = CONFIG_FILE):
        self.config = self.load_config(config_path)
        self.setup_logging()
        self.anomaly_detector = AnomalyDetector()
        self.alert_history: List[Alert] = []
        self.running = False
        self.executor = ThreadPoolExecutor(max_workers=5)
        
        # Metrics storage
        self.system_metrics_history: List[SystemMetrics] = []
        self.app_metrics_history: List[ApplicationMetrics] = []
        
        # Prometheus metrics
        if self.config['notifications']['prometheus']['enabled']:
            self.setup_prometheus_metrics()
        
        # InfluxDB client
        if self.config['notifications']['influxdb']['enabled']:
            self.setup_influxdb_client()
        
        # Load alert history
        self.load_alert_history()
    
    def load_config(self, config_path: str) -> Dict:
        """Load configuration from file or create default"""
        if os.path.exists(config_path):
            try:
                with open(config_path, 'r') as f:
                    config = json.load(f)
                # Merge with defaults for missing keys
                return self.merge_configs(DEFAULT_CONFIG, config)
            except Exception as e:
                print(f"Error loading config: {e}. Using defaults.")
        
        # Create default config file
        with open(config_path, 'w') as f:
            json.dump(DEFAULT_CONFIG, f, indent=2)
        
        return DEFAULT_CONFIG.copy()
    
    def merge_configs(self, default: Dict, user: Dict) -> Dict:
        """Recursively merge user config with defaults"""
        result = default.copy()
        for key, value in user.items():
            if key in result and isinstance(result[key], dict) and isinstance(value, dict):
                result[key] = self.merge_configs(result[key], value)
            else:
                result[key] = value
        return result
    
    def setup_logging(self):
        """Setup logging configuration"""
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(LOG_FILE),
                logging.StreamHandler(sys.stdout)
            ]
        )
        self.logger = logging.getLogger(__name__)
    
    def setup_prometheus_metrics(self):
        """Setup Prometheus metrics"""
        self.registry = CollectorRegistry()
        self.prom_metrics = {
            'cpu_usage': Gauge('system_cpu_usage_percent', 'CPU usage percentage', registry=self.registry),
            'memory_usage': Gauge('system_memory_usage_percent', 'Memory usage percentage', registry=self.registry),
            'disk_usage': Gauge('system_disk_usage_percent', 'Disk usage percentage', registry=self.registry),
            'response_time': Histogram('app_response_time_ms', 'Application response time', registry=self.registry),
            'error_rate': Gauge('app_error_rate', 'Application error rate', registry=self.registry),
            'alerts_total': Counter('alerts_total', 'Total number of alerts', ['severity'], registry=self.registry)
        }
    
    def setup_influxdb_client(self):
        """Setup InfluxDB client"""
        influx_config = self.config['notifications']['influxdb']
        self.influx_client = InfluxDBClient(
            url=influx_config['url'],
            token=influx_config['token'],
            org=influx_config['org']
        )
        self.influx_write_api = self.influx_client.write_api(write_options=SYNCHRONOUS)
    
    def load_alert_history(self):
        """Load alert history from file"""
        if os.path.exists(ALERT_HISTORY_FILE):
            try:
                with open(ALERT_HISTORY_FILE, 'r') as f:
                    data = json.load(f)
                    self.alert_history = [
                        Alert(
                            timestamp=datetime.fromisoformat(alert['timestamp']),
                            severity=alert['severity'],
                            category=alert['category'],
                            message=alert['message'],
                            value=alert['value'],
                            threshold=alert['threshold'],
                            duration_minutes=alert['duration_minutes'],
                            resolved=alert.get('resolved', False),
                            resolution_time=datetime.fromisoformat(alert['resolution_time']) if alert.get('resolution_time') else None
                        )
                        for alert in data
                    ]
            except Exception as e:
                self.logger.error(f"Error loading alert history: {e}")
    
    def save_alert_history(self):
        """Save alert history to file"""
        try:
            data = []
            for alert in self.alert_history:
                alert_dict = asdict(alert)
                alert_dict['timestamp'] = alert.timestamp.isoformat()
                if alert.resolution_time:
                    alert_dict['resolution_time'] = alert.resolution_time.isoformat()
                data.append(alert_dict)
            
            with open(ALERT_HISTORY_FILE, 'w') as f:
                json.dump(data, f, indent=2)
        except Exception as e:
            self.logger.error(f"Error saving alert history: {e}")
    
    def collect_system_metrics(self) -> SystemMetrics:
        """Collect system-level metrics"""
        cpu_percent = psutil.cpu_percent(interval=1)
        memory = psutil.virtual_memory()
        disk = psutil.disk_usage('/')
        disk_io = psutil.disk_io_counters()
        network_io = psutil.net_io_counters()
        load_avg = os.getloadavg() if hasattr(os, 'getloadavg') else [0, 0, 0]
        
        # Network connections
        try:
            connections = len(psutil.net_connections())
        except (psutil.AccessDenied, OSError):
            connections = 0
        
        return SystemMetrics(
            timestamp=datetime.now(),
            cpu_percent=cpu_percent,
            memory_percent=memory.percent,
            memory_available_mb=memory.available / 1024 / 1024,
            disk_usage_percent=disk.percent,
            disk_io_read_mb=disk_io.read_bytes / 1024 / 1024 if disk_io else 0,
            disk_io_write_mb=disk_io.write_bytes / 1024 / 1024 if disk_io else 0,
            network_sent_mb=network_io.bytes_sent / 1024 / 1024 if network_io else 0,
            network_recv_mb=network_io.bytes_recv / 1024 / 1024 if network_io else 0,
            load_average=list(load_avg),
            active_connections=connections,
            process_count=len(psutil.pids())
        )
    
    def collect_application_metrics(self) -> ApplicationMetrics:
        """Collect application-specific metrics"""
        # This is a simplified version - in reality, you'd integrate with your application's metrics
        response_time = self.measure_response_time()
        error_rate = self.calculate_error_rate()
        
        return ApplicationMetrics(
            timestamp=datetime.now(),
            response_time_ms=response_time,
            error_rate=error_rate,
            throughput_rps=self.calculate_throughput(),
            active_users=self.get_active_users(),
            database_connections=self.get_database_connections(),
            queue_size=self.get_queue_size(),
            memory_usage_mb=self.get_app_memory_usage(),
            cpu_usage_percent=self.get_app_cpu_usage()
        )
    
    def measure_response_time(self) -> float:
        """Measure application response time"""
        try:
            grpc_target = self.config['targets']['grpc_server']
            start_time = time.time()
            
            # Test gRPC connection
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(5)
            result = sock.connect_ex((grpc_target['host'], grpc_target['port']))
            sock.close()
            
            if result == 0:
                response_time = (time.time() - start_time) * 1000
                return response_time
            else:
                return float('inf')  # Connection failed
        except Exception as e:
            self.logger.error(f"Error measuring response time: {e}")
            return float('inf')
    
    def calculate_error_rate(self) -> float:
        """Calculate application error rate"""
        # Simplified - analyze logs or metrics for actual error rate
        try:
            # Check if service is responding
            response_time = self.measure_response_time()
            if response_time == float('inf'):
                return 100.0  # Service is down
            elif response_time > self.config['monitoring']['response_time_threshold_ms']:
                return 25.0   # High response time indicates issues
            else:
                return 1.0    # Normal error rate
        except Exception:
            return 50.0
    
    def calculate_throughput(self) -> float:
        """Calculate requests per second"""
        # Simplified calculation
        return max(0, 100 - (self.measure_response_time() / 10))
    
    def get_active_users(self) -> int:
        """Get number of active users (simplified)"""
        return min(100, max(1, int(time.time() % 100)))
    
    def get_database_connections(self) -> int:
        """Get database connection count (simplified)"""
        return min(50, max(1, int(time.time() % 50)))
    
    def get_queue_size(self) -> int:
        """Get message queue size (simplified)"""
        return max(0, int(time.time() % 20))
    
    def get_app_memory_usage(self) -> float:
        """Get application memory usage"""
        try:
            # Find processes by name (simplified)
            total_memory = 0
            for proc in psutil.process_iter(['pid', 'name', 'memory_info']):
                if 'java' in proc.info['name'].lower() or 'notebook' in proc.info['name'].lower():
                    total_memory += proc.info['memory_info'].rss
            return total_memory / 1024 / 1024  # Convert to MB
        except Exception:
            return 0
    
    def get_app_cpu_usage(self) -> float:
        """Get application CPU usage"""
        try:
            total_cpu = 0
            count = 0
            for proc in psutil.process_iter(['pid', 'name', 'cpu_percent']):
                if 'java' in proc.info['name'].lower() or 'notebook' in proc.info['name'].lower():
                    total_cpu += proc.cpu_percent()
                    count += 1
            return total_cpu / max(1, count)
        except Exception:
            return 0
    
    def analyze_metrics(self, system_metrics: SystemMetrics, app_metrics: ApplicationMetrics):
        """Analyze metrics and generate alerts"""
        monitoring_config = self.config['monitoring']
        
        # System metrics analysis
        self.check_threshold_alert("CPU Usage", system_metrics.cpu_percent, 
                                 monitoring_config['cpu_threshold'], "system")
        self.check_threshold_alert("Memory Usage", system_metrics.memory_percent, 
                                 monitoring_config['memory_threshold'], "system")
        self.check_threshold_alert("Disk Usage", system_metrics.disk_usage_percent, 
                                 monitoring_config['disk_threshold'], "system")
        
        # Application metrics analysis
        self.check_threshold_alert("Response Time", app_metrics.response_time_ms, 
                                 monitoring_config['response_time_threshold_ms'], "application")
        self.check_threshold_alert("Error Rate", app_metrics.error_rate, 20.0, "application")
        
        # Anomaly detection
        if self.config['advanced']['anomaly_detection']:
            self.detect_anomalies(system_metrics, app_metrics)
        
        # Predictive alerts
        if monitoring_config['enable_predictive_alerts']:
            self.generate_predictive_alerts()
    
    def check_threshold_alert(self, metric_name: str, value: float, threshold: float, category: str):
        """Check if metric exceeds threshold and generate alert"""
        if value > threshold:
            severity = "CRITICAL" if value > threshold * 1.2 else "WARNING"
            
            # Check if we already have an active alert for this metric
            active_alerts = [a for a in self.alert_history 
                           if not a.resolved and a.category == category and metric_name in a.message]
            
            if not active_alerts:
                alert = Alert(
                    timestamp=datetime.now(),
                    severity=severity,
                    category=category,
                    message=f"{metric_name} is {value:.2f}% (threshold: {threshold}%)",
                    value=value,
                    threshold=threshold,
                    duration_minutes=0
                )
                self.alert_history.append(alert)
                self.send_alert(alert)
        else:
            # Resolve any active alerts for this metric
            for alert in self.alert_history:
                if not alert.resolved and alert.category == category and metric_name in alert.message:
                    alert.resolved = True
                    alert.resolution_time = datetime.now()
                    self.logger.info(f"Alert resolved: {alert.message}")
    
    def detect_anomalies(self, system_metrics: SystemMetrics, app_metrics: ApplicationMetrics):
        """Detect anomalies using statistical methods"""
        metrics_to_check = [
            ("cpu_percent", system_metrics.cpu_percent, "system"),
            ("memory_percent", system_metrics.memory_percent, "system"),
            ("response_time_ms", app_metrics.response_time_ms, "application"),
            ("error_rate", app_metrics.error_rate, "application")
        ]
        
        for metric_name, value, category in metrics_to_check:
            self.anomaly_detector.add_data_point(metric_name, value)
            is_anomaly, z_score = self.anomaly_detector.is_anomaly(metric_name, value)
            
            if is_anomaly:
                trend = self.anomaly_detector.detect_trend(metric_name)
                alert = Alert(
                    timestamp=datetime.now(),
                    severity="WARNING",
                    category=category,
                    message=f"Anomaly detected in {metric_name}: {value:.2f} (z-score: {z_score:.2f}, trend: {trend})",
                    value=value,
                    threshold=z_score,
                    duration_minutes=0
                )
                self.alert_history.append(alert)
                self.send_alert(alert)
    
    def generate_predictive_alerts(self):
        """Generate predictive alerts based on trends"""
        if len(self.system_metrics_history) < 10:
            return
        
        # Predict CPU usage trend
        recent_cpu = [m.cpu_percent for m in self.system_metrics_history[-10:]]
        cpu_trend = self.anomaly_detector.detect_trend("cpu_recent", 5)
        
        if cpu_trend == "increasing":
            avg_cpu = statistics.mean(recent_cpu)
            if avg_cpu > 60:  # Predictive threshold
                alert = Alert(
                    timestamp=datetime.now(),
                    severity="INFO",
                    category="predictive",
                    message=f"CPU usage trending upward: {avg_cpu:.2f}% (trend: {cpu_trend})",
                    value=avg_cpu,
                    threshold=60.0,
                    duration_minutes=0
                )
                self.alert_history.append(alert)
                self.send_alert(alert)
    
    def send_alert(self, alert: Alert):
        """Send alert through configured channels"""
        self.logger.warning(f"ALERT [{alert.severity}] {alert.category}: {alert.message}")
        
        # Slack notification
        if self.config['notifications']['slack_webhook']:
            self.send_slack_alert(alert)
        
        # Email notification
        if self.config['notifications']['email']['enabled']:
            self.send_email_alert(alert)
        
        # Prometheus metrics
        if self.config['notifications']['prometheus']['enabled']:
            self.prom_metrics['alerts_total'].labels(severity=alert.severity.lower()).inc()
    
    def send_slack_alert(self, alert: Alert):
        """Send alert to Slack"""
        try:
            color_map = {"INFO": "#36a64f", "WARNING": "#ff9800", "CRITICAL": "#f44336"}
            emoji_map = {"INFO": "ℹ️", "WARNING": "⚠️", "CRITICAL": "🚨"}
            
            payload = {
                "text": f"{emoji_map.get(alert.severity, '🔔')} Performance Alert",
                "attachments": [{
                    "color": color_map.get(alert.severity, "#cccccc"),
                    "fields": [
                        {"title": "Severity", "value": alert.severity, "short": True},
                        {"title": "Category", "value": alert.category, "short": True},
                        {"title": "Message", "value": alert.message, "short": False},
                        {"title": "Value", "value": f"{alert.value:.2f}", "short": True},
                        {"title": "Threshold", "value": f"{alert.threshold:.2f}", "short": True},
                        {"title": "Time", "value": alert.timestamp.strftime("%Y-%m-%d %H:%M:%S"), "short": False}
                    ]
                }]
            }
            
            response = requests.post(self.config['notifications']['slack_webhook'], 
                                   json=payload, timeout=10)
            response.raise_for_status()
        except Exception as e:
            self.logger.error(f"Failed to send Slack alert: {e}")
    
    def send_email_alert(self, alert: Alert):
        """Send alert via email"""
        try:
            email_config = self.config['notifications']['email']
            
            msg = MimeMultipart()
            msg['From'] = email_config['username']
            msg['To'] = ', '.join(email_config['recipients'])
            msg['Subject'] = f"[{alert.severity}] Performance Alert - {alert.category}"
            
            body = f"""
Performance Alert Details:

Severity: {alert.severity}
Category: {alert.category}
Message: {alert.message}
Value: {alert.value:.2f}
Threshold: {alert.threshold:.2f}
Time: {alert.timestamp.strftime("%Y-%m-%d %H:%M:%S")}

This is an automated alert from the Performance Monitoring System.
            """
            
            msg.attach(MimeText(body, 'plain'))
            
            server = smtplib.SMTP(email_config['smtp_server'], email_config['smtp_port'])
            server.starttls()
            server.login(email_config['username'], email_config['password'])
            server.send_message(msg)
            server.quit()
            
        except Exception as e:
            self.logger.error(f"Failed to send email alert: {e}")
    
    def export_metrics_to_influxdb(self, system_metrics: SystemMetrics, app_metrics: ApplicationMetrics):
        """Export metrics to InfluxDB"""
        try:
            points = []
            
            # System metrics
            point = Point("system_metrics") \
                .field("cpu_percent", system_metrics.cpu_percent) \
                .field("memory_percent", system_metrics.memory_percent) \
                .field("disk_usage_percent", system_metrics.disk_usage_percent) \
                .field("network_sent_mb", system_metrics.network_sent_mb) \
                .field("network_recv_mb", system_metrics.network_recv_mb) \
                .time(system_metrics.timestamp)
            points.append(point)
            
            # Application metrics
            point = Point("app_metrics") \
                .field("response_time_ms", app_metrics.response_time_ms) \
                .field("error_rate", app_metrics.error_rate) \
                .field("throughput_rps", app_metrics.throughput_rps) \
                .field("active_users", app_metrics.active_users) \
                .time(app_metrics.timestamp)
            points.append(point)
            
            self.influx_write_api.write(
                bucket=self.config['notifications']['influxdb']['bucket'],
                record=points
            )
            
        except Exception as e:
            self.logger.error(f"Failed to export metrics to InfluxDB: {e}")
    
    def export_metrics_to_prometheus(self, system_metrics: SystemMetrics, app_metrics: ApplicationMetrics):
        """Export metrics to Prometheus"""
        try:
            self.prom_metrics['cpu_usage'].set(system_metrics.cpu_percent)
            self.prom_metrics['memory_usage'].set(system_metrics.memory_percent)
            self.prom_metrics['disk_usage'].set(system_metrics.disk_usage_percent)
            self.prom_metrics['response_time'].observe(app_metrics.response_time_ms)
            self.prom_metrics['error_rate'].set(app_metrics.error_rate)
            
            push_to_gateway(
                self.config['notifications']['prometheus']['gateway_url'],
                job=self.config['notifications']['prometheus']['job_name'],
                registry=self.registry
            )
            
        except Exception as e:
            self.logger.error(f"Failed to export metrics to Prometheus: {e}")
    
    def generate_performance_report(self) -> Dict:
        """Generate comprehensive performance report"""
        now = datetime.now()
        
        # Calculate averages and statistics
        if self.system_metrics_history:
            avg_cpu = statistics.mean([m.cpu_percent for m in self.system_metrics_history[-60:]])  # Last hour
            avg_memory = statistics.mean([m.memory_percent for m in self.system_metrics_history[-60:]])
            max_cpu = max([m.cpu_percent for m in self.system_metrics_history[-60:]])
            max_memory = max([m.memory_percent for m in self.system_metrics_history[-60:]])
        else:
            avg_cpu = avg_memory = max_cpu = max_memory = 0
        
        if self.app_metrics_history:
            avg_response_time = statistics.mean([m.response_time_ms for m in self.app_metrics_history[-60:]])
            avg_error_rate = statistics.mean([m.error_rate for m in self.app_metrics_history[-60:]])
        else:
            avg_response_time = avg_error_rate = 0
        
        # Alert statistics
        recent_alerts = [a for a in self.alert_history if a.timestamp > now - timedelta(hours=24)]
        critical_alerts = [a for a in recent_alerts if a.severity == "CRITICAL"]
        warning_alerts = [a for a in recent_alerts if a.severity == "WARNING"]
        
        report = {
            "generated_at": now.isoformat(),
            "monitoring_period_hours": 24,
            "system_performance": {
                "cpu": {
                    "average_percent": round(avg_cpu, 2),
                    "peak_percent": round(max_cpu, 2),
                    "status": "good" if avg_cpu < 70 else "warning" if avg_cpu < 85 else "critical"
                },
                "memory": {
                    "average_percent": round(avg_memory, 2),
                    "peak_percent": round(max_memory, 2),
                    "status": "good" if avg_memory < 80 else "warning" if avg_memory < 90 else "critical"
                }
            },
            "application_performance": {
                "response_time": {
                    "average_ms": round(avg_response_time, 2),
                    "status": "good" if avg_response_time < 500 else "warning" if avg_response_time < 1000 else "critical"
                },
                "error_rate": {
                    "average_percent": round(avg_error_rate, 2),
                    "status": "good" if avg_error_rate < 5 else "warning" if avg_error_rate < 15 else "critical"
                }
            },
            "alerts": {
                "total_24h": len(recent_alerts),
                "critical": len(critical_alerts),
                "warnings": len(warning_alerts),
                "resolved": len([a for a in recent_alerts if a.resolved])
            },
            "recommendations": self.generate_recommendations()
        }
        
        return report
    
    def generate_recommendations(self) -> List[str]:
        """Generate performance recommendations based on metrics"""
        recommendations = []
        
        if self.system_metrics_history:
            recent_cpu = [m.cpu_percent for m in self.system_metrics_history[-30:]]
            recent_memory = [m.memory_percent for m in self.system_metrics_history[-30:]]
            
            avg_cpu = statistics.mean(recent_cpu)
            avg_memory = statistics.mean(recent_memory)
            
            if avg_cpu > 80:
                recommendations.append("Consider scaling up CPU resources or optimizing CPU-intensive processes")
            
            if avg_memory > 85:
                recommendations.append("Memory usage is high - consider increasing memory or optimizing memory usage")
            
            if len([c for c in recent_cpu if c > 90]) > 5:
                recommendations.append("Frequent CPU spikes detected - investigate background processes")
        
        if self.app_metrics_history:
            recent_response_times = [m.response_time_ms for m in self.app_metrics_history[-30:]]
            recent_error_rates = [m.error_rate for m in self.app_metrics_history[-30:]]
            
            if statistics.mean(recent_response_times) > 1000:
                recommendations.append("High response times detected - optimize database queries and API endpoints")
            
            if statistics.mean(recent_error_rates) > 10:
                recommendations.append("High error rate detected - review application logs and fix critical issues")
        
        # Alert-based recommendations
        recent_alerts = [a for a in self.alert_history 
                        if a.timestamp > datetime.now() - timedelta(hours=24)]
        
        if len(recent_alerts) > 20:
            recommendations.append("High alert frequency - review monitoring thresholds and system stability")
        
        if not recommendations:
            recommendations.append("System performance is within normal parameters")
        
        return recommendations
    
    async def monitoring_loop(self):
        """Main monitoring loop"""
        self.logger.info("Starting performance monitoring...")
        self.running = True
        
        interval = self.config['monitoring']['interval_seconds']
        historical_limit = self.config['monitoring']['historical_data_points']
        
        while self.running:
            try:
                # Collect metrics
                system_metrics = self.collect_system_metrics()
                app_metrics = self.collect_application_metrics()
                
                # Store in history
                self.system_metrics_history.append(system_metrics)
                self.app_metrics_history.append(app_metrics)
                
                # Maintain history size
                if len(self.system_metrics_history) > historical_limit:
                    self.system_metrics_history = self.system_metrics_history[-historical_limit:]
                if len(self.app_metrics_history) > historical_limit:
                    self.app_metrics_history = self.app_metrics_history[-historical_limit:]
                
                # Analyze metrics and generate alerts
                self.analyze_metrics(system_metrics, app_metrics)
                
                # Export to external systems
                if self.config['notifications']['prometheus']['enabled']:
                    self.export_metrics_to_prometheus(system_metrics, app_metrics)
                
                if self.config['notifications']['influxdb']['enabled']:
                    self.export_metrics_to_influxdb(system_metrics, app_metrics)
                
                # Save alert history periodically
                if len(self.alert_history) % 10 == 0:
                    self.save_alert_history()
                
                # Log current status
                self.logger.info(f"Metrics collected - CPU: {system_metrics.cpu_percent:.1f}%, "
                               f"Memory: {system_metrics.memory_percent:.1f}%, "
                               f"Response Time: {app_metrics.response_time_ms:.1f}ms")
                
                await asyncio.sleep(interval)
                
            except Exception as e:
                self.logger.error(f"Error in monitoring loop: {e}")
                await asyncio.sleep(interval)
    
    def start_monitoring(self):
        """Start the monitoring system"""
        try:
            asyncio.run(self.monitoring_loop())
        except KeyboardInterrupt:
            self.logger.info("Monitoring stopped by user")
        finally:
            self.stop_monitoring()
    
    def stop_monitoring(self):
        """Stop the monitoring system"""
        self.running = False
        self.save_alert_history()
        
        # Generate final report
        report = self.generate_performance_report()
        report_file = f"/tmp/performance_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        
        with open(report_file, 'w') as f:
            json.dump(report, f, indent=2)
        
        self.logger.info(f"Final performance report saved to: {report_file}")
        self.logger.info("Performance monitoring stopped")

def main():
    """Main function"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Advanced Performance Monitoring System")
    parser.add_argument("--config", default=CONFIG_FILE, help="Configuration file path")
    parser.add_argument("--report", action="store_true", help="Generate performance report and exit")
    parser.add_argument("--test-alert", action="store_true", help="Send test alert and exit")
    
    args = parser.parse_args()
    
    monitor = PerformanceMonitor(args.config)
    
    if args.report:
        report = monitor.generate_performance_report()
        print(json.dumps(report, indent=2))
        return
    
    if args.test_alert:
        test_alert = Alert(
            timestamp=datetime.now(),
            severity="INFO",
            category="test",
            message="This is a test alert from the monitoring system",
            value=42.0,
            threshold=40.0,
            duration_minutes=0
        )
        monitor.send_alert(test_alert)
        print("Test alert sent")
        return
    
    # Start monitoring
    monitor.start_monitoring()

if __name__ == "__main__":
    main()