package f95zone

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"

	// Import decoders for common image formats
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gamelauncher/search"

	"github.com/gocolly/colly/v2"
	// Support for additional image formats
	"image/png"

	_ "github.com/gen2brain/avif"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
)

// alias types
type SearchResult = search.SearchResult
type ImageCandidate = search.ImageCandidate

// RSS structs
type F95ZoneRSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category"`
}

// Service implements search.Plugin for F95Zone
type Service struct {
	baseURL    string
	httpClient *http.Client
	imageDir   string
}

var _ search.Plugin = (*Service)(nil)

func (s *Service) Name() string { return "f95zone" }

func NewService() *Service {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	imgDir := filepath.Join(home, ".gamelauncher", "images")
	_ = os.MkdirAll(imgDir, 0755)
	return &Service{
		baseURL:    "https://f95zone.to/sam/latest_alpha/latest_data.php",
		httpClient: &http.Client{Timeout: 30 * time.Second}, // Increased timeout for scraping
		imageDir:   imgDir,
	}
}

func init() { search.RegisterPlugin(NewService()) }

// ---------------- core methods ----------------

func (s *Service) SearchGame(gameName string) ([]SearchResult, error) {
	friendly := s.makeSearchFriendly(gameName)
	searchURL := fmt.Sprintf("%s?cmd=rss&cat=games&search=%s", s.baseURL, url.QueryEscape(friendly))
	resp, err := s.httpClient.Get(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var rss F95ZoneRSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, err
	}
	var results []SearchResult
	for _, item := range rss.Channel.Items {
		score := s.calculateMatchScore(gameName, item.Title)
		if score < 0.4 {
			continue
		}
		results = append(results, SearchResult{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			PubDate:     item.PubDate,
			Category:    item.Category,
			MatchScore:  score,
			ImageURL:    "", // Intentionally blank, will be fetched from source
		})
	}
	if len(results) == 0 && len(gameName) > 4 {
		return s.searchWithFallback(gameName)
	}
	return results, nil
}

// ExtractImageFromSourceURL uses a proper web scraper (Colly) to find the best image.
func (s *Service) ExtractImageFromSourceURL(sourceURL string) (string, error) {
	if sourceURL == "" {
		return "", fmt.Errorf("source URL is empty")
	}

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)
	c.SetRequestTimeout(30 * time.Second)

	var imageURL string
	found := false

	// This selector is now highly specific to the main post body's content wrapper.
	c.OnHTML("article.message-threadStarterPost .bbWrapper", func(e *colly.HTMLElement) {
		// The first image inside this wrapper is the cover.
		// We only want to do this once.
		if found {
			return
		}

		// Prioritize the zoomer div, as it's the most reliable source.
		zoomerSrc := e.ChildAttr("div.lbContainer-zoomer[data-src]", "data-src")
		if zoomerSrc != "" {
			imageURL = e.Request.AbsoluteURL(zoomerSrc)
			fmt.Printf("DEBUG: Found PRIMARY image candidate via zoomer: %s\n", imageURL)
			found = true
			return
		}

		// Fallback to the first standard image if the zoomer isn't there.
		firstImageSrc := e.ChildAttr("img", "src")
		if firstImageSrc != "" {
			imageURL = e.Request.AbsoluteURL(firstImageSrc)
			fmt.Printf("DEBUG: Found PRIMARY image candidate via first img tag: %s\n", imageURL)
			found = true
		}
	})

	if err := c.Visit(sourceURL); err != nil {
		return "", fmt.Errorf("failed to visit URL: %w", err)
	}
	c.Wait()

	if !found || imageURL == "" {
		return "", fmt.Errorf("no suitable image found in post body")
	}

	fullSizeURL := strings.Replace(imageURL, "/thumb/", "/", 1)
	fmt.Printf("DEBUG: Attempting to download image: %s\n", fullSizeURL)

	downloadedPath, err := s.downloadImageURL(fullSizeURL)
	if err != nil {
		fmt.Printf("DEBUG: Download failed: %v\n", err)
		return "", fmt.Errorf("failed to download image: %w", err)
	}

	fmt.Printf("DEBUG: Successfully downloaded image to: %s\n", downloadedPath)
	return downloadedPath, nil
}

func (s *Service) DownloadImageForResult(r *SearchResult) error {
	var imagePath string
	var err error

	if r.Link != "" {
		imagePath, err = s.ExtractImageFromSourceURL(r.Link)
		if err == nil && imagePath != "" {
			r.ImagePath = imagePath
			return nil
		}
		fmt.Printf("Could not extract image from source link %s: %v\n", r.Link, err)
	}

	if r.ImageURL != "" {
		imagePath, err = s.downloadImageURL(r.ImageURL)
		if err == nil && imagePath != "" {
			r.ImagePath = imagePath
			return nil
		}
		fmt.Printf("Could not download fallback image URL %s: %v\n", r.ImageURL, err)
	}

	return fmt.Errorf("failed to acquire image for %s", r.Title)
}

// ---------------- helpers ----------------

func (s *Service) downloadImageURL(imageURL string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("empty image url")
	}
	if strings.HasPrefix(imageURL, "/") {
		imageURL = "https://f95zone.to" + imageURL
	}

	fmt.Printf("DEBUG: downloadImageURL called with: %s\n", imageURL)

	filename := filepath.Base(imageURL)
	if qIndex := strings.Index(filename, "?"); qIndex != -1 {
		filename = filename[:qIndex]
	}

	fmt.Printf("DEBUG: Generated filename: %s\n", filename)

	localPath := filepath.Join(s.imageDir, filename)
	fmt.Printf("DEBUG: Target local path: %s\n", localPath)

	if _, err := os.Stat(localPath); err == nil {
		fmt.Printf("DEBUG: File already exists, returning: %s\n", localPath)
		return localPath, nil
	}

	fmt.Printf("DEBUG: Making HTTP request to: %s\n", imageURL)

	// Create request with proper headers to mimic a browser
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: HTTP response status: %s\n", resp.Status)
	fmt.Printf("DEBUG: Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("DEBUG: Content-Length: %s\n", resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("DEBUG: Downloaded %d bytes\n", len(data))

	// Check if we got HTML instead of an image
	if len(data) > 0 {
		previewLen := 100
		if len(data) < previewLen {
			previewLen = len(data)
		}
		contentStart := string(data[:previewLen])
		if strings.Contains(strings.ToLower(contentStart), "<html") || strings.Contains(strings.ToLower(contentStart), "<!doctype") {
			fmt.Printf("DEBUG: Received HTML instead of image data: %s...\n", contentStart)
			return "", fmt.Errorf("received HTML page instead of image data")
		}
	}

	// Decode the image to validate and potentially convert format
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("image validation failed for %s (format detection): %w", imageURL, err)
	}

	fmt.Printf("DEBUG: Image format validated: %s\n", format)

	// Get original dimensions
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	fmt.Printf("DEBUG: Original image size: %dx%d\n", originalWidth, originalHeight)

	// Resize if image is too large (optimize for UI performance)
	const maxWidth = 800   // Max width for game cover art
	const maxHeight = 1200 // Max height for game cover art

	resizedImg := img
	needsResize := originalWidth > maxWidth || originalHeight > maxHeight

	if needsResize {
		fmt.Printf("DEBUG: Image is large (%dx%d), resizing for better performance\n", originalWidth, originalHeight)

		// Calculate new dimensions maintaining aspect ratio
		var newWidth, newHeight uint
		if originalWidth > originalHeight {
			// Landscape or square - limit by width
			newWidth = maxWidth
			newHeight = 0 // auto-calculate to maintain aspect ratio
		} else {
			// Portrait - limit by height
			newWidth = 0 // auto-calculate to maintain aspect ratio
			newHeight = maxHeight
		}

		resizedImg = resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
		newBounds := resizedImg.Bounds()
		fmt.Printf("DEBUG: Resized to: %dx%d\n", newBounds.Dx(), newBounds.Dy())
	}

	// Always save as PNG for optimal UI performance, whether converted from AVIF or resized
	if format == "avif" || needsResize || strings.HasSuffix(localPath, ".png") == false {
		if format == "avif" {
			fmt.Printf("DEBUG: Converting AVIF to PNG to improve UI performance\n")
		}
		if needsResize {
			fmt.Printf("DEBUG: Saving resized image as PNG\n")
		}

		// Ensure PNG extension
		if !strings.HasSuffix(localPath, ".png") {
			localPath = strings.TrimSuffix(localPath, filepath.Ext(localPath)) + ".png"
		}

		// Create PNG file
		outFile, err := os.Create(localPath)
		if err != nil {
			return "", fmt.Errorf("failed to create PNG file: %w", err)
		}
		defer outFile.Close()

		// Encode as optimized PNG
		err = png.Encode(outFile, resizedImg)
		if err != nil {
			return "", fmt.Errorf("failed to encode as PNG: %w", err)
		}

		fmt.Printf("DEBUG: Successfully optimized and saved as PNG: %s\n", localPath)
	} else {
		// For small non-AVIF formats, save as-is
		err = os.WriteFile(localPath, data, 0666)
		if err != nil {
			return "", fmt.Errorf("failed to write file to %s: %w", localPath, err)
		}

		fmt.Printf("DEBUG: Successfully wrote file to: %s\n", localPath)
	}
	return localPath, nil
}

// --- Other helpers ---

func (s *Service) cleanGameName(name string) string {
	clean := strings.ToLower(name)
	for _, w := range []string{"the", "game", "version", "edition", "deluxe", "premium", "complete", "full", "my", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"} {
		clean = strings.ReplaceAll(clean, " "+w+" ", " ")
	}
	return strings.TrimSpace(clean)
}

func (s *Service) makeSearchFriendly(n string) string {
	out := strings.ReplaceAll(n, "'s", "")
	for _, ch := range []string{"'", "\"", "&", "(", ")", "[", "]", "{", "}", "<", ">", "|", "\\", "/", ":", ";", ",", ".", "!", "?"} {
		out = strings.ReplaceAll(out, ch, "")
	}
	out = strings.TrimSpace(out)
	if len(out) < 3 {
		return s.cleanGameName(n)
	}
	return out
}

func (s *Service) calculateMatchScore(name, title string) float64 {
	name = strings.ToLower(name)
	title = strings.ToLower(title)
	if strings.Contains(title, name) {
		return 1
	}
	nwords := strings.Fields(name)
	if len(nwords) == 0 {
		return 0
	}
	matches := 0
	for _, w := range nwords {
		if len(w) > 2 && strings.Contains(title, w) {
			matches++
		}
	}
	return float64(matches) / float64(len(nwords))
}

func (s *Service) ExtractImageURL(desc string) string {
	re := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)
	m := re.FindStringSubmatch(desc)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func (s *Service) searchWithFallback(gameName string) ([]SearchResult, error) {
	words := strings.Fields(gameName)
	if len(words) == 0 {
		return nil, fmt.Errorf("no words")
	}
	return s.SearchGame(words[0])
}
