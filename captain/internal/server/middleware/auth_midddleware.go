package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
)

func AdminAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminKey := os.Getenv("ADMIN_API_KEY")
		if adminKey == "" {
			functions.RespondwithError(w, http.StatusInternalServerError, "api key not found", fmt.Errorf("api key not found"))
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "ApiKey" {
			http.Error(w, "invalid auth format", http.StatusUnauthorized)
			return
		}

		key := parts[1]
		if key != adminKey {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func WorkerAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workerKey := os.Getenv("WORKER_API_KEY")
		if workerKey == "" {
			functions.RespondwithError(w, http.StatusInternalServerError, "api key not found", fmt.Errorf("api key not found"))
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "ApiKey" {
			http.Error(w, "invalid auth format", http.StatusUnauthorized)
			return
		}

		key := parts[1]
		if key != workerKey {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
