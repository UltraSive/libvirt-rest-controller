package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"libvirt-controller/internal/filesystem"
	"libvirt-controller/internal/helpers"
	"libvirt-controller/internal/server/utils"
)

type CreateDiskRequest struct {
	ID       float64 `json:"id"`
	Size     int     `json:"size"`
	Path     string  `json:"path"`
	ImageURL string  `json:"image_url,omitempty"`
}

// CreateDiskHandler handles creating a disk for a VM
func CreateDiskHandler(w http.ResponseWriter, r *http.Request) {
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
	var req CreateDiskRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		utils.JSONErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		log.Println("JSON Unmarshal error:", err) // Print error for debugging
		return
	}

	// filesystem.CreateDirectory will create the directory if it doesn't exist,
	// and do nothing if it already exists.
	if err := filesystem.CreateDirectory(req.Path, 0755); err != nil {
		// Log the error for debugging
		log.Printf("Error creating directory %s: %v", req.Path, err)
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to create disk directory: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Process disk image
	imagePath := filepath.Join(req.Path, fmt.Sprintf("%.0f.img", req.ID))

	if err := filesystem.DownloadCachedFile(req.ImageURL, imagePath, 0660); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to download image from URL %s: %v", req.ImageURL, err), http.StatusInternalServerError)
		return
	}

	if err := helpers.ResizeDisk(imagePath, req.Size); err != nil {
		utils.JSONErrorResponse(w, fmt.Sprintf("Failed to resize disk at %s: %v", imagePath, err), http.StatusInternalServerError)
		return
	}
}

type ResizeDiskRequest struct {
	Size int    `json:"size"`
	Path string `json:"path"`
}

// ResizeDiskHandler handles resizing a disk for a VM
func ResizeDiskHandler(w http.ResponseWriter, r *http.Request) {

}

type DeleteDiskRequest struct {
	Path string `json:"path"`
}

// DeleteDiskHandler handles deleting a VM disk
func DeleteDiskHandler(w http.ResponseWriter, r *http.Request) {

}

// MigrateDiskHandler handles migrating a VM disk to another node
func MigrateDiskHandler(w http.ResponseWriter, r *http.Request) {

}
