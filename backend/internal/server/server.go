package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/containerscope/backend/handlers"
	"github.com/containerscope/backend/internal/alerts"
	"github.com/containerscope/backend/internal/config"
	"github.com/containerscope/backend/internal/flows"
	"github.com/containerscope/backend/internal/metrics"
	csmiddleware "github.com/containerscope/backend/internal/middleware"
	"github.com/containerscope/backend/internal/store"
	"github.com/containerscope/backend/internal/ws"
)

const version = "0.1.0"

type Server struct {
	router *chi.Mux
	logger *slog.Logger
	pool   *pgxpool.Pool
	hub    *ws.Hub
}

func New(logger *slog.Logger, pool *pgxpool.Pool, cfg *config.Config) *Server {
	hub := ws.NewHub(logger)
	go hub.Run()

	s := &Server{
		router: chi.NewRouter(),
		logger: logger,
		pool:   pool,
		hub:    hub,
	}
	s.setupRoutes(cfg)
	return s
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) Hub() *ws.Hub {
	return s.hub
}

func (s *Server) setupRoutes(cfg *config.Config) {
	st := store.New(s.pool)
	authHandler := handlers.NewAuthHandler(st, &cfg.Auth, s.logger)
	orgHandler := handlers.NewOrgHandler(st, s.logger)
	connHandler := handlers.NewConnectionHandler(st, s.logger)
	topologyHandler := handlers.NewTopologyHandler(st, s.logger)
	containerHandler, err := handlers.NewContainerHandler(st, s.logger)
	if err != nil {
		s.logger.Error("creating container handler", "error", err)
	}
	agentHandler := handlers.NewAgentHandler(s.pool, s.logger)

	metricsClient := metrics.NewClient(cfg.Metrics.VictoriaMetricsURL, s.logger)
	metricsHandler := handlers.NewMetricsHandler(st, metricsClient, s.logger)

	vulnHandler := handlers.NewVulnHandler(st, s.logger)
	alertEngine := alerts.NewEngine(s.logger)
	alertHandler := handlers.NewAlertHandler(alertEngine, s.logger)
	healthHandler := handlers.NewHealthHandler(s.pool, cfg.Metrics.VictoriaMetricsURL, s.logger)
	execHandler, _ := handlers.NewExecHandler(s.logger)

	var flowHandler *handlers.FlowHandler
	flowClient, err := flows.NewClient(
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.User,
		cfg.ClickHouse.Password,
		s.logger,
	)
	if err != nil {
		s.logger.Warn("ClickHouse not available, flow features disabled", "error", err)
	} else {
		flowHandler = handlers.NewFlowHandler(st, flowClient, s.logger)
	}

	s.router.Use(chimiddleware.RequestID)
	s.router.Use(chimiddleware.RealIP)
	s.router.Use(csmiddleware.SecurityHeaders)
	s.router.Use(csmiddleware.RequestLogger(s.logger))
	s.router.Use(chimiddleware.Recoverer)
	s.router.Use(chimiddleware.Heartbeat("/ping"))
	s.router.Use(csmiddleware.RateLimitMiddleware(100))
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://*.containerscope.io"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s.router.Get("/healthz", s.handleHealthz)
	s.router.Get("/healthz/services", healthHandler.GetServicesHealth)
	s.router.Get("/version", s.handleVersion)

	s.router.Post("/api/v1/agent/enroll", agentHandler.Enroll)
	s.router.Post("/api/v1/agent/heartbeat", agentHandler.Heartbeat)
	s.router.Post("/api/v1/agent/topology", agentHandler.SyncTopology)

	s.router.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(csmiddleware.AuthMiddleware(cfg.Auth.JWTSecret))

			r.Get("/auth/me", authHandler.Me)
			r.Post("/orgs", orgHandler.Create)
			r.Get("/orgs", orgHandler.List)

			r.Route("/orgs/{orgID}", func(r chi.Router) {
				r.Use(csmiddleware.RequireRole("viewer", st))

				r.Get("/", orgHandler.Get)
				r.Get("/members", orgHandler.ListMembers)
				r.Get("/audit-logs", orgHandler.AuditLogs)

				r.Route("/connections", func(r chi.Router) {
					r.Post("/", connHandler.Create)
					r.Get("/", connHandler.List)
					r.Get("/{connectionID}", connHandler.Get)
					r.Get("/{connectionID}/hosts", connHandler.ListHosts)
					r.Get("/{connectionID}/topology", topologyHandler.GetTopology)
					r.Get("/{connectionID}/metrics", metricsHandler.GetContainerMetrics)
					r.Get("/{connectionID}/metrics/instant", metricsHandler.GetContainerMetricsInstant)
					r.Get("/{connectionID}/containers/{containerID}", containerHandler.GetContainerByID)
					r.Get("/{connectionID}/containers/{containerID}/logs", containerHandler.GetContainerLogs)
					r.Get("/{connectionID}/containers/{containerID}/stats", containerHandler.GetContainerStats)

					// Vulnerability endpoints
					r.Get("/{connectionID}/vulns", vulnHandler.ListScans)
					r.Post("/{connectionID}/vulns", vulnHandler.RecordScan)
					r.Get("/{connectionID}/vulns/{scanID}", vulnHandler.GetScan)
					r.Get("/{connectionID}/vulns/dashboard", vulnHandler.GetDashboard)

					// Alert endpoints
					r.Get("/{connectionID}/alerts", alertHandler.ListAlerts)
					r.Get("/{connectionID}/alerts/firing", alertHandler.ListFiringAlerts)
					r.Get("/{connectionID}/alerts/rules", alertHandler.ListRules)
					r.Post("/{connectionID}/alerts/rules", alertHandler.CreateRule)
					r.Get("/{connectionID}/alerts/channels", alertHandler.ListChannels)
					r.Post("/{connectionID}/alerts/channels", alertHandler.CreateChannel)
					r.Post("/{connectionID}/alerts/silence", alertHandler.SilenceAlert)

					if flowHandler != nil {
						r.Get("/{connectionID}/flows", flowHandler.ListFlows)
						r.Get("/{connectionID}/bandwidth", flowHandler.GetBandwidth)
					}
				})

				r.Group(func(r chi.Router) {
					r.Use(csmiddleware.RequireRole("admin", st))
					r.Post("/invite", orgHandler.Invite)
					r.Put("/members/{userID}/role", orgHandler.UpdateRole)
					r.Delete("/members/{userID}", orgHandler.RemoveMember)
				})
			})
		})
	})

	s.router.Get("/ws/orgs/{orgID}/connections/{connectionID}", func(w http.ResponseWriter, r *http.Request) {
		orgIDStr := chi.URLParam(r, "orgID")
		connIDStr := chi.URLParam(r, "connectionID")

		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			http.Error(w, "invalid org id", http.StatusBadRequest)
			return
		}

		connID, err := uuid.Parse(connIDStr)
		if err != nil {
			http.Error(w, "invalid connection id", http.StatusBadRequest)
			return
		}

		s.hub.HandleWebSocket(w, r, orgID, connID)
	})

	// Container exec WebSocket endpoint
	if execHandler != nil {
		s.router.Get("/ws/orgs/{orgID}/connections/{connectionID}/containers/{containerID}/exec", execHandler.HandleExec)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := s.pool.Ping(ctx); err != nil {
		s.logger.Error("health check failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"db":     "unavailable",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"db":     "ok",
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"version": version,
	})
}
