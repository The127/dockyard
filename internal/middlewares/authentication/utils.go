package authentication

import (
	"fmt"
	"net/http"
	"strings"
)

func extractBearerToken(r *http.Request, authorizationHeader string) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("authorization header is missing or invalid")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	return tokenStr, nil
}
