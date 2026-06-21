package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/containerscope/backend/internal/middleware"
	"github.com/containerscope/backend/internal/store"
)

type VulnHandler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewVulnHandler(store *store.Store, logger *slog.Logger) *VulnHandler {
	return &VulnHandler{store: store, logger: logger}
}

type ScanRequest struct {
	Image   string `json:"image"`
	ImageID string `json:"image_id"`
}

type ScanResult struct {
	ScanID          uuid.UUID             `json:"scan_id"`
	Image           string                `json:"image"`
	CriticalCount   int                   `json:"critical_count"`
	HighCount       int                   `json:"high_count"`
	MediumCount     int                   `json:"medium_count"`
	LowCount        int                   `json:"low_count"`
	TotalCount      int                   `json:"total_count"`
	Vulnerabilities []store.Vulnerability `json:"vulnerabilities"`
}

func (h *VulnHandler) ListScans(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	_ = orgID

	scans, err := h.store.ListVulnScans(r.Context(), connID)
	if err != nil {
		h.logger.Error("listing vuln scans", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, scans)
}

func (h *VulnHandler) GetScan(w http.ResponseWriter, r *http.Request) {
	scanIDStr := chi.URLParam(r, "scanID")
	scanID, err := uuid.Parse(scanIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid scan id")
		return
	}

	vulns, err := h.store.ListVulnerabilities(r.Context(), scanID)
	if err != nil {
		h.logger.Error("listing vulnerabilities", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, vulns)
}

func (h *VulnHandler) RecordScan(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	var result ScanResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	scan := &store.VulnerabilityScan{
		ConnectionID:  connID,
		Image:         result.Image,
		CriticalCount: result.CriticalCount,
		HighCount:     result.HighCount,
		MediumCount:   result.MediumCount,
		LowCount:      result.LowCount,
		TotalCount:    result.TotalCount,
	}

	if err := h.store.CreateVulnScan(r.Context(), scan); err != nil {
		h.logger.Error("creating vuln scan", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	for _, v := range result.Vulnerabilities {
		vuln := &store.Vulnerability{
			ScanID:      scan.ID,
			VulnID:      v.VulnID,
			Severity:    v.Severity,
			Package:     v.Package,
			Version:     v.Version,
			FixedIn:     v.FixedIn,
			Title:       v.Title,
			Description: v.Description,
		}
		if err := h.store.CreateVulnerability(r.Context(), vuln); err != nil {
			h.logger.Error("creating vulnerability", "error", err)
		}
	}

	writeJSON(w, http.StatusCreated, scan)
}

func (h *VulnHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	_ = orgID

	scans, err := h.store.ListVulnScans(r.Context(), connID)
	if err != nil {
		h.logger.Error("listing vuln scans", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var totalCritical, totalHigh, totalMedium, totalLow int
	imageVulns := make(map[string]int)

	for _, scan := range scans {
		totalCritical += scan.CriticalCount
		totalHigh += scan.HighCount
		totalMedium += scan.MediumCount
		totalLow += scan.LowCount
		if scan.CriticalCount > 0 || scan.HighCount > 0 {
			imageVulns[scan.Image] = scan.CriticalCount + scan.HighCount
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_scans":     len(scans),
		"critical":        totalCritical,
		"high":            totalHigh,
		"medium":          totalMedium,
		"low":             totalLow,
		"affected_images": imageVulns,
	})
}
