package handlers

import (
	"context"
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
	"libvirt-controller/internal/qemu"
	"libvirt-controller/internal/server/utils"

	"github.com/go-chi/chi/v5"
)

// Request struct to handle expected JSON fields
type DefineRequest struct {
	ID        string `json:"id"`
	XMLConfig string `json:"xml_config"`
}

// DefineDomainHandler handles libvirt domain creation and updates
func DefineDomainHandler(w http.ResponseWriter, r *http.Request) {
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
	var req DefineRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		utils.JSONErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		log.Println("JSON Unmarshal error:", err) // Print error for debugging
		return
	}

	// Validate required fields
	if req.ID == "" {
		utils.JSONErrorResponse(w, "Missing 'id'", http.StatusBadRequest)
		return
	}
	if req.XMLConfig == "" {
		utils.JSONErrorResponse(w, "Missing 'xmlConfig'", http.StatusBadRequest)
		return
	}

	vmID := req.ID
	definitionsDir := os.Getenv("DEFINITIONS_DIR")

	// Basic validation for DEFINITIONS_DIR
	if definitionsDir == "" {
		utils.JSONErrorResponse(w, "DEFINITIONS_DIR environment variable not set", http.StatusInternalServerError)
		return
	}

	// Create VM directory
	vmDir := filepath.Join(definitionsDir, vmID)

	// filesystem.CreateDirectory will create the directory if it doesn't exist,
	// and do nothing if it already exists.
	if err := filesystem.CreateDirectory(vmDir, 0755); err != nil {
		// Log the error for debugging
		log.Printf("Error creating directory %s: %v", vmDir, err)
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to create VM directory: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	// Define the domain (VM) using the saved XML configuration
	xmlConfig := req.XMLConfig

	// filesystem.SaveFile will overwrite "server.xml" if it exists,
	// and create it if it doesn't.
	if err := filesystem.SaveFile(vmDir, "server.xml", []byte(xmlConfig)); err != nil {
		// Log the error for debugging
		log.Printf("Error saving XML config to %s/server.xml: %v", vmDir, err)
		utils.JSONErrorResponse(w, "Failed to save XML config", http.StatusInternalServerError)
		return
	}

	// Define the domain in libvirt
	// Ensure your libvirt.DefineDomain can handle an existing domain definition
	// (e.g., if you're redefining, it should update or detach/attach)
	if _, err := libvirt.DefineDomain(filepath.Join(vmDir, "server.xml")); err != nil {
		// Log the error for debugging
		log.Printf("Error defining domain with libvirt from %s/server.xml: %v", vmDir, err)
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to define domain: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Domain defined
	response := map[string]interface{}{
		"success": true,
		"message": "Domain defined",
		"id":      vmID,
		"path":    vmDir,
	}
	utils.JSONResponse(w, response, http.StatusCreated)
}

// DomainMiddleware ensures that a valid domain exists
func DomainMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the VM ID from the URL parameter
		vmID := chi.URLParam(r, "id")
		if vmID == "" {
			utils.JSONErrorResponse(w, "VM ID missing from URL", http.StatusBadRequest)
			return
		}

		definitionsDir := os.Getenv("DEFINITIONS_DIR")
		if definitionsDir == "" {
			utils.JSONErrorResponse(w, "DEFINITIONS_DIR environment variable not set", http.StatusInternalServerError)
			return
		}

		// 2. Construct VM directory path
		vmDir := filepath.Join(definitionsDir, vmID)

		// 3. Use the helper function to check if the VM directory exists
		exists, err := filesystem.CheckDirectoryExists(vmDir)
		if err != nil {
			// This catches cases where path exists but isn't a directory, or other os.Stat errors
			fmt.Printf("Error during VM directory check %s: %v\n", vmDir, err) // Log for debugging
			if err.Error() == fmt.Sprintf("path '%s' exists but is not a directory", vmDir) {
				utils.JSONErrorResponse(w, fmt.Sprintf("Path '%s' exists but is not a directory for VM ID '%s'.", vmDir, vmID), http.StatusConflict)
			} else {
				utils.JSONErrorResponse(w, fmt.Sprintf("Failed to verify VM directory: %s", err.Error()), http.StatusInternalServerError)
			}
			return
		}
		if !exists {
			// Directory does not exist
			utils.JSONErrorResponse(w, fmt.Sprintf("VM directory for ID '%s' not found.", vmID), http.StatusNotFound)
			return
		}

		// 4. Add vmID and vmDir to the request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, helpers.VMIDKey, vmID)
		ctx = context.WithValue(ctx, helpers.VMDirKey, vmDir)

		// 5. Proceed with the request with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Request struct to handle expected JSON fields
type CloudInitRequest struct {
	MetaData      string `json:"metaData,omitempty"`
	VendorData    string `json:"vendorData,omitempty"`
	UserData      string `json:"userData,omitempty"`
	NetworkConfig string `json:"networkConfig,omitempty"`
}

// CloudInitHandler handles cloud init image generation
func CloudInitHandler(w http.ResponseWriter, r *http.Request) {
	vmID := helpers.MustGetVMID(r.Context())
	vmDir := helpers.MustGetVMDir(r.Context())

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
	var req CloudInitRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		utils.JSONErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		log.Println("JSON Unmarshal error:", err) // Print error for debugging
		return
	}

	// Save CloudInit files
	cloudInitFiles := map[string]string{
		"meta-data":      req.MetaData,
		"vendor-data":    req.VendorData,
		"user-data":      req.UserData,
		"network-config": req.NetworkConfig,
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

	// Respond
	response := map[string]interface{}{
		"message": "cloud-init drive generated",
		"id":      vmID,
		"path":    vmDir,
	}
	utils.JSONResponse(w, response, http.StatusCreated)
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

func RetrieveDomainHandler(w http.ResponseWriter, r *http.Request) {
	vmID := helpers.MustGetVMID(r.Context())

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
	utils.JSONResponse(w, response, http.StatusOK)
}

// DeleteVMHandler handles the deletion of a VM directory
func DeleteDomainHandler(w http.ResponseWriter, r *http.Request) {
	// Get the VM ID from the URL parameter
	vmID := helpers.MustGetVMID(r.Context())
	vmDir := helpers.MustGetVMDir(r.Context())

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
	response := map[string]interface{}{
		"success": true,
		"message": "Domain deleted successfully",
	}
	utils.JSONResponse(w, response, http.StatusOK)
}

func StartDomainHandler(w http.ResponseWriter, r *http.Request) {
	vmID := helpers.MustGetVMID(r.Context())

	// Attempt to start the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.StartDomain(vmID); err != nil {
		log.Printf("Warning: Failed to start VM, it might be already running: %v", err)
	}

	utils.JSONResponse(w, map[string]interface{}{"status": "success"}, http.StatusOK)
}

func RebootDomainHandler(w http.ResponseWriter, r *http.Request) {
	vmID := helpers.MustGetVMID(r.Context())

	// Attempt to reboot the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.RebootDomain(vmID); err != nil {
		log.Printf("Warning: Failed to reboot VM, it might be already running: %v", err)
	}

	utils.JSONResponse(w, map[string]interface{}{"status": "success"}, http.StatusOK)
}

func ResetDomainHandler(w http.ResponseWriter, r *http.Request) {
	vmID := helpers.MustGetVMID(r.Context())

	// Attempt to reset the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.ResetDomain(vmID); err != nil {
		log.Printf("Warning: Failed to reset VM, it might be already running: %v", err)
	}

	utils.JSONResponse(w, map[string]interface{}{"status": "success"}, http.StatusOK)
}

func ShutdownDomainHandler(w http.ResponseWriter, r *http.Request) {
	vmID := helpers.MustGetVMID(r.Context())

	// Attempt to shut down the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.ShutdownDomain(vmID); err != nil {
		log.Printf("Warning: Failed to shut down VM, it might be already off: %v", err)
	}

	utils.JSONResponse(w, map[string]interface{}{"status": "success"}, http.StatusOK)
}

func StopDomainHandler(w http.ResponseWriter, r *http.Request) {
	vmID := helpers.MustGetVMID(r.Context())

	// Attempt to destroy the VM. Log a message if it fails but respond as success.
	if _, err := libvirt.DestroyDomain(vmID); err != nil {
		log.Printf("Warning: Failed to power off VM, it might be already off: %v", err)
	}

	utils.JSONResponse(w, map[string]interface{}{"status": "success"}, http.StatusOK)
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
	response := map[string]interface{}{
		"success": true,
		"message": "Password reset successfully",
		"output":  output,
	}
	utils.JSONResponse(w, response, http.StatusOK)
}
