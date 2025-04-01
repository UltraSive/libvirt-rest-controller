package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"libvirt-controller/internal/filesystem"
	"libvirt-controller/internal/helpers"
	"libvirt-controller/internal/libvirt"
	"libvirt-controller/internal/server/utils"

	"github.com/go-chi/chi/v5"
)

// Request struct to handle expected JSON fields
type VMRequest struct {
	VM struct {
		ID       string      `json:"id"`
		Template *VMTemplate `json:"template"`
		Disks    []VMDisk    `json:"disks"`
	} `json:"vm"`
	XMLConfig string    `json:"xmlConfig"`
	CloudInit CloudInit `json:"cloudInit,omitempty"`
}

type VMTemplate struct {
	ImageURL string `json:"imageURL"`
}

type CloudInit struct {
	MetaData      string `json:"metaData,omitempty"`
	VendorData    string `json:"vendorData,omitempty"`
	UserData      string `json:"userData,omitempty"`
	NetworkConfig string `json:"NetworkConfig,omitempty"`
}

type VMDisk struct {
	ID       float64           `json:"id"`
	Capacity int               `json:"capacity"`
	Storage  HypervisorStorage `json:"storage"`
}

type HypervisorStorage struct {
	Path string `json:"path"`
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
	var req VMRequest
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
	firstDisk := req.VM.Disks[0]

	// Create VM directory
	vmDir := filepath.Join(firstDisk.Storage.Path, vmID)
	if err := filesystem.CreateDirectory(vmDir, 0755); err != nil {
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
	if err := filesystem.SaveFile(vmDir, "server.json", formattedJSON); err != nil {
		utils.JSONErrorResponse(w, "Failed to save request body", http.StatusInternalServerError)
		return
	}

	// Define the domain (VM) using the saved XML configuration
	xmlConfig := req.XMLConfig // This is the XML config for the VM

	// Save XML config
	if err := filesystem.SaveFile(vmDir, "server.xml", []byte(xmlConfig)); err != nil {
		utils.JSONErrorResponse(w, "Failed to save XML config", http.StatusInternalServerError)
		return
	}

	// Save CloudInit files
	cloudInitFiles := map[string]string{
		"meta-data":      req.CloudInit.MetaData,
		"vendor-data":    req.CloudInit.VendorData,
		"user-data":      req.CloudInit.UserData,
		"network-config": req.CloudInit.NetworkConfig,
	}

	for fileName, content := range cloudInitFiles {
		if content != "" {
			if err := filesystem.SaveFile(vmDir, fileName, []byte(content)); err != nil {
				utils.JSONErrorResponse(w, fmt.Sprintf("Failed to save '%s' file", fileName), http.StatusInternalServerError)
				return
			}
		}
	}

	// Generate cloud-init ISO
	if err := helpers.GenerateCloudInitISO(vmDir); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to create cloud-init ISO: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Process disk image
	imagePath := filepath.Join(vmDir, fmt.Sprintf("%.0f.img", firstDisk.ID))

	if err := filesystem.DownloadCachedFile(req.VM.Template.ImageURL, imagePath, "/data/images", 0660); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to download image from URL %s: %v", req.VM.Template.ImageURL, err), http.StatusInternalServerError)
		return
	}

	if err := helpers.ResizeDisk(imagePath, firstDisk.Capacity); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to resize disk at %s: %v", imagePath, err), http.StatusInternalServerError)
		return
	}

	// Define and start the VM
	if _, err := libvirt.DefineDomain(vmDir + "/" + "server.xml"); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to define domain: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	/*if _, err := libvirt.StartDomain(vmID); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to start domain: %s", err.Error()), http.StatusInternalServerError)
		return
	}*/

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

// UpdateVMHandler handles VM updates
func UpdateVMHandler(w http.ResponseWriter, r *http.Request) {
	// Read raw request body
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		utils.JSONErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Decode JSON request from rawBody
	var req VMRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		utils.JSONErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		log.Println("JSON Unmarshal error:", err)
		return
	}

	// Validate required fields
	if req.VM.ID == "" {
		utils.JSONErrorResponse(w, "Missing 'vm.id'", http.StatusBadRequest)
		return
	}

	vmID := req.VM.ID
	firstDisk := req.VM.Disks[0]

	// Create VM directory
	vmDir := filepath.Join(firstDisk.Storage.Path, vmID)
	if err := filesystem.CreateDirectory(vmDir, 0755); err != nil {
		utils.JSONErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove existing files, ignoring errors if they don't exist
	filesToRemove := []string{"server.xml", "server.json", "meta-data", "vendor-data", "user-data", "network-config"}
	for _, fileName := range filesToRemove {
		if err := filesystem.DeleteFile(vmDir, fileName); err != nil && !os.IsNotExist(err) {
			utils.JSONErrorResponse(w, fmt.Sprintf("Failed to remove %s: %v", fileName, err), http.StatusInternalServerError)
			return
		}
	}

	// Read the old server.json
	oldServerJSONPath := filepath.Join(vmDir, "server.json")
	oldServerJSON, err := os.ReadFile(oldServerJSONPath)
	if err != nil {
		utils.JSONErrorResponse(w, "Failed to read old server.json", http.StatusInternalServerError)
		return
	}

	var oldReq VMRequest
	if err := json.Unmarshal(oldServerJSON, &oldReq); err != nil {
		utils.JSONErrorResponse(w, "Failed to parse old server.json", http.StatusInternalServerError)
		return
	}

	// Check if disk size needs to be resized
	for i, oldDisk := range oldReq.VM.Disks {
		if len(req.VM.Disks) > i && req.VM.Disks[i].Capacity > oldDisk.Capacity {
			imagePath := filepath.Join(vmDir, fmt.Sprintf("%.0f.img", oldDisk.ID))
			if err := helpers.ResizeDisk(imagePath, req.VM.Disks[i].Capacity); err != nil {
				utils.JSONErrorResponse(w, fmt.Sprintf("Failed to resize disk at %s: %v", imagePath, err), http.StatusInternalServerError)
				return
			}
		}
	}

	// Save new JSON request
	formattedJSON, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		utils.JSONErrorResponse(w, "Failed to format JSON", http.StatusInternalServerError)
		return
	}

	if err := filesystem.SaveFile(vmDir, "server.json", formattedJSON); err != nil {
		utils.JSONErrorResponse(w, "Failed to save server.json", http.StatusInternalServerError)
		return
	}

	// Save new XML config
	if err := filesystem.SaveFile(vmDir, "server.xml", []byte(req.XMLConfig)); err != nil {
		utils.JSONErrorResponse(w, "Failed to save server.xml", http.StatusInternalServerError)
		return
	}

	// Save CloudInit files
	cloudInitFiles := map[string]string{
		"meta-data":      req.CloudInit.MetaData,
		"vendor-data":    req.CloudInit.VendorData,
		"user-data":      req.CloudInit.UserData,
		"network-config": req.CloudInit.NetworkConfig,
	}

	for fileName, content := range cloudInitFiles {
		if content != "" {
			if err := filesystem.SaveFile(vmDir, fileName, []byte(content)); err != nil {
				utils.JSONErrorResponse(w, fmt.Sprintf("Failed to save '%s' file", fileName), http.StatusInternalServerError)
				return
			}
		}
	}

	// Respond
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "VM configuration updated",
		"vm_id":   vmID,
		"path":    vmDir,
	})
}

// DeleteVMHandler handles the deletion of a VM directory
func DeleteVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	vmID := chi.URLParam(r, "id")
	vmDir := filepath.Join("/data/vm", vmID)

	// Attempt to destroy the VM. Log the error if it fails.
	if _, err := libvirt.DestroyDomain(vmID); err != nil {
		log.Printf("Warning: Failed to destroy VM, it might be already off: %v", err)
	}

	// Undefine the VM.
	if _, err := libvirt.UndefineDomain(vmID); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to undefine VM: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete the VM directory.
	if err := filesystem.DeleteDirectory(vmDir); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to delete VM directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with success.
	jsonResp, _ := json.Marshal(map[string]string{"status": "success"})
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func BootVMHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	// Attempt to start the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.StartDomain(vmID); err != nil {
		log.Printf("Warning: Failed to start VM, it might be already running: %v", err)
	}

	utils.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func RestartVMHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	// Attempt to reboot the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.RebootDomain(vmID); err != nil {
		log.Printf("Warning: Failed to reboot VM, it might be already running: %v", err)
	}

	utils.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func ShutdownVMHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	// Attempt to shut down the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.StopDomain(vmID); err != nil {
		log.Printf("Warning: Failed to shut down VM, it might be already off: %v", err)
	}

	utils.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func PowerOffVMHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	// Attempt to destroy the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.DestroyDomain(vmID); err != nil {
		log.Printf("Warning: Failed to power off VM, it might be already off: %v", err)
	}

	utils.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func ElevateVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	//vmID := chi.URLParam(r, "id")
}

func CommitVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	//vmID := chi.URLParam(r, "id")
}

func RevertVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	//vmID := chi.URLParam(r, "id")
}

func MigrateVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	//vmID := chi.URLParam(r, "id")
}
