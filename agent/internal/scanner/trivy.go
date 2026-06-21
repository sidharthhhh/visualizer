package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityUnknown  Severity = "UNKNOWN"
)

type Vulnerability struct {
	ID          string
	Severity    Severity
	Package     string
	Version     string
	FixedIn     string
	Title       string
	Description string
	References  []string
}

type ScanResult struct {
	Image           string
	ImageID         string
	ScanTime        time.Time
	Vulnerabilities []Vulnerability
	Summary         ScanSummary
}

type ScanSummary struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Total    int
}

type Scanner struct {
	logger *slog.Logger
}

func NewScanner(logger *slog.Logger) *Scanner {
	return &Scanner{logger: logger}
}

func (s *Scanner) IsAvailable() bool {
	_, err := exec.LookPath("trivy")
	return err == nil
}

func (s *Scanner) ScanImage(ctx context.Context, image string) (*ScanResult, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("trivy not found in PATH")
	}

	s.logger.Info("scanning image", "image", image)

	cmd := exec.CommandContext(ctx, "trivy",
		"image",
		"--format", "json",
		"--severity", "CRITICAL,HIGH,MEDIUM,LOW",
		"--no-progress",
		image,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running trivy: %w", err)
	}

	return s.parseTrivyOutput(image, output)
}

func (s *Scanner) parseTrivyOutput(image string, output []byte) (*ScanResult, error) {
	var trivyResult struct {
		Results []struct {
			Target          string
			Vulnerabilities []struct {
				VulnerabilityID  string
				Severity         string
				PkgName          string
				InstalledVersion string
				FixedVersion     string
				Title            string
				Description      string
				References       []string
			}
		}
	}

	if err := json.Unmarshal(output, &trivyResult); err != nil {
		return nil, fmt.Errorf("parsing trivy output: %w", err)
	}

	result := &ScanResult{
		Image:    image,
		ScanTime: time.Now(),
	}

	for _, r := range trivyResult.Results {
		for _, v := range r.Vulnerabilities {
			vuln := Vulnerability{
				ID:          v.VulnerabilityID,
				Severity:    Severity(v.Severity),
				Package:     v.PkgName,
				Version:     v.InstalledVersion,
				FixedIn:     v.FixedVersion,
				Title:       v.Title,
				Description: v.Description,
				References:  v.References,
			}

			result.Vulnerabilities = append(result.Vulnerabilities, vuln)

			switch vuln.Severity {
			case SeverityCritical:
				result.Summary.Critical++
			case SeverityHigh:
				result.Summary.High++
			case SeverityMedium:
				result.Summary.Medium++
			case SeverityLow:
				result.Summary.Low++
			}
			result.Summary.Total++
		}
	}

	s.logger.Info("scan complete",
		"image", image,
		"critical", result.Summary.Critical,
		"high", result.Summary.High,
		"medium", result.Summary.Medium,
		"low", result.Summary.Low,
	)

	return result, nil
}

func (s *Scanner) ScanImageAsync(ctx context.Context, image string, callback func(*ScanResult, error)) {
	go func() {
		result, err := s.ScanImage(ctx, image)
		callback(result, err)
	}()
}

type ScanStore struct {
	results map[string]*ScanResult
}

func NewScanStore() *ScanStore {
	return &ScanStore{
		results: make(map[string]*ScanResult),
	}
}

func (ss *ScanStore) Get(image string) *ScanResult {
	return ss.results[image]
}

func (ss *ScanStore) Set(image string, result *ScanResult) {
	ss.results[image] = result
}

func (ss *ScanStore) GetAll() map[string]*ScanResult {
	return ss.results
}

func (ss *ScanStore) GetBySeverity(severity Severity) []*ScanResult {
	var results []*ScanResult
	for _, r := range ss.results {
		hasSeverity := false
		for _, v := range r.Vulnerabilities {
			if v.Severity == severity {
				hasSeverity = true
				break
			}
		}
		if hasSeverity {
			results = append(results, r)
		}
	}
	return results
}
