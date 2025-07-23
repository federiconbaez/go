package security

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrInsufficientRole = errors.New("insufficient role")
	ErrMissingMetadata  = errors.New("missing metadata")
)

type Role string

const (
	RoleGuest  Role = "guest"
	RoleUser   Role = "user"
	RoleAdmin  Role = "admin"
	RoleSystem Role = "system"
)

type AuthClaims struct {
	UserID    string            `json:"user_id"`
	Role      Role              `json:"role"`
	IssuedAt  time.Time         `json:"issued_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	Issuer    string            `json:"issuer"`
	Subject   string            `json:"subject"`
	Audience  []string          `json:"audience"`
	Metadata  map[string]string `json:"metadata"`
}

func (c *AuthClaims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

func (c *AuthClaims) HasRole(role Role) bool {
	return c.Role == role || c.isHigherRole(role)
}

func (c *AuthClaims) isHigherRole(requiredRole Role) bool {
	roleHierarchy := map[Role]int{
		RoleGuest:  0,
		RoleUser:   1,
		RoleAdmin:  2,
		RoleSystem: 3,
	}
	
	currentLevel, exists := roleHierarchy[c.Role]
	if !exists {
		return false
	}
	
	requiredLevel, exists := roleHierarchy[requiredRole]
	if !exists {
		return false
	}
	
	return currentLevel >= requiredLevel
}

type TokenManager struct {
	secretKey     []byte
	issuer        string
	defaultExpiry time.Duration
	mu            sync.RWMutex
	blacklist     map[string]time.Time
	rateLimiter   *RateLimiter
}

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rl.window)
	
	requests := rl.requests[identifier]
	var validRequests []time.Time
	
	for _, req := range requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	
	if len(validRequests) >= rl.limit {
		return false
	}
	
	validRequests = append(validRequests, now)
	rl.requests[identifier] = validRequests
	
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window * 2)
		
		for key, requests := range rl.requests {
			var validRequests []time.Time
			for _, req := range requests {
				if req.After(cutoff) {
					validRequests = append(validRequests, req)
				}
			}
			
			if len(validRequests) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validRequests
			}
		}
		rl.mu.Unlock()
	}
}

func NewTokenManager(secretKey string, issuer string, defaultExpiry time.Duration) *TokenManager {
	return &TokenManager{
		secretKey:     []byte(secretKey),
		issuer:        issuer,
		defaultExpiry: defaultExpiry,
		blacklist:     make(map[string]time.Time),
		rateLimiter:   NewRateLimiter(100, time.Minute),
	}
}

func (tm *TokenManager) GenerateToken(claims *AuthClaims) (string, error) {
	if claims.IssuedAt.IsZero() {
		claims.IssuedAt = time.Now()
	}
	if claims.ExpiresAt.IsZero() {
		claims.ExpiresAt = claims.IssuedAt.Add(tm.defaultExpiry)
	}
	if claims.Issuer == "" {
		claims.Issuer = tm.issuer
	}
	
	tokenData := fmt.Sprintf("%s:%s:%s:%d:%d:%s",
		claims.UserID,
		string(claims.Role),
		claims.Issuer,
		claims.IssuedAt.Unix(),
		claims.ExpiresAt.Unix(),
		claims.Subject,
	)
	
	signature := tm.sign(tokenData)
	token := fmt.Sprintf("%s.%s", hex.EncodeToString([]byte(tokenData)), signature)
	
	return token, nil
}

func (tm *TokenManager) ValidateToken(token string) (*AuthClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}
	
	tokenData, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	
	expectedSignature := tm.sign(string(tokenData))
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSignature)) {
		return nil, ErrInvalidToken
	}
	
	tm.mu.RLock()
	if expiry, blacklisted := tm.blacklist[token]; blacklisted && time.Now().Before(expiry) {
		tm.mu.RUnlock()
		return nil, ErrInvalidToken
	}
	tm.mu.RUnlock()
	
	claims, err := tm.parseTokenData(string(tokenData))
	if err != nil {
		return nil, err
	}
	
	if claims.IsExpired() {
		return nil, ErrTokenExpired
	}
	
	return claims, nil
}

func (tm *TokenManager) parseTokenData(data string) (*AuthClaims, error) {
	parts := strings.Split(data, ":")
	if len(parts) < 6 {
		return nil, ErrInvalidToken
	}
	
	userID := parts[0]
	role := Role(parts[1])
	issuer := parts[2]
	
	issuedAt := time.Unix(0, 0)
	if len(parts) > 3 {
		if timestamp, err := time.Parse("1136239445", parts[3]); err == nil {
			issuedAt = timestamp
		}
	}
	
	expiresAt := time.Unix(0, 0)
	if len(parts) > 4 {
		if timestamp, err := time.Parse("1136239445", parts[4]); err == nil {
			expiresAt = timestamp
		}
	}
	
	subject := ""
	if len(parts) > 5 {
		subject = parts[5]
	}
	
	return &AuthClaims{
		UserID:    userID,
		Role:      role,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
		Issuer:    issuer,
		Subject:   subject,
		Metadata:  make(map[string]string),
	}, nil
}

func (tm *TokenManager) sign(data string) string {
	h := hmac.New(sha256.New, tm.secretKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (tm *TokenManager) RevokeToken(token string, expiry time.Time) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.blacklist[token] = expiry
}

func (tm *TokenManager) RefreshToken(oldToken string) (string, error) {
	claims, err := tm.ValidateToken(oldToken)
	if err != nil {
		return "", err
	}
	
	claims.IssuedAt = time.Now()
	claims.ExpiresAt = claims.IssuedAt.Add(tm.defaultExpiry)
	
	tm.RevokeToken(oldToken, claims.ExpiresAt)
	
	return tm.GenerateToken(claims)
}

type AuthInterceptor struct {
	tokenManager   *TokenManager
	publicMethods  map[string]bool
	requiredRoles  map[string]Role
	enableLogging  bool
	requestTracker map[string]int
	mu             sync.RWMutex
}

func NewAuthInterceptor(tokenManager *TokenManager) *AuthInterceptor {
	return &AuthInterceptor{
		tokenManager:   tokenManager,
		publicMethods:  make(map[string]bool),
		requiredRoles:  make(map[string]Role),
		requestTracker: make(map[string]int),
	}
}

func (ai *AuthInterceptor) AddPublicMethod(method string) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.publicMethods[method] = true
}

func (ai *AuthInterceptor) SetMethodRole(method string, role Role) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.requiredRoles[method] = role
}

func (ai *AuthInterceptor) EnableLogging(enable bool) {
	ai.enableLogging = enable
}

func (ai *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ai.trackRequest(info.FullMethod)
		
		if ai.enableLogging {
			fmt.Printf("Auth interceptor: %s\n", info.FullMethod)
		}
		
		ai.mu.RLock()
		isPublic := ai.publicMethods[info.FullMethod]
		requiredRole := ai.requiredRoles[info.FullMethod]
		ai.mu.RUnlock()
		
		if isPublic {
			return handler(ctx, req)
		}
		
		claims, err := ai.extractAndValidateToken(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
		}
		
		if requiredRole != "" && !claims.HasRole(requiredRole) {
			return nil, status.Errorf(codes.PermissionDenied, "insufficient permissions")
		}
		
		clientIP := ai.extractClientIP(ctx)
		if !ai.tokenManager.rateLimiter.Allow(clientIP) {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}
		
		ctxWithClaims := context.WithValue(ctx, "auth_claims", claims)
		return handler(ctxWithClaims, req)
	}
}

func (ai *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ai.trackRequest(info.FullMethod)
		
		if ai.enableLogging {
			fmt.Printf("Auth stream interceptor: %s\n", info.FullMethod)
		}
		
		ai.mu.RLock()
		isPublic := ai.publicMethods[info.FullMethod]
		requiredRole := ai.requiredRoles[info.FullMethod]
		ai.mu.RUnlock()
		
		if isPublic {
			return handler(srv, stream)
		}
		
		claims, err := ai.extractAndValidateToken(stream.Context())
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
		}
		
		if requiredRole != "" && !claims.HasRole(requiredRole) {
			return status.Errorf(codes.PermissionDenied, "insufficient permissions")
		}
		
		clientIP := ai.extractClientIP(stream.Context())
		if !ai.tokenManager.rateLimiter.Allow(clientIP) {
			return status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}
		
		wrappedStream := &wrappedStream{
			ServerStream: stream,
			ctx:          context.WithValue(stream.Context(), "auth_claims", claims),
		}
		
		return handler(srv, wrappedStream)
	}
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func (ai *AuthInterceptor) extractAndValidateToken(ctx context.Context) (*AuthClaims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, ErrMissingMetadata
	}
	
	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return nil, ErrInvalidToken
	}
	
	token := tokens[0]
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}
	
	return ai.tokenManager.ValidateToken(token)
}

func (ai *AuthInterceptor) extractClientIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "unknown"
	}
	
	if ips := md.Get("x-forwarded-for"); len(ips) > 0 {
		return strings.Split(ips[0], ",")[0]
	}
	
	if ips := md.Get("x-real-ip"); len(ips) > 0 {
		return ips[0]
	}
	
	return "unknown"
}

func (ai *AuthInterceptor) trackRequest(method string) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.requestTracker[method]++
}

func (ai *AuthInterceptor) GetRequestStats() map[string]int {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	
	stats := make(map[string]int)
	for method, count := range ai.requestTracker {
		stats[method] = count
	}
	return stats
}

func GenerateSecretKey() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func ExtractClaimsFromContext(ctx context.Context) (*AuthClaims, bool) {
	claims, ok := ctx.Value("auth_claims").(*AuthClaims)
	return claims, ok
}