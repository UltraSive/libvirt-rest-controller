package server

import (
	"encoding/json"
	"log"
	"net/http"

	"libvirt-controller/internal/server/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check routes
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Use(AuthMiddleware) // Apply authentication

	r.Route("/v1", func(r chi.Router) {
		// Host-related routes
		r.Route("/host", func(r chi.Router) {
			r.Post("/statistics", handlers.SystemStatsHandler)
			// Add more host-related routes here if needed
		})

		// Domain-related routes
		r.Route("/domain", func(r chi.Router) {
			r.Post("/", handlers.DefineDomainHandler) // Create a VM.
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handlers.RetrieveDomainHandler)          // Get information about VM.
				r.Delete("/", handlers.DeleteDomainHandler)         // Delete a VM.
				r.Post("/cloud-init", handlers.CloudInitHandler)    // Create/Update Cloud Init image
				r.Post("/start", handlers.StartDomainHandler)       // Turn on the VM
				r.Post("/start", handlers.StartDomainHandler)       // Turn on the VM
				r.Post("/reboot", handlers.RebootDomainHandler)     // Reboot the VM
				r.Post("/reset", handlers.RebootDomainHandler)      // Reboot the VM
				r.Post("/shutdowm", handlers.ShutdownDomainHandler) // Shutdown the VM
				r.Post("/stop", handlers.StopDomainHandler)         // Power off the VM
				r.Post("/elevate", handlers.ElevateVMHandler)       // Snapshot the VM
				r.Post("/commit", handlers.CommitVMHandler)         // Commit snapshot changes the VM
				r.Post("/revert", handlers.RevertVMHandler)         // Revert snapshot changes the VM
			})
		})

		// Disk-related routes
		r.Route("/disk", func(r chi.Router) {
			r.Post("/", handlers.CreateDiskHandler)
			r.Route("/{id}", func(r chi.Router) {
				r.Post("/resize", handlers.ResizeDiskHandler)
				r.Delete("/", handlers.DeleteDiskHandler)
				//r.Post("/migrate", handlers.MigrateDiskHandler)    // Migrate Disk to new hypervisor
			})
			// Add more host-related routes here if needed
		})

	})

	return r
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}
