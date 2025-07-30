package search

// SearchResult represents a search result returned by any search plugin.
type SearchResult struct {
	Title       string
	Link        string
	Description string
	PubDate     string
	Category    string
	MatchScore  float64 // How well the game name matches
	ImageURL    string  // URL of the image from description or scraped page
	ImagePath   string  // Local path where image is stored (after download)
}

// ImageCandidate is an intermediate structure used by plugins while scraping
// a web page for pictures.  It is kept here so that helper utilities can be
// shared across plugins without duplication.
type ImageCandidate struct {
	URL          string  // The image URL
	Alt          string  // Alt text
	Title        string  // Title attribute
	Class        string  // CSS classes
	Context      string  // Where the image was found (e.g., "thread-starter", "lightbox")
	Score        float64 // Quality score for ranking
	Width        int     // Image width if available
	Height       int     // Image height if available
	IsLightbox   bool    // Whether this is a lightbox/zoomable image
	IsCover      bool    // Whether this appears to be a cover image
	IsScreenshot bool    // Whether this appears to be a screenshot
}
