package handlers

import (
	"encoding/json"
	"libvirt-controller/internal/server/utils"
	"log"
	"net/http"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

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
func SystemStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Decode JSON request
	var req DiskStatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONErrorResponse(w, "Invalid JSON request", http.StatusBadRequest)
		log.Printf("error decoding request body: %v", err)
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
		utils.JSONErrorResponse(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
