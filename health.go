/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

var serverStartTime = time.Now()

// HealthStatus represents the health check response
type HealthStatus struct {
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	Uptime      string            `json:"uptime"`
	UptimeSec   int64             `json:"uptime_seconds"`
	GoVersion   string            `json:"go_version"`
	GoRoutines  int               `json:"goroutines"`
	DeviceCount int               `json:"device_count"`
	GroupCount  int               `json:"group_count"`
	Groups      map[string]int    `json:"groups"`
	Memory      MemoryStats       `json:"memory"`
}

// MemoryStats contains memory usage information
type MemoryStats struct {
	Alloc      uint64 `json:"alloc_bytes"`
	TotalAlloc uint64 `json:"total_alloc_bytes"`
	Sys        uint64 `json:"sys_bytes"`
	NumGC      uint32 `json:"num_gc"`
}

func (srv *RttyServer) handleHealth(c *gin.Context) {
	uptime := time.Since(serverStartTime)
	
	deviceCount := 0
	groupCount := 0
	groups := make(map[string]int)

	srv.groups.Range(func(key, value any) bool {
		groupCount++
		g := value.(*DeviceGroup)
		count := int(g.count.Load())
		deviceCount += count
		
		groupName := key.(string)
		if groupName == "" {
			groupName = "(ungrouped)"
		}
		groups[groupName] = count
		return true
	})

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	health := HealthStatus{
		Status:      "healthy",
		Version:     RttysVersion,
		Uptime:      uptime.Round(time.Second).String(),
		UptimeSec:   int64(uptime.Seconds()),
		GoVersion:   runtime.Version(),
		GoRoutines:  runtime.NumGoroutine(),
		DeviceCount: deviceCount,
		GroupCount:  groupCount,
		Groups:      groups,
		Memory: MemoryStats{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
		},
	}

	c.JSON(http.StatusOK, health)
}

func (srv *RttyServer) handleHealthSimple(c *gin.Context) {
	// Simple health check for load balancers/kubernetes probes
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (srv *RttyServer) handleReady(c *gin.Context) {
	// Readiness check - server is ready to accept traffic
	c.JSON(http.StatusOK, gin.H{"ready": true})
}

