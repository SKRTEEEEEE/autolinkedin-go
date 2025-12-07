package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linkgen-ai/backend/src/interfaces/handlers"
	"go.uber.org/zap"
)

// APIRouter holds all API route configurations
type APIRouter struct {
	router        *mux.Router
	ideasHandler  *handlers.IdeasHandler
	draftsHandler *handlers.DraftsHandler
	logger        *zap.Logger
}

// APIRouterConfig holds the configuration for APIRouter
type APIRouterConfig struct {
	IdeasHandler  *handlers.IdeasHandler
	DraftsHandler *handlers.DraftsHandler
	Logger        *zap.Logger
}

// NewAPIRouter creates a new APIRouter instance
func NewAPIRouter(config APIRouterConfig) *APIRouter {
	if config.Logger == nil {
		config.Logger, _ = zap.NewProduction()
	}

	router := mux.NewRouter()

	return &APIRouter{
		router:        router,
		ideasHandler:  config.IdeasHandler,
		draftsHandler: config.DraftsHandler,
		logger:        config.Logger,
	}
}

// RegisterRoutes registers all API routes
func (ar *APIRouter) RegisterRoutes() {
	// Register health check
	ar.router.HandleFunc("/health", ar.healthCheck).Methods(http.MethodGet)
	ar.router.HandleFunc("/v1/health", ar.healthCheck).Methods(http.MethodGet)

	// Register ideas routes
	if ar.ideasHandler != nil {
		ar.ideasHandler.RegisterRoutes(ar.router)
	}

	// Register drafts routes
	if ar.draftsHandler != nil {
		ar.draftsHandler.RegisterRoutes(ar.router)
	}

	ar.logger.Info("API routes registered successfully")
}

// healthCheck handles health check endpoint
func (ar *APIRouter) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"linkgen-ai"}`))
}

// GetRouter returns the underlying mux router
func (ar *APIRouter) GetRouter() *mux.Router {
	return ar.router
}

// ServeHTTP implements http.Handler interface
func (ar *APIRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log request
	ar.logger.Debug("incoming request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_addr", r.RemoteAddr),
	)

	// Serve request
	ar.router.ServeHTTP(w, r)
}
