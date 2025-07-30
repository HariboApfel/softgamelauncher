package search

import "fmt"

// Plugin is implemented by any package that can search for games and manage
// the accompanying artwork.
type Plugin interface {
	Name() string

	// SearchGame returns a slice of potential matches for the supplied name.
	SearchGame(gameName string) ([]SearchResult, error)

	// ExtractImageFromSourceURL attempts to scrape an image from a source page.
	ExtractImageFromSourceURL(sourceURL string) (string, error)

	// DownloadImageForResult downloads the image referenced by a result (if any)
	// and updates result.ImagePath accordingly.
	DownloadImageForResult(result *SearchResult) error
}

// global registry that plugins populate from their init() functions.
var registeredPlugins []Plugin

// RegisterPlugin is called by a plugin's init() to make itself available.
func RegisterPlugin(p Plugin) {
	registeredPlugins = append(registeredPlugins, p)
}

// Manager is the façade that the rest of the application talks to.  It
// forwards requests to all registered plugins until one of them returns a
// non-empty result set / nil error.
type Manager struct {
	plugins []Plugin
}

// NewManager constructs a manager using the registered plugin list.
func NewManager() *Manager {
	return &Manager{plugins: registeredPlugins}
}

// SearchGame asks each plugin in order until some results are found.
func (m *Manager) SearchGame(gameName string) ([]SearchResult, error) {
	for _, p := range m.plugins {
		results, err := p.SearchGame(gameName)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}
	return nil, fmt.Errorf("no plugin produced results for %s", gameName)
}

// FindBestMatch runs SearchGame and returns the highest-scoring item.
func (m *Manager) FindBestMatch(gameName string) (*SearchResult, error) {
	results, err := m.SearchGame(gameName)
	if err != nil {
		return nil, err
	}

	best := &results[0]
	for i := 1; i < len(results); i++ {
		if results[i].MatchScore > best.MatchScore {
			best = &results[i]
		}
	}
	return best, nil
}

// ExtractImageFromSourceURL delegates to the first plugin that succeeds.
func (m *Manager) ExtractImageFromSourceURL(url string) (string, error) {
	for _, p := range m.plugins {
		if img, err := p.ExtractImageFromSourceURL(url); err == nil && img != "" {
			return img, nil
		}
	}
	return "", fmt.Errorf("no plugin could extract image from %s", url)
}

// DownloadImageForResult finds a plugin that can handle the result (currently
// we just try all of them) and lets it download the picture.
func (m *Manager) DownloadImageForResult(r *SearchResult) error {
	for _, p := range m.plugins {
		if err := p.DownloadImageForResult(r); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no plugin succeeded downloading image for %s", r.Title)
}
