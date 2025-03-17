package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"libvirt-controller/internal/libvirt"
	"libvirt-controller/internal/server/utils"

	"github.com/go-chi/chi/v5"
)

func setQEMUOwnership(filePath string) error {
	// Set ownership to a user/group that libvirt uses (e.g., qemu)
	// Adjust UID and GID as needed
	uid, gid := 64055, 994
	if err := os.Chown(filePath, uid, gid); err != nil {
		return fmt.Errorf("failed to change ownership: %w", err)
	}

	return nil
}

func createVMDirectory(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("VM directory already exists")
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Printf("Error creating VM directory: %v", err)
		return fmt.Errorf("Failed to create VM directory")
	}
	return nil
}

func saveFile(dir, filename string, data []byte) error {
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, data, 0644) // Write raw bytes directly
}

// DownloadFile downloads a file from a given URL and saves it to a specified path
func DownloadFile(url, name string, mode os.FileMode) error {
	// Create the file
	out, err := os.Create(name)
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

	/*// Set ownership for directory
	if err := setQEMUOwnership(name); err != nil {
		return err
	}*/

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

func downloadAndResizeImage(url, path string, sizeGB int) error {
	if err := DownloadFile(url, path, 0666); err != nil {
		return fmt.Errorf("Failed to download image")
	}
	if err := ResizeDisk(path, sizeGB); err != nil {
		return fmt.Errorf("Failed to resize image: %v", err)
	}
	return nil
}

// Request struct to handle expected JSON fields
type CreateVMRequest struct {
	VM struct {
		ID       string      `json:"id"`
		Template *VMTemplate `json:"template"`
		Disks    []VMDisk    `json:"disks"`
	} `json:"vm"`
	XMLConfig string `json:"xmlConfig"`
}

type VMTemplate struct {
	ImageURL string `json:"imageURL"`
}

type VMDisk struct {
	ID       float64 `json:"id"`
	Capacity int     `json:"capacity"`
}

// CreateVMHandler handles VM creation
func CreateVMHandler(w http.ResponseWriter, r *http.Request) {
	// Read raw request body
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		utils.JSONErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Ensure body is not empty
	if len(rawBody) == 0 {
		utils.JSONErrorResponse(w, "Empty request body", http.StatusBadRequest)
		return
	}

	// Decode JSON request from rawBody
	var req CreateVMRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		utils.JSONErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		log.Println("JSON Unmarshal error:", err) // Print error for debugging
		return
	}

	// Validate required fields
	if req.VM.ID == "" {
		utils.JSONErrorResponse(w, "Missing 'vm.id'", http.StatusBadRequest)
		return
	}
	if req.XMLConfig == "" {
		utils.JSONErrorResponse(w, "Missing 'xmlConfig'", http.StatusBadRequest)
		return
	}
	if req.VM.Template == nil || req.VM.Template.ImageURL == "" {
		utils.JSONErrorResponse(w, "Missing 'template.imageURL'", http.StatusBadRequest)
		return
	}
	if len(req.VM.Disks) == 0 {
		utils.JSONErrorResponse(w, "Missing 'disks'", http.StatusBadRequest)
		return
	}

	vmID := req.VM.ID

	// Create VM directory
	vmDir := filepath.Join("/home/sive/vm", vmID)
	if err := createVMDirectory(vmDir); err != nil {
		utils.JSONErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse the full JSON request body
	var fullRequest map[string]interface{}
	if err := json.Unmarshal(rawBody, &fullRequest); err != nil {
		utils.JSONErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		log.Println("JSON Unmarshal error:", err) // Debugging
		return
	}

	// Properly format the JSON with indentation
	formattedJSON, err := json.MarshalIndent(fullRequest, "", "  ")
	if err != nil {
		utils.JSONErrorResponse(w, "Failed to format JSON", http.StatusInternalServerError)
		return
	}

	// Save JSON request
	if err := saveFile(vmDir, "server.json", formattedJSON); err != nil {
		utils.JSONErrorResponse(w, "Failed to save request body", http.StatusInternalServerError)
		return
	}

	// Define the domain (VM) using the saved XML configuration
	xmlConfig := req.XMLConfig // This is the XML config for the VM

	// Save XML config
	if err := saveFile(vmDir, "server.xml", []byte(xmlConfig)); err != nil {
		utils.JSONErrorResponse(w, "Failed to save XML config", http.StatusInternalServerError)
		return
	}

	// Process disk image
	firstDisk := req.VM.Disks[0]
	imagePath := filepath.Join(vmDir, fmt.Sprintf("%.0f.img", firstDisk.ID))
	if err := downloadAndResizeImage(req.VM.Template.ImageURL, imagePath, firstDisk.Capacity); err != nil {
		utils.JSONErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := libvirt.DefineDomain(vmDir + "/" + "server.xml"); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to define domain: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if _, err := libvirt.StartDomain(vmID); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to start domain: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Respond
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "VM configuration created",
		"vm_id":   vmID,
		"path":    vmDir,
	})
}

func RetrieveVMHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := libvirt.GetConnection()
	if err != nil {
		log.Fatalf("Failed to get libvirt connection: %v", err)
	}

	domains, _, err := conn.ConnectListAllDomains(-1, 0)
	if err != nil {
		log.Fatalf("Failed to list domains: %v", err)
	}

	// Log the count of domains
	log.Printf("Total domains: %d", len(domains))

	for _, domain := range domains {
		log.Println("Domain:", domain.Name)
	}
}

func UpdateVMHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

// DeleteVMHandler handles the deletion of a VM directory
func DeleteVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	vmID := chi.URLParam(r, "id")
	vmDir := filepath.Join("/home/sive/vm", vmID)

	// Check if the directory exists
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		utils.JSONErrorResponse(w, "VM directory does not exist", http.StatusNotFound)
		log.Printf("VM directory not found: %v", vmDir)
		return
	} else if err != nil {
		// Handle other errors (e.g., permission issues)
		utils.JSONErrorResponse(w, "Failed to check VM directory", http.StatusInternalServerError)
		log.Printf("Error checking VM directory: %v", err)
		return
	}

	// Delete the directory and its contents
	if err := os.RemoveAll(vmDir); err != nil {
		utils.JSONErrorResponse(w, "Failed to delete VM directory", http.StatusInternalServerError)
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

func MigrateVMHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}
