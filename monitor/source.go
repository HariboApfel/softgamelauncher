package monitor

import (
	"fmt"
	"gamelauncher/models"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// SourceMonitor monitors game sources for updates
type SourceMonitor struct {
	client *http.Client
}

// NewSourceMonitor creates a new source monitor
func NewSourceMonitor() *SourceMonitor {
	return &SourceMonitor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckForUpdates checks if a game has updates available
func (m *SourceMonitor) CheckForUpdates(game *models.Game) (*UpdateInfo, error) {
	if game.SourceURL == "" {
		return nil, fmt.Errorf("no source URL configured")
	}

	// Check F95zone URLs
	if strings.Contains(game.SourceURL, "f95zone.to") {
		return m.checkF95zoneSource(game)
	}

	// Generic web scraping for other sources
	return m.checkGenericSource(game)
}

// UpdateInfo contains information about available updates
type UpdateInfo struct {
	HasUpdate   bool
	Version     string
	URL         string
	ReleaseDate time.Time
	Description string
}

// checkF95zoneSource performs specialized scraping for F95zone game threads
func (m *SourceMonitor) checkF95zoneSource(game *models.Game) (*UpdateInfo, error) {
	resp, err := m.client.Get(game.SourceURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("F95zone returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// F95zone specific version extraction
	version := m.extractF95zoneVersion(doc)

	// If no version found and this is the first check, try generic extraction
	if version == "" && game.CurrentVersion == "" {
		version = m.extractVersionFromPage(doc)
		if version != "" {
			// Store this as the current version for future comparisons
			game.CurrentVersion = version
		}
	}

	hasUpdate := false
	if version != "" && version != game.CurrentVersion {
		hasUpdate = true
	}

	return &UpdateInfo{
		HasUpdate:   hasUpdate,
		Version:     version,
		URL:         game.SourceURL,
		ReleaseDate: time.Now(),
		Description: fmt.Sprintf("F95zone - Current: %s, Found: %s", game.CurrentVersion, version),
	}, nil
}

// checkGenericSource performs generic web scraping for updates
func (m *SourceMonitor) checkGenericSource(game *models.Game) (*UpdateInfo, error) {
	resp, err := m.client.Get(game.SourceURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("source returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract version using configured selector and pattern
	version := m.extractVersionWithConfig(doc, game)

	// If no version found and this is the first check, try to find any version
	if version == "" && game.CurrentVersion == "" {
		version = m.extractVersionFromPage(doc)
		if version != "" {
			// Store this as the current version for future comparisons
			game.CurrentVersion = version
		}
	}

	hasUpdate := false
	if version != "" && version != game.CurrentVersion {
		hasUpdate = true
	}

	return &UpdateInfo{
		HasUpdate:   hasUpdate,
		Version:     version,
		URL:         game.SourceURL,
		ReleaseDate: time.Now(),
		Description: fmt.Sprintf("Current: %s, Found: %s", game.CurrentVersion, version),
	}, nil
}

// extractVersionWithConfig extracts version using configured selector and pattern
func (m *SourceMonitor) extractVersionWithConfig(doc *goquery.Document, game *models.Game) string {
	// If no custom selector is configured, return empty
	if game.VersionSelector == "" {
		return ""
	}

	var foundVersion string

	// Find elements matching the selector
	doc.Find(game.VersionSelector).Each(func(i int, s *goquery.Selection) {
		if foundVersion != "" {
			return // Already found a version
		}

		text := strings.TrimSpace(s.Text())

		// If a custom pattern is configured, use it
		if game.VersionPattern != "" {
			re, err := regexp.Compile(game.VersionPattern)
			if err == nil {
				matches := re.FindStringSubmatch(text)
				if len(matches) > 1 {
					foundVersion = matches[1] // Return the first capture group
					return
				}
			}
		}

		// Otherwise, check if the text looks like a version
		if m.isVersionString(text) {
			foundVersion = text
			return
		}
	})

	return foundVersion
}

// extractF95zoneVersion extracts version information specifically from F95zone game threads
func (m *SourceMonitor) extractF95zoneVersion(doc *goquery.Document) string {
	// F95zone specific selectors for version information
	// Based on the page structure: "**Version**: 0.514.0.3 with RTP"

	// Look for version information in the game details section
	selectors := []string{
		"strong:contains('Version')", // **Version**: pattern
		"b:contains('Version')",      // <b>Version</b> pattern
		"[class*='version']",         // Any class containing 'version'
		"[id*='version']",            // Any id containing 'version'
		".message",                   // Forum message content
		"#message-1",                 // First message (usually contains game info)
	}

	var foundVersion string

	for _, selector := range selectors {
		if foundVersion != "" {
			break
		}

		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			if foundVersion != "" {
				return
			}

			text := s.Text()

			// Look for F95zone version patterns
			// Pattern: "Version: X.X.X.X with RTP" or "Version: X.X.X.X"
			versionPatterns := []*regexp.Regexp{
				regexp.MustCompile(`(?i)version[:\s]*([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`),
				regexp.MustCompile(`(?i)version[:\s]*([0-9]+\.[0-9]+\.[0-9]+)`),
				regexp.MustCompile(`(?i)version[:\s]*([0-9]+\.[0-9]+)`),
				regexp.MustCompile(`(?i)v([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`),
				regexp.MustCompile(`(?i)v([0-9]+\.[0-9]+\.[0-9]+)`),
			}

			for _, pattern := range versionPatterns {
				matches := pattern.FindStringSubmatch(text)
				if len(matches) > 1 {
					foundVersion = matches[1]
					return
				}
			}
		})
	}

	return foundVersion
}

// extractVersionFromPage tries to extract version information from a webpage
func (m *SourceMonitor) extractVersionFromPage(doc *goquery.Document) string {
	// Look for common version patterns
	selectors := []string{
		"[class*='version']",
		"[id*='version']",
		".version",
		"#version",
		"h1", "h2", "h3",
	}

	var foundVersion string

	for _, selector := range selectors {
		if foundVersion != "" {
			break
		}

		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			if foundVersion != "" {
				return
			}

			text := strings.TrimSpace(s.Text())
			if m.isVersionString(text) {
				foundVersion = text
				return
			}
		})
	}

	return foundVersion
}

// isVersionString checks if a string looks like a version number
func (m *SourceMonitor) isVersionString(s string) bool {
	// Simple version pattern matching
	versionPatterns := []string{
		"v\\d+\\.\\d+",
		"\\d+\\.\\d+\\.\\d+",
		"version \\d+",
	}

	for _, pattern := range versionPatterns {
		if strings.Contains(strings.ToLower(s), pattern) {
			return true
		}
	}

	return false
}
