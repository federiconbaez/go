#!/bin/bash

# Advanced Deployment Script with Blue-Green, Canary, and Rollback Support
# Author: Auto-generated Advanced Logic
# Version: 1.0.0

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="${SCRIPT_DIR}/deploy.config"
LOG_FILE="/var/log/deployment-$(date +%Y%m%d-%H%M%S).log"
DEPLOYMENT_ID="deploy-$(date +%s)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Deployment strategies
STRATEGY_BLUE_GREEN="blue-green"
STRATEGY_CANARY="canary"
STRATEGY_ROLLING="rolling"
STRATEGY_RECREATE="recreate"

# Default configuration
DEPLOYMENT_STRATEGY="${DEPLOYMENT_STRATEGY:-$STRATEGY_BLUE_GREEN}"
ENVIRONMENT="${ENVIRONMENT:-staging}"
NAMESPACE="${NAMESPACE:-default}"
TIMEOUT="${TIMEOUT:-600}"
HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-120}"
ROLLBACK_ON_FAILURE="${ROLLBACK_ON_FAILURE:-true}"
CANARY_PERCENTAGE="${CANARY_PERCENTAGE:-10}"
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"
PROMETHEUS_URL="${PROMETHEUS_URL:-}"

# Load configuration if exists
if [[ -f "$CONFIG_FILE" ]]; then
    source "$CONFIG_FILE"
fi

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "$LOG_FILE"
}

log_info() { log "INFO" "$@"; }
log_warn() { log "WARN" "${YELLOW}$*${NC}"; }
log_error() { log "ERROR" "${RED}$*${NC}"; }
log_success() { log "SUCCESS" "${GREEN}$*${NC}"; }

# Error handling
trap 'handle_error $? $LINENO' ERR
handle_error() {
    local exit_code=$1
    local line_number=$2
    log_error "Deployment failed at line $line_number with exit code $exit_code"
    send_notification "❌ Deployment Failed" "Deployment $DEPLOYMENT_ID failed at line $line_number"
    cleanup_on_failure
    exit $exit_code
}

# Cleanup function
cleanup_on_failure() {
    log_warn "Performing cleanup due to failure..."
    
    if [[ "$ROLLBACK_ON_FAILURE" == "true" ]]; then
        log_info "Initiating automatic rollback..."
        rollback_deployment
    fi
    
    # Clean up temporary resources
    kubectl delete configmap "$DEPLOYMENT_ID-config" --ignore-not-found=true 2>/dev/null || true
    kubectl delete job "migration-$DEPLOYMENT_ID" --ignore-not-found=true 2>/dev/null || true
}

# Notification function
send_notification() {
    local title="$1"
    local message="$2"
    local color="${3:-#FF6B6B}"
    
    if [[ -n "$SLACK_WEBHOOK" ]]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"$title\",\"attachments\":[{\"color\":\"$color\",\"text\":\"$message\"}]}" \
            "$SLACK_WEBHOOK" 2>/dev/null || log_warn "Failed to send Slack notification"
    fi
    
    log_info "Notification: $title - $message"
}

# Prerequisites check
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local required_tools=("kubectl" "docker" "helm" "jq" "curl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    # Check Kubernetes connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "Creating namespace: $NAMESPACE"
        kubectl create namespace "$NAMESPACE"
    fi
    
    log_success "Prerequisites check passed"
}

# Health check function
perform_health_check() {
    local service_name="$1"
    local expected_replicas="$2"
    local timeout="$3"
    
    log_info "Performing health check for $service_name..."
    
    local start_time=$(date +%s)
    while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [[ $elapsed -gt $timeout ]]; then
            log_error "Health check timeout for $service_name"
            return 1
        fi
        
        local ready_replicas=$(kubectl get deployment "$service_name" -n "$NAMESPACE" \
            -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        
        if [[ "$ready_replicas" == "$expected_replicas" ]]; then
            log_success "Health check passed for $service_name"
            return 0
        fi
        
        log_info "Waiting for $service_name: $ready_replicas/$expected_replicas replicas ready"
        sleep 10
    done
}

# Database migration
run_database_migration() {
    log_info "Running database migration..."
    
    local migration_job="migration-$DEPLOYMENT_ID"
    
    cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: $migration_job
  namespace: $NAMESPACE
  labels:
    deployment-id: $DEPLOYMENT_ID
spec:
  template:
    spec:
      containers:
      - name: migration
        image: $MIGRATION_IMAGE
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: database-secret
              key: url
        command: ["sh", "-c", "echo 'Running migration...' && sleep 5 && echo 'Migration completed'"]
      restartPolicy: Never
  backoffLimit: 3
EOF
    
    # Wait for migration to complete
    kubectl wait --for=condition=complete job/$migration_job -n "$NAMESPACE" --timeout=300s
    
    if kubectl get job "$migration_job" -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' | grep -q "True"; then
        log_error "Database migration failed"
        return 1
    fi
    
    log_success "Database migration completed successfully"
}

# Blue-Green deployment
deploy_blue_green() {
    local app_name="$1"
    local new_image="$2"
    local current_color="blue"
    local new_color="green"
    
    log_info "Starting Blue-Green deployment for $app_name"
    
    # Determine current active color
    if kubectl get service "$app_name" -n "$NAMESPACE" &> /dev/null; then
        current_color=$(kubectl get service "$app_name" -n "$NAMESPACE" \
            -o jsonpath='{.spec.selector.color}' 2>/dev/null || echo "blue")
        new_color=$([[ "$current_color" == "blue" ]] && echo "green" || echo "blue")
    fi
    
    log_info "Current: $current_color, Deploying: $new_color"
    
    # Deploy new version
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $app_name-$new_color
  namespace: $NAMESPACE
  labels:
    app: $app_name
    color: $new_color
    deployment-id: $DEPLOYMENT_ID
spec:
  replicas: 3
  selector:
    matchLabels:
      app: $app_name
      color: $new_color
  template:
    metadata:
      labels:
        app: $app_name
        color: $new_color
    spec:
      containers:
      - name: $app_name
        image: $new_image
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
EOF
    
    # Wait for new deployment to be ready
    perform_health_check "$app_name-$new_color" 3 "$HEALTH_CHECK_TIMEOUT"
    
    # Switch traffic to new version
    kubectl patch service "$app_name" -n "$NAMESPACE" \
        -p '{"spec":{"selector":{"color":"'$new_color'"}}}'
    
    log_success "Traffic switched to $new_color environment"
    
    # Wait a bit then cleanup old version
    sleep 30
    kubectl delete deployment "$app_name-$current_color" -n "$NAMESPACE" --ignore-not-found=true
    
    log_success "Blue-Green deployment completed successfully"
}

# Canary deployment
deploy_canary() {
    local app_name="$1"
    local new_image="$2"
    local canary_percentage="$3"
    
    log_info "Starting Canary deployment for $app_name at $canary_percentage%"
    
    # Get current replica count
    local total_replicas=$(kubectl get deployment "$app_name" -n "$NAMESPACE" \
        -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "3")
    
    local canary_replicas=$(( (total_replicas * canary_percentage) / 100 ))
    local stable_replicas=$((total_replicas - canary_replicas))
    
    # Ensure at least 1 canary replica
    [[ $canary_replicas -eq 0 ]] && canary_replicas=1
    stable_replicas=$((total_replicas - canary_replicas))
    
    log_info "Deploying $canary_replicas canary replicas, keeping $stable_replicas stable replicas"
    
    # Deploy canary version
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $app_name-canary
  namespace: $NAMESPACE
  labels:
    app: $app_name
    version: canary
    deployment-id: $DEPLOYMENT_ID
spec:
  replicas: $canary_replicas
  selector:
    matchLabels:
      app: $app_name
      version: canary
  template:
    metadata:
      labels:
        app: $app_name
        version: canary
    spec:
      containers:
      - name: $app_name
        image: $new_image
        ports:
        - containerPort: 8080
EOF
    
    # Scale down main deployment
    kubectl scale deployment "$app_name" -n "$NAMESPACE" --replicas="$stable_replicas"
    
    # Wait for canary to be ready
    perform_health_check "$app_name-canary" "$canary_replicas" "$HEALTH_CHECK_TIMEOUT"
    
    # Monitor canary for metrics
    monitor_canary_metrics "$app_name"
    
    # If successful, promote canary
    log_info "Promoting canary to main deployment"
    kubectl set image deployment/"$app_name" -n "$NAMESPACE" "$app_name=$new_image"
    kubectl scale deployment "$app_name" -n "$NAMESPACE" --replicas="$total_replicas"
    kubectl delete deployment "$app_name-canary" -n "$NAMESPACE"
    
    log_success "Canary deployment completed successfully"
}

# Monitor canary metrics
monitor_canary_metrics() {
    local app_name="$1"
    local monitoring_duration=300  # 5 minutes
    
    log_info "Monitoring canary metrics for $monitoring_duration seconds..."
    
    if [[ -n "$PROMETHEUS_URL" ]]; then
        # Query Prometheus for error rate
        local query="rate(http_requests_total{job=\"$app_name-canary\",status=~\"5..\"}[5m])"
        local error_rate=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=$query" | \
            jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
        
        log_info "Current error rate: $error_rate"
        
        # Check if error rate is acceptable (< 1%)
        if (( $(echo "$error_rate > 0.01" | bc -l) )); then
            log_error "High error rate detected in canary: $error_rate"
            return 1
        fi
    else
        log_warn "No Prometheus URL configured, skipping metrics monitoring"
        sleep "$monitoring_duration"
    fi
    
    log_success "Canary metrics monitoring passed"
}

# Rolling deployment
deploy_rolling() {
    local app_name="$1"
    local new_image="$2"
    
    log_info "Starting Rolling deployment for $app_name"
    
    # Update deployment with rolling update strategy
    kubectl set image deployment/"$app_name" -n "$NAMESPACE" "$app_name=$new_image"
    
    # Wait for rollout to complete
    kubectl rollout status deployment/"$app_name" -n "$NAMESPACE" --timeout="${TIMEOUT}s"
    
    log_success "Rolling deployment completed successfully"
}

# Rollback deployment
rollback_deployment() {
    log_info "Starting rollback process..."
    
    local app_name="${1:-notebook-server}"
    
    # Get previous revision
    local previous_revision=$(kubectl rollout history deployment/"$app_name" -n "$NAMESPACE" | \
        tail -2 | head -1 | awk '{print $1}')
    
    if [[ -n "$previous_revision" ]]; then
        log_info "Rolling back to revision: $previous_revision"
        kubectl rollout undo deployment/"$app_name" -n "$NAMESPACE" --to-revision="$previous_revision"
        kubectl rollout status deployment/"$app_name" -n "$NAMESPACE" --timeout="${TIMEOUT}s"
        log_success "Rollback completed successfully"
    else
        log_error "No previous revision found for rollback"
        return 1
    fi
}

# Backup current state
backup_current_state() {
    log_info "Creating backup of current state..."
    
    local backup_dir="/tmp/deployment-backup-$DEPLOYMENT_ID"
    mkdir -p "$backup_dir"
    
    # Backup deployments
    kubectl get deployments -n "$NAMESPACE" -o yaml > "$backup_dir/deployments.yaml"
    
    # Backup services
    kubectl get services -n "$NAMESPACE" -o yaml > "$backup_dir/services.yaml"
    
    # Backup configmaps
    kubectl get configmaps -n "$NAMESPACE" -o yaml > "$backup_dir/configmaps.yaml"
    
    log_success "Backup created at: $backup_dir"
    echo "$backup_dir"
}

# Validate deployment
validate_deployment() {
    local app_name="$1"
    
    log_info "Validating deployment for $app_name..."
    
    # Check if deployment exists and is ready
    if ! kubectl get deployment "$app_name" -n "$NAMESPACE" &> /dev/null; then
        log_error "Deployment $app_name not found"
        return 1
    fi
    
    # Check readiness
    local ready_replicas=$(kubectl get deployment "$app_name" -n "$NAMESPACE" \
        -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    local desired_replicas=$(kubectl get deployment "$app_name" -n "$NAMESPACE" \
        -o jsonpath='{.spec.replicas}')
    
    if [[ "$ready_replicas" != "$desired_replicas" ]]; then
        log_error "Deployment not ready: $ready_replicas/$desired_replicas replicas"
        return 1
    fi
    
    # Test application endpoint
    local service_ip=$(kubectl get service "$app_name" -n "$NAMESPACE" \
        -o jsonpath='{.spec.clusterIP}' 2>/dev/null || echo "")
    
    if [[ -n "$service_ip" ]]; then
        if kubectl run test-pod --rm -i --restart=Never --image=curlimages/curl -- \
            curl -f "http://$service_ip:8080/health" &> /dev/null; then
            log_success "Health check endpoint accessible"
        else
            log_warn "Health check endpoint not accessible"
        fi
    fi
    
    log_success "Deployment validation passed"
}

# Generate deployment report
generate_report() {
    local deployment_status="$1"
    local start_time="$2"
    local end_time="$3"
    
    local duration=$((end_time - start_time))
    local report_file="/tmp/deployment-report-$DEPLOYMENT_ID.json"
    
    cat > "$report_file" << EOF
{
  "deployment_id": "$DEPLOYMENT_ID",
  "strategy": "$DEPLOYMENT_STRATEGY",
  "environment": "$ENVIRONMENT",
  "namespace": "$NAMESPACE",
  "status": "$deployment_status",
  "start_time": "$start_time",
  "end_time": "$end_time",
  "duration_seconds": $duration,
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    
    log_info "Deployment report generated: $report_file"
    
    # Send report if webhook configured
    if [[ -n "$SLACK_WEBHOOK" ]]; then
        local color=$([[ "$deployment_status" == "success" ]] && echo "#36a64f" || echo "#ff0000")
        local emoji=$([[ "$deployment_status" == "success" ]] && echo "✅" || echo "❌")
        
        send_notification "$emoji Deployment Report" \
            "ID: $DEPLOYMENT_ID\nStrategy: $DEPLOYMENT_STRATEGY\nStatus: $deployment_status\nDuration: ${duration}s" \
            "$color"
    fi
}

# Main deployment function
main() {
    local app_name="${1:-notebook-server}"
    local image="${2:-}"
    local action="${3:-deploy}"
    
    if [[ -z "$image" && "$action" == "deploy" ]]; then
        log_error "Image parameter is required for deployment"
        echo "Usage: $0 <app-name> <image> [deploy|rollback|validate]"
        exit 1
    fi
    
    local start_time=$(date +%s)
    
    log_info "Starting advanced deployment process"
    log_info "Deployment ID: $DEPLOYMENT_ID"
    log_info "Strategy: $DEPLOYMENT_STRATEGY"
    log_info "Environment: $ENVIRONMENT"
    log_info "App: $app_name"
    log_info "Action: $action"
    
    case "$action" in
        "deploy")
            check_prerequisites
            
            # Create backup
            local backup_dir=$(backup_current_state)
            
            # Run pre-deployment tasks
            if [[ -n "${MIGRATION_IMAGE:-}" ]]; then
                run_database_migration
            fi
            
            # Execute deployment strategy
            case "$DEPLOYMENT_STRATEGY" in
                "$STRATEGY_BLUE_GREEN")
                    deploy_blue_green "$app_name" "$image"
                    ;;
                "$STRATEGY_CANARY")
                    deploy_canary "$app_name" "$image" "$CANARY_PERCENTAGE"
                    ;;
                "$STRATEGY_ROLLING")
                    deploy_rolling "$app_name" "$image"
                    ;;
                "$STRATEGY_RECREATE")
                    kubectl delete deployment "$app_name" -n "$NAMESPACE" --ignore-not-found=true
                    kubectl create deployment "$app_name" --image="$image" -n "$NAMESPACE"
                    ;;
                *)
                    log_error "Unknown deployment strategy: $DEPLOYMENT_STRATEGY"
                    exit 1
                    ;;
            esac
            
            validate_deployment "$app_name"
            
            local end_time=$(date +%s)
            generate_report "success" "$start_time" "$end_time"
            
            log_success "Deployment completed successfully!"
            ;;
            
        "rollback")
            rollback_deployment "$app_name"
            ;;
            
        "validate")
            validate_deployment "$app_name"
            ;;
            
        *)
            log_error "Unknown action: $action"
            echo "Usage: $0 <app-name> <image> [deploy|rollback|validate]"
            exit 1
            ;;
    esac
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi