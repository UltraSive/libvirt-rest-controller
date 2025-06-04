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
	"libvirt-controller/internal/events"
	"libvirt-controller/internal/libvirt"
	"libvirt-controller/internal/qemu"
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
	NetworkConfig string `json:"networkConfig,omitempty"`
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

	if err := filesystem.DownloadCachedFile(req.VM.Template.ImageURL, imagePath, 0660); err != nil {
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

type QemuAgentStateInfo struct {
	Hostname   string                  `json:"hostname"`
	OSInfo     *qemu.OSInfo            `json:"osInfo"`
	FSInfo     []qemu.FileSystemInfo   `json:"fsInfo"`
	Interfaces []qemu.NetworkInterface `json:"interfaces"`
	Time       *qemu.GuestTime         `json:"time"`
	Users      []qemu.GuestUser        `json:"users"`
}

type VMStatusResponse struct {
	ID         string              `json:"id"`
	Status     string              `json:"status"`
	RemoteInfo *QemuAgentStateInfo `json:"remoteState,omitempty"`
}

func RetrieveVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	vmID := chi.URLParam(r, "id")
	includeRemote := r.URL.Query().Get("remoteState") == "true"

	// Get domain info using the libvirt package
	domInfo, err := libvirt.GetDomainInfo(vmID)
	if err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to get domain info: %s", err),
			http.StatusInternalServerError)
		return
	}

	// Parse the status from the domain info
	status, err := helpers.ParseDomainStatus(domInfo)
	if err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to parse domain status: %s", err),
			http.StatusInternalServerError)
		return
	}

	// Create the response object
	response := VMStatusResponse{
		ID:     vmID,
		Status: status,
	}

	if includeRemote {
		if err := qemu.GuestPing(vmID); err == nil {
			hostname, _ := qemu.GetHostName(vmID)
			osInfo, _ := qemu.GetOSInfo(vmID)
			fsInfo, _ := qemu.GetFileSystemInfo(vmID)
			interfaces, _ := qemu.GetNetworkInterfaces(vmID)
			guestTime, _ := qemu.GetGuestTime(vmID)
			users, _ := qemu.GetLoggedInUsers(vmID)

			response.RemoteInfo = &QemuAgentStateInfo{
				Hostname:   hostname,
				OSInfo:     osInfo,
				FSInfo:     fsInfo,
				Interfaces: interfaces,
				Time:       guestTime,
				Users:      users,
			}
		} else {
			// Optionally log the issue
			log.Printf("Guest agent not available for VM %s: %v", vmID, err)
		}
	}

	// Marshal the response to JSON
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to encode JSON: %s", err),
			http.StatusInternalServerError)
		return
	}
}

// UpdateVMHandler handles VM updates
func UpdateVMHandler(w http.ResponseWriter, r *http.Request) {
 
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

func StartVMHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	// Attempt to start the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.StartDomain(vmID); err != nil {
		log.Printf("Warning: Failed to start VM, it might be already running: %v", err)
	}

	utils.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func RebootVMHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	// Attempt to reboot the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.RebootDomain(vmID); err != nil {
		log.Printf("Warning: Failed to reboot VM, it might be already running: %v", err)
	}

	utils.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func ResetVMHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	// Attempt to reset the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.ResetDomain(vmID); err != nil {
		log.Printf("Warning: Failed to reset VM, it might be already running: %v", err)
	}

	utils.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func ShutdownVMHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	// Attempt to shut down the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.ShutdownDomain(vmID); err != nil {
		log.Printf("Warning: Failed to shut down VM, it might be already off: %v", err)
	}

	utils.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func StopVMHandler(w http.ResponseWriter, r *http.Request) {
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

type ResetPasswordRequest struct {
	Username string `json:"user"`
	Password string `json:"password"`
}

func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	vmID := chi.URLParam(r, "id")

	var request ResetPasswordRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Invalid request body: %s", err),
			http.StatusBadRequest)
		return
	}

	if request.Username == "" || request.Password == "" {
		utils.JSONErrorResponse(w, "Username and password are required",
			http.StatusBadRequest)
		return
	}

	// Construct the command to reset the password.  This is just an example,
	// and the exact command will depend on the guest OS.  Also, BE VERY
	// CAREFUL when constructing commands from user input to avoid command
	// injection vulnerabilities.  Sanitize the username and password!
	command := "chpasswd" // Example command, might be different for your OS
	args := []string{
		fmt.Sprintf("%s:%s", request.Username, request.Password),
	}

	// Execute the command using the qemu guest agent
	output, err := libvirt.QemuAgentExec(vmID, command, args, true)
	if err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to execute command: %s, Output: %s",
			err, output), http.StatusInternalServerError)
		return
	}

	// Return a success response
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"message": "Password reset successfully", "output": output}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to encode JSON: %s", err),
			http.StatusInternalServerError)
		return
	}
}

func MigrateVMHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	//vmID := chi.URLParam(r, "id")
}
