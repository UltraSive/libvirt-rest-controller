package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// ErrorResponse defines the structure of an error response
type ErrorResponseBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// jsonErrorResponse sends a standardized JSON error response
func ErrorResponse(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponseBody{
		Error:   http.StatusText(statusCode), // e.g., "Unauthorized"
		Message: errMsg,
	})
}

// AuthMiddleware checks for a valid Bearer token in the Authorization header
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedToken := os.Getenv("AUTH_TOKEN")
		if expectedToken == "" {
			ErrorResponse(w, "Server misconfiguration: AUTH_TOKEN not set", http.StatusInternalServerError)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			ErrorResponse(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Check for Bearer prefix and extract the token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != expectedToken {
			ErrorResponse(w, "Invalid or missing token", http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed with the request
		next.ServeHTTP(w, r)
	})
}

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

	r.Get("/", s.HelloWorldHandler)

	// Host-related routes
	r.Route("/host", func(r chi.Router) {
		r.Post("/statistics", s.SystemStatsHandler)
		// Add more host-related routes here if needed
	})

	// Host-related routes
	r.Route("/vm", func(r chi.Router) {
		r.Post("/", s.CreateVMHandler) // Create a VM.
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", s.RetrieveVMHandler)          // Get information about VM.
			r.Patch("/", s.UpdateVMHandler)          // Update a VM config.
			r.Delete("/", s.DeleteVMHandler)         // Delete a VM.
			r.Post("/migrate", s.MigrateVMHandler)   // Migrate VM to new hypervisor
			r.Post("/power_on", s.MigrateVMHandler)  // Turn on the VM
			r.Post("/reboot", s.MigrateVMHandler)    // Reboot the VM
			r.Post("/shutdowm", s.MigrateVMHandler)  // Shutdown the VM
			r.Post("/power_off", s.MigrateVMHandler) // Power off the VM
			r.Post("/elevate", s.MigrateVMHandler)   // Snapshot the VM
			r.Post("/commit", s.MigrateVMHandler)    // Commit snapshot changes the VM
			r.Post("/revert", s.MigrateVMHandler)    // Revert snapshot changes the VM
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

// DiskStatsRequest represents the expected request body structure
type DiskStatsRequest struct {
	MountPoints []string `json:"mount_points"`
}

// DiskUsageStat represents disk usage for a specific mount point
type DiskUsageStat struct {
	MountPoint string `json:"mount_point"`
	Used       uint64 `json:"disk_used"`
	Total      uint64 `json:"disk_total"`
}

// SystemStatsHandler handles system statistics retrieval with disk mount points
func (s *Server) SystemStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the request is a POST request
	if r.Method != http.MethodPost {
		ErrorResponse(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode JSON request
	var req DiskStatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, "Invalid JSON request", http.StatusBadRequest)
		log.Printf("error decoding request body: %v", err)
		return
	}

	// Validate that mount points are provided
	if len(req.MountPoints) == 0 {
		ErrorResponse(w, "No mount points provided", http.StatusBadRequest)
		return
	}

	// Get CPU usage
	cpuPercentages, err := cpu.Percent(0, false)
	if err != nil {
		log.Printf("error getting CPU usage: %v", err)
		cpuPercentages = []float64{0}
	}

	// Get memory usage
	memStats, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("error getting memory stats: %v", err)
		memStats = &mem.VirtualMemoryStat{}
	}

	// Get system uptime
	hostStats, err := host.Info()
	if err != nil {
		log.Printf("error getting host stats: %v", err)
		hostStats = &host.InfoStat{}
	}

	// Collect disk usage for specified mount points
	var diskUsageStats []DiskUsageStat
	for _, mount := range req.MountPoints {
		diskStats, err := disk.Usage(mount)
		if err != nil {
			log.Printf("error getting disk stats for mount %s: %v", mount, err)
			continue
		}
		diskUsageStats = append(diskUsageStats, DiskUsageStat{
			MountPoint: mount,
			Used:       diskStats.Used,
			Total:      diskStats.Total,
		})
	}

	stats := struct {
		CPUUsage    []float64       `json:"cpu_usage"`
		MemoryUsage uint64          `json:"memory_used"`
		MemoryTotal uint64          `json:"memory_total"`
		Uptime      uint64          `json:"uptime"`
		DiskUsage   []DiskUsageStat `json:"disk_usage"`
	}{
		CPUUsage:    cpuPercentages,
		MemoryUsage: memStats.Used,
		MemoryTotal: memStats.Total,
		Uptime:      hostStats.Uptime,
		DiskUsage:   diskUsageStats,
	}

	// Encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("error marshalling response: %v", err)
		ErrorResponse(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DownloadFile downloads a file from a given URL and saves it to a specified path
func DownloadFile(url, filePath string) error {
	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// ResizeDisk resizes the disk image to the desired size in GB.
func ResizeDisk(imagePath string, sizeGB int) error {
	// Convert size in GB to the required format for qemu-img (e.g., "10G" for 10 GB)
	size := fmt.Sprintf("%dG", sizeGB)

	// Construct the qemu-img command to resize the disk
	cmd := exec.Command("qemu-img", "resize", imagePath, size)

	// Run the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to resize disk image: %w", err)
	}

	log.Printf("Successfully resized the disk image to %d GB", sizeGB)
	return nil
}

// CreateVMHandler handles VM creation
func (s *Server) CreateVMHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request (same as before)
	var req interface{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Type assertion for the root object (map)
	dataMap, dataPresent := req.(map[string]interface{})
	if !dataPresent {
		ErrorResponse(w, "Error: Unable to assert root object", http.StatusBadRequest)
		return
	}

	if vmMap, vmPresent := dataMap["vm"].(map[string]interface{}); vmPresent {
		// Ensure "id" is a string before using it
		id, idPresent := vmMap["id"].(string)
		if !idPresent {
			ErrorResponse(w, "Invalid or missing 'id' field", http.StatusBadRequest)
			return
		}

		// Create VM directory
		vmDir := filepath.Join("/home/sive/vm", id)

		// Check if the directory already exists
		if _, err := os.Stat(vmDir); err == nil {
			// Directory already exists
			ErrorResponse(w, "VM directory already exists", http.StatusConflict)
			log.Printf("VM directory already exists: %v", vmDir)
			return
		} else if !os.IsNotExist(err) {
			// Error other than "does not exist"
			ErrorResponse(w, "Failed to check VM directory", http.StatusInternalServerError)
			log.Printf("Error checking VM directory: %v", err)
			return
		}

		// Directory does not exist, create it
		if err := os.MkdirAll(vmDir, 0755); err != nil {
			ErrorResponse(w, "Failed to create VM directory", http.StatusInternalServerError)
			log.Printf("Error creating VM directory: %v", err)
			return
		}

		// Save request body as a JSON file inside the VM directory so we can keep track of state
		reqJSONPath := filepath.Join(vmDir, "server.json")
		reqJSON, err := json.Marshal(req)
		if err != nil {
			ErrorResponse(w, "Failed to serialize request body", http.StatusInternalServerError)
			log.Printf("Error serializing request body: %v", err)
			return
		}

		if err := os.WriteFile(reqJSONPath, reqJSON, 0644); err != nil {
			ErrorResponse(w, "Failed to save request body", http.StatusInternalServerError)
			log.Printf("Error writing request body JSON: %v", err)
			return
		}

		// Write the Libvirt XML config to the directory
		reqXMLPath := filepath.Join(vmDir, "server.xml")
		xmlConfig, xmlPresent := dataMap["xmlConfig"].(string)
		if !xmlPresent {
			ErrorResponse(w, "Error finding the XML config", http.StatusBadRequest)
			return
		}

		if err := os.WriteFile(reqXMLPath, []byte(xmlConfig), 0644); err != nil {
			ErrorResponse(w, "Failed to save XML config", http.StatusInternalServerError)
			log.Printf("Error writing XML config: %v", err)
			return
		}

		// Pull the disk image from the template URL
		if templateMap, templatePresent := vmMap["template"].(map[string]interface{}); templatePresent {
			// Ensure "id" is a string before using it
			imageURL, imagePresent := templateMap["imageURL"].(string)
			if !imagePresent {
				ErrorResponse(w, "Invalid or missing 'imageURL' field", http.StatusBadRequest)
				return
			}
			log.Printf("Image URL: %s", imageURL)

			if disksMap, disksPresent := vmMap["disks"].([]interface{}); disksPresent && len(disksMap) > 0 {
				log.Printf("Disks present")

				// Ensure the first disk has a valid ID
				if disk, ok := disksMap[0].(map[string]interface{}); ok {
					if id, idPresent := disk["id"].(float64); idPresent {
						imageID := fmt.Sprintf("%v", id)
						imagePath := filepath.Join(vmDir, imageID+".img")
						log.Printf("Image path: %s", imagePath)

						// Pull the image URL from the template config
						if templateMap, templatePresent := vmMap["template"].(map[string]interface{}); templatePresent {
							if imageURL, imagePresent := templateMap["imageURL"].(string); imagePresent {
								log.Printf("Image URL: %s", imageURL)

								// Download the image
								imageDownloadError := DownloadFile(imageURL, imagePath)
								if imageDownloadError != nil {
									ErrorResponse(w, "Failed to download image", http.StatusBadRequest)
									return
								}

								// Resize the image to a specific size (e.g., 20GB)
								if capacity, ok := disk["capacity"].(float64); ok {
									desiredSizeGB := int(capacity) // Convert to int
									resizeError := ResizeDisk(imagePath, desiredSizeGB)
									if resizeError != nil {
										ErrorResponse(w, fmt.Sprintf("Failed to resize image: %s", resizeError.Error()), http.StatusInternalServerError)
										return
									}
								} else {
									ErrorResponse(w, "Invalid or missing 'capacity' field", http.StatusBadRequest)
									return
								}
							}
						}
					} else {
						ErrorResponse(w, "Invalid or missing 'id' field in disk", http.StatusBadRequest)
						return
					}
				} else {
					ErrorResponse(w, "Error processing disk information", http.StatusBadRequest)
					return
				}
			}
		}

		// Respond
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "VM configuration created",
			"vm_id":   id,
			"path":    vmDir,
		})
	}
}

func (s *Server) RetrieveVMHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) UpdateVMHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

// DeleteVMHandler handles the deletion of a VM directory
func (s *Server) DeleteVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	vmID := chi.URLParam(r, "id")
	vmDir := filepath.Join("/home/sive/vm", vmID)

	// Check if the directory exists
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		ErrorResponse(w, "VM directory does not exist", http.StatusNotFound)
		log.Printf("VM directory not found: %v", vmDir)
		return
	} else if err != nil {
		// Handle other errors (e.g., permission issues)
		ErrorResponse(w, "Failed to check VM directory", http.StatusInternalServerError)
		log.Printf("Error checking VM directory: %v", err)
		return
	}

	// Delete the directory and its contents
	if err := os.RemoveAll(vmDir); err != nil {
		ErrorResponse(w, "Failed to delete VM directory", http.StatusInternalServerError)
		log.Printf("Error deleting VM directory: %v", err)
		return
	}

	// Return a success response
	resp := make(map[string]string)
	resp["message"] = "VM directory deleted successfully"
	resp["vm_id"] = vmID

	// Respond with a JSON message
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) MigrateVMHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}
