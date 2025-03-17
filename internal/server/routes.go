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

	r.Use(AuthMiddleware) // Apply authentication

	// Host-related routes
	r.Route("/host", func(r chi.Router) {
		r.Post("/statistics", handlers.SystemStatsHandler)
		// Add more host-related routes here if needed
	})

	// Host-related routes
	r.Route("/vm", func(r chi.Router) {
		r.Post("/", handlers.CreateVMHandler) // Create a VM.
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", handlers.RetrieveVMHandler)           // Get information about VM.
			r.Patch("/", handlers.UpdateVMHandler)           // Update a VM config.
			r.Delete("/", handlers.DeleteVMHandler)          // Delete a VM.
			r.Post("/migrate", handlers.RetrieveVMHandler)   // Migrate VM to new hypervisor
			r.Post("/power_on", handlers.RetrieveVMHandler)  // Turn on the VM
			r.Post("/reboot", handlers.RetrieveVMHandler)    // Reboot the VM
			r.Post("/shutdowm", handlers.RetrieveVMHandler)  // Shutdown the VM
			r.Post("/power_off", handlers.RetrieveVMHandler) // Power off the VM
			r.Post("/elevate", handlers.RetrieveVMHandler)   // Snapshot the VM
			r.Post("/commit", handlers.RetrieveVMHandler)    // Commit snapshot changes the VM
			r.Post("/revert", handlers.RetrieveVMHandler)    // Revert snapshot changes the VM
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
