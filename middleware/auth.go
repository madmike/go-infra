package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth returns a gin.HandlerFunc that validates a JWT token.
// It extracts "tenant_id" / "tid" and "sub" (user_id) from the claims
// and sets them in the gin context.
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// Extract tenant_id and user_id/sub
		if tid, ok := claims["tenant_id"].(string); ok && tid != "" {
			c.Set("tenant_id", tid)
		}
		if tid, ok := claims["tid"].(string); ok && tid != "" {
			c.Set("tenant_id", tid)
		}
		if sub, ok := claims["sub"].(string); ok {
			c.Set("user_id", sub)
		}

		c.Next()
	}
}

// ServiceAuth returns a gin.HandlerFunc that validates a service-to-service JWT token.
// It checks if the token has the required scopes.
func ServiceAuth(secret string, requiredScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired service token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// Extract scopes
		var tokenScopes []string
		if sc, ok := claims["scopes"].([]interface{}); ok {
			for _, s := range sc {
				if str, ok := s.(string); ok {
					tokenScopes = append(tokenScopes, str)
				}
			}
		}

		// Verify required scopes
		for _, reqScope := range requiredScopes {
			found := false
			for _, tokenScope := range tokenScopes {
				if tokenScope == reqScope {
					found = true
					break
				}
			}
			if !found {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Missing required scope: %s", reqScope)})
				return
			}
		}

		// Set service name and tenant_id in context
		if sub, ok := claims["sub"].(string); ok {
			c.Set("service_name", sub)
		}
		if tid, ok := claims["tenant_id"].(string); ok && tid != "" {
			c.Set("tenant_id", tid)
		}
		if tid, ok := claims["tid"].(string); ok && tid != "" {
			c.Set("tenant_id", tid)
		}

		c.Next()
	}
}
