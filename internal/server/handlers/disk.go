package handlers

import (
	"net/http"
)

type CreateDiskRequest struct {
	ID       float64 `json:"id"`
	Capacity int     `json:"capacity"`
	Path     string  `json:"path"`
	ImageURL string  `json:"image_url,omitempty"`
}

// CreateDiskHandler handles creating a disk for a VM
func CreateDiskHandler(w http.ResponseWriter, r *http.Request) {

}

type UpdateDiskRequest struct {
	Capacity int    `json:"capacity"`
	Path     string `json:"path"`
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
