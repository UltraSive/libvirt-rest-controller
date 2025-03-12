package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
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
			r.Get("/", s.RetrieveVMHandler)        // Get information about VM.
			r.Patch("/", s.UpdateVMHandler)        // Update a VM config.
			r.Delete("/", s.DeleteVMHandler)       // Delete a VM.
			r.Post("/migrate", s.MigrateVMHandler) // Migrate VM to new hypervisor
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
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode JSON request
	var req DiskStatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		log.Printf("error decoding request body: %v", err)
		return
	}

	// Validate that mount points are provided
	if len(req.MountPoints) == 0 {
		http.Error(w, "No mount points provided", http.StatusBadRequest)
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// VMRequest represents the expected JSON structure for VM creation
type VMRequest struct {
	ID        string           `json:"id"`
	ImageURL  string           `json:"image_url"`
	CPUs      int              `json:"cpus"`
	MemoryMB  int              `json:"memory_mb"`
	Networks  []NetworkConfig  `json:"networks"`   // Network interfaces
	Storage   []StorageConfig  `json:"storage"`    // Additional disks
	CloudInit *CloudInitConfig `json:"cloud_init"` // Cloud-init user data
	XMLConfig string           `json:"xml_config"` // Config tto write to disk
}

// NetworkConfig represents a network interface
type NetworkConfig struct {
	Type       string `json:"type"`        // "bridge", "nat", "macvtap", "direct"
	Network    string `json:"network"`     // Network name or bridge name
	MacAddress string `json:"mac_address"` // Optional custom MAC address
	IP         string `json:"ip"`          // Optional static IP assignment
}

// StorageConfig represents additional storage devices
type StorageConfig struct {
	ID        string `json:"id"`
	Type      string `json:"type"`       // "disk", "cdrom", "nvme"
	Path      string `json:"path"`       // File path or volume name
	TargetDev string `json:"target_dev"` // Target device (e.g., vdb, vdc)
	ReadOnly  bool   `json:"read_only"`  // Mount as read-only
	//CacheMode   string `json:"cache_mode"`   // "none", "writeback", "writethrough"
	//DiskBus     string `json:"disk_bus"`     // "virtio", "sata", "scsi"
	CapacityGB int `json:"capacity_gb"` // Capacity of the disk in GB
}

// CloudInitConfig represents cloud-init user data for VM customization
type CloudInitConfig struct {
	UserData      string `json:"user_data"`       // Cloud-init YAML user data
	MetaData      string `json:"meta_data"`       // Cloud-init metadata
	NetworkConfig string `json:"network_config"`  // Cloud-init network config
	EnableSSHKeys bool   `json:"enable_ssh_keys"` // Inject SSH keys
}

// CreateVMHandler handles VM creation
func (s *Server) CreateVMHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request (same as before)
	var req VMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request (same as before)
	if req.ID == "" || req.CPUs <= 0 || req.MemoryMB <= 0 {
		http.Error(w, "Missing required VM parameters", http.StatusBadRequest)
		return
	}

	// Create VM directory
	vmDir := filepath.Join("/home/sive/vm", req.ID)

	// Check if the directory already exists
	if _, err := os.Stat(vmDir); err == nil {
		// Directory already exists
		http.Error(w, "VM directory already exists", http.StatusConflict)
		log.Printf("VM directory already exists: %v", vmDir)
		return
	} else if !os.IsNotExist(err) {
		// Error other than "does not exist"
		http.Error(w, "Failed to check VM directory", http.StatusInternalServerError)
		log.Printf("Error checking VM directory: %v", err)
		return
	}

	// Directory does not exist, create it
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		http.Error(w, "Failed to create VM directory", http.StatusInternalServerError)
		log.Printf("Error creating VM directory: %v", err)
		return
	}

	// Save request body as a JSON file inside the VM directory so we can keep track of state
	reqJSONPath := filepath.Join(vmDir, "server.json")
	reqJSON, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "Failed to serialize request body", http.StatusInternalServerError)
		log.Printf("Error serializing request body: %v", err)
		return
	}

	if err := os.WriteFile(reqJSONPath, reqJSON, 0644); err != nil {
		http.Error(w, "Failed to save request body", http.StatusInternalServerError)
		log.Printf("Error writing request body JSON: %v", err)
		return
	}

	// Respond
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "VM configuration created",
		"vm_id":   req.ID,
		"path":    vmDir,
	})
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
		http.Error(w, "VM directory does not exist", http.StatusNotFound)
		log.Printf("VM directory not found: %v", vmDir)
		return
	} else if err != nil {
		// Handle other errors (e.g., permission issues)
		http.Error(w, "Failed to check VM directory", http.StatusInternalServerError)
		log.Printf("Error checking VM directory: %v", err)
		return
	}

	// Delete the directory and its contents
	if err := os.RemoveAll(vmDir); err != nil {
		http.Error(w, "Failed to delete VM directory", http.StatusInternalServerError)
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
