package search

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gen2brain/avif"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	_ "golang.org/x/image/webp"
)

// F95ZoneRSS represents the RSS feed structure from F95Zone
type F95ZoneRSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Channel represents the channel element in the RSS feed
type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

// Item represents an item in the RSS feed
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category"`
}

// SearchResult represents a search result from F95Zone
type SearchResult struct {
	Title       string
	Link        string
	Description string
	PubDate     string
	Category    string
	MatchScore  float64 // How well the game name matches
	ImageURL    string  // URL of the image from description
	ImagePath   string  // Local path where image is saved
}

// ImageCandidate represents a potential image found during scraping
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

// Service handles game searching functionality
type Service struct {
	baseURL    string
	httpClient *http.Client
	imageDir   string // Directory to store downloaded images
}

// NewService creates a new search service
func NewService() *Service {
	// Create .gamelauncher directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "." // Fallback to current directory
	}

	imageDir := filepath.Join(homeDir, ".gamelauncher", "images")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		fmt.Printf("Warning: Could not create image directory %s: %v\n", imageDir, err)
		imageDir = "." // Fallback to current directory
	}

	return &Service{
		baseURL: "https://f95zone.to/sam/latest_alpha/latest_data.php",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		imageDir: imageDir,
	}
}

// SearchGame searches for a game on F95Zone and returns matching results
func (s *Service) SearchGame(gameName string) ([]SearchResult, error) {
	// Create a search-friendly version for the API (remove special characters)
	searchFriendlyName := s.makeSearchFriendly(gameName)

	fmt.Printf("DEBUG: Original game name: '%s'\n", gameName)
	fmt.Printf("DEBUG: Search-friendly name: '%s'\n", searchFriendlyName)

	// Build the search URL with the search-friendly name
	searchURL := fmt.Sprintf("%s?cmd=rss&cat=games&search=%s",
		s.baseURL, url.QueryEscape(searchFriendlyName))

	// Replace + with %20 for better API compatibility
	searchURL = strings.ReplaceAll(searchURL, "+", "%20")

	fmt.Printf("DEBUG: Search URL: %s\n", searchURL)

	// Make the HTTP request
	resp, err := s.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the RSS feed
	var rss F95ZoneRSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	fmt.Printf("DEBUG: Found %d items in RSS feed\n", len(rss.Channel.Items))

	// Convert items to search results and calculate match scores
	var results []SearchResult
	for i, item := range rss.Channel.Items {
		matchScore := s.calculateMatchScore(gameName, item.Title)

		fmt.Printf("DEBUG: Item %d: '%s' (score: %.2f)\n", i+1, item.Title, matchScore)

		// Only include results with a reasonable match score
		if matchScore > 0.5 {
			// Try to extract image from source URL first, fallback to description
			imageURL := ""
			if item.Link != "" {
				// For F95Zone links, we'll extract from the actual page later
				// For now, just note that we prefer source URL extraction
				fmt.Printf("DEBUG: Will extract image from source URL: %s\n", item.Link)
			}

			// Fallback to description image if needed
			if imageURL == "" {
				imageURL = s.ExtractImageURL(item.Description)
				if imageURL != "" {
					fmt.Printf("DEBUG: Found fallback image from description: %s\n", imageURL)
				}
			}

			results = append(results, SearchResult{
				Title:       item.Title,
				Link:        item.Link,
				Description: item.Description,
				PubDate:     item.PubDate,
				Category:    item.Category,
				MatchScore:  matchScore,
				ImageURL:    imageURL,
				ImagePath:   "", // Will be set when game is added
			})
		}
	}

	fmt.Printf("DEBUG: Returning %d results with score > 0.3\n", len(results))

	// If no good results found, try fallback with first word
	if len(results) == 0 {
		fmt.Printf("DEBUG: No good matches found, trying fallback with first word\n")
		return s.searchWithFallback(gameName)
	}

	return results, nil
}

// searchWithFallback tries searching with just the first word and matches locally
func (s *Service) searchWithFallback(gameName string) ([]SearchResult, error) {
	// Get the first word of the game name
	words := strings.Fields(gameName)
	if len(words) == 0 {
		return nil, fmt.Errorf("no words found in game name")
	}

	firstWord := words[0]
	fmt.Printf("DEBUG: Fallback search with first word: '%s'\n", firstWord)

	// Create search-friendly version of first word
	searchFriendlyFirstWord := s.makeSearchFriendly(firstWord)

	// Build the search URL with just the first word
	searchURL := fmt.Sprintf("%s?cmd=rss&cat=games&search=%s",
		s.baseURL, url.QueryEscape(searchFriendlyFirstWord))

	// Replace + with %20 for better API compatibility
	searchURL = strings.ReplaceAll(searchURL, "+", "%20")

	fmt.Printf("DEBUG: Fallback search URL: %s\n", searchURL)

	// Make the HTTP request
	resp, err := s.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fallback search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the RSS feed
	var rss F95ZoneRSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	fmt.Printf("DEBUG: Fallback found %d items in RSS feed\n", len(rss.Channel.Items))

	// Convert items to search results and calculate match scores
	var results []SearchResult
	for i, item := range rss.Channel.Items {
		matchScore := s.calculateMatchScore(gameName, item.Title)

		fmt.Printf("DEBUG: Fallback Item %d: '%s' (score: %.2f)\n", i+1, item.Title, matchScore)

		// Only include results with a reasonable match score
		if matchScore > 0.5 {
			// Try to extract image from source URL first, fallback to description
			imageURL := ""
			if item.Link != "" {
				// For F95Zone links, we'll extract from the actual page later
				// For now, just note that we prefer source URL extraction
				fmt.Printf("DEBUG: Fallback will extract image from source URL: %s\n", item.Link)
			}

			// Fallback to description image if needed
			if imageURL == "" {
				imageURL = s.ExtractImageURL(item.Description)
				if imageURL != "" {
					fmt.Printf("DEBUG: Fallback found image from description: %s\n", imageURL)
				}
			}

			results = append(results, SearchResult{
				Title:       item.Title,
				Link:        item.Link,
				Description: item.Description,
				PubDate:     item.PubDate,
				Category:    item.Category,
				MatchScore:  matchScore,
				ImageURL:    imageURL,
				ImagePath:   "", // Will be set when game is added
			})
		}
	}

	fmt.Printf("DEBUG: Fallback returning %d results with score > 0.3\n", len(results))
	return results, nil
}

// FindBestMatch searches for a game and returns the best matching result
func (s *Service) FindBestMatch(gameName string) (*SearchResult, error) {
	results, err := s.SearchGame(gameName)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no matches found for game: %s", gameName)
	}

	// Find the result with the highest match score
	bestMatch := &results[0]
	for i := 1; i < len(results); i++ {
		if results[i].MatchScore > bestMatch.MatchScore {
			bestMatch = &results[i]
		}
	}

	return bestMatch, nil
}

// cleanGameName cleans and normalizes a game name for better search matching
func (s *Service) cleanGameName(name string) string {
	// Convert to lowercase
	clean := strings.ToLower(name)

	// Remove common words that don't help with search
	removeWords := []string{
		"the", "game", "version", "edition", "deluxe", "premium", "complete", "full",
		"my", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by",
		"new", "old", "latest", "updated", "final", "beta", "alpha", "demo", "pro", "plus",
	}

	for _, word := range removeWords {
		// Remove word with spaces around it
		clean = strings.ReplaceAll(clean, " "+word+" ", " ")
		// Remove word at start
		clean = strings.TrimPrefix(clean, word+" ")
		// Remove word at end
		clean = strings.TrimSuffix(clean, " "+word)
		// Remove standalone word
		clean = strings.ReplaceAll(clean, word, "")
	}

	// Remove extra whitespace
	clean = strings.TrimSpace(clean)

	// Replace multiple spaces with single space
	for strings.Contains(clean, "  ") {
		clean = strings.ReplaceAll(clean, "  ", " ")
	}

	return clean
}

// makeSearchFriendly creates a search-friendly version of the game name for API calls
func (s *Service) makeSearchFriendly(name string) string {
	// Start with the original name to preserve capitalization
	searchFriendly := name

	// Remove 's (possessive) FIRST - remove both apostrophe and the s
	searchFriendly = strings.ReplaceAll(searchFriendly, "'s", "")

	// Remove special characters that might cause API issues
	specialChars := []string{"'", "'", "\"", "\"", "&", "(", ")", "[", "]", "{", "}", "<", ">", "|", "\\", "/", ":", ";", ",", ".", "!", "?"}
	for _, char := range specialChars {
		searchFriendly = strings.ReplaceAll(searchFriendly, char, "")
	}

	// Replace multiple spaces with single space
	for strings.Contains(searchFriendly, "  ") {
		searchFriendly = strings.ReplaceAll(searchFriendly, "  ", " ")
	}

	// Trim whitespace
	searchFriendly = strings.TrimSpace(searchFriendly)

	// If the result is empty or too short, use the original cleaned name
	if len(searchFriendly) < 3 {
		return s.cleanGameName(name)
	}

	return searchFriendly
}

// calculateMatchScore calculates how well a game name matches a search result title
func (s *Service) calculateMatchScore(gameName, resultTitle string) float64 {
	gameWords := strings.Fields(strings.ToLower(gameName))
	resultWords := strings.Fields(strings.ToLower(resultTitle))

	if len(gameWords) == 0 || len(resultWords) == 0 {
		return 0.0
	}

	// Check for exact title match first (highest priority)
	exactMatch := strings.Contains(strings.ToLower(resultTitle), strings.ToLower(gameName))
	if exactMatch {
		return 1.0 // Perfect match
	}

	// Check for exact phrase match (without version info)
	cleanGameName := s.cleanGameName(gameName)
	cleanResultTitle := s.cleanGameName(resultTitle)

	// Remove version patterns from result title for comparison
	versionPattern := regexp.MustCompile(`\[.*?\]|v\d+\.\d+.*`)
	cleanResultTitle = versionPattern.ReplaceAllString(cleanResultTitle, "")
	cleanResultTitle = strings.TrimSpace(cleanResultTitle)

	if strings.Contains(cleanResultTitle, cleanGameName) || strings.Contains(cleanGameName, cleanResultTitle) {
		return 0.95 // Very close match
	}

	// For multi-word searches, prioritize phrase matches over single word matches
	if len(gameWords) > 1 {
		// Check if the game name appears as a phrase in the result
		gamePhrase := strings.Join(gameWords, " ")
		if strings.Contains(strings.ToLower(resultTitle), gamePhrase) {
			return 0.9 // Good phrase match
		}

		// Check for consecutive word matches
		consecutiveMatches := 0
		for i := 0; i < len(gameWords)-1; i++ {
			wordPair := gameWords[i] + " " + gameWords[i+1]
			if strings.Contains(strings.ToLower(resultTitle), wordPair) {
				consecutiveMatches++
			}
		}
		if consecutiveMatches > 0 {
			return 0.8 + (float64(consecutiveMatches) * 0.05) // Boost for consecutive matches
		}
	}

	// Count how many game words are found in the result title
	matches := 0
	totalWords := len(gameWords)

	for _, gameWord := range gameWords {
		// Skip very short words (less than 3 characters)
		if len(gameWord) < 3 {
			totalWords--
			continue
		}

		// Check for exact word matches first
		exactWordMatch := false
		for _, resultWord := range resultWords {
			if resultWord == gameWord {
				matches++
				exactWordMatch = true
				break
			}
		}

		// If no exact match, check for partial matches
		if !exactWordMatch {
			for _, resultWord := range resultWords {
				if strings.Contains(resultWord, gameWord) || strings.Contains(gameWord, resultWord) {
					matches++
					break
				}
			}
		}
	}

	// Calculate base score based on percentage of words matched
	if totalWords == 0 {
		return 0.0
	}

	score := float64(matches) / float64(totalWords)

	// Penalize for very short matches (likely false positives)
	if len(gameWords) < 3 && score < 0.8 {
		score *= 0.5
	}

	// Boost score for longer, more specific matches
	if len(gameWords) >= 3 && score > 0.7 {
		score += 0.1
	}

	// Penalize for very generic words that match
	genericWords := []string{"my", "the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"}
	for _, word := range genericWords {
		if strings.Contains(strings.ToLower(gameName), word) && strings.Contains(strings.ToLower(resultTitle), word) {
			score *= 0.9 // Slight penalty for generic word matches
		}
	}

	// Additional penalty for single-word searches that match generic terms
	if len(gameWords) == 1 && len(gameWords[0]) < 8 {
		score *= 0.7 // Significant penalty for short single-word searches
	}

	// Cap score at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// ExtractImageURL extracts image URL from description
func (s *Service) ExtractImageURL(description string) string {
	// Look for img src pattern in description (both CDATA and direct HTML)
	imgPattern := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*>`)
	matches := imgPattern.FindStringSubmatch(description)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ExtractImageFromSourceURL scrapes a webpage using Colly to find and download images
func (s *Service) ExtractImageFromSourceURL(sourceURL string) (string, error) {
	if sourceURL == "" {
		return "", fmt.Errorf("source URL is empty")
	}

	fmt.Printf("DEBUG: Starting Colly extraction for URL: %s\n", sourceURL)

	// Create a new collector
	c := colly.NewCollector(
		colly.Debugger(&debug.LogDebugger{}),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	// Set timeout
	c.SetRequestTimeout(30 * time.Second)

	// Add some delay between requests to be respectful
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*f95zone.to*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})

	var foundImages []ImageCandidate
	var pageTitle string

	// Capture page title for context
	c.OnHTML("title", func(e *colly.HTMLElement) {
		pageTitle = strings.TrimSpace(e.Text)
		fmt.Printf("DEBUG: Page title: %s\n", pageTitle)
	})

	// Extract images from thread starter post
	if strings.Contains(sourceURL, "f95zone.to") {
		s.setupF95ZoneImageExtraction(c, &foundImages, sourceURL)
	} else {
		s.setupGenericImageExtraction(c, &foundImages, sourceURL)
	}

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("DEBUG: Colly error: %v\n", err)
	})

	// Start scraping
	err := c.Visit(sourceURL)
	if err != nil {
		return "", fmt.Errorf("failed to visit URL: %w", err)
	}

	// Wait for the collector to finish
	c.Wait()

	if len(foundImages) == 0 {
		return "", fmt.Errorf("no suitable images found on page")
	}

	// Find the best image candidate
	bestImage := s.selectBestImageCandidate(foundImages, pageTitle)
	if bestImage == nil {
		return "", fmt.Errorf("no suitable image candidate found")
	}

	fmt.Printf("DEBUG: Selected best image: %s (score: %.2f)\n", bestImage.URL, bestImage.Score)

	// Try downloading the selected image
	imagePath, err := s.downloadImage(bestImage.URL)
	if err != nil {
		// If the best image fails, try a few alternative candidates
		fmt.Printf("DEBUG: Best image failed (%v), trying alternative candidates\n", err)

		// Sort candidates by score and try alternatives
		alternativeCandidates := s.sortCandidatesByScore(foundImages)
		for _, candidate := range alternativeCandidates {
			if candidate.URL == bestImage.URL {
				continue // Skip the one that already failed
			}

			fmt.Printf("DEBUG: Trying alternative candidate: %s (score: %.2f)\n", candidate.URL, candidate.Score)
			altImagePath, altErr := s.downloadImage(candidate.URL)
			if altErr == nil {
				fmt.Printf("DEBUG: Successfully downloaded alternative image: %s\n", altImagePath)
				return altImagePath, nil
			} else {
				fmt.Printf("DEBUG: Alternative failed: %v\n", altErr)
			}
		}

		return "", fmt.Errorf("failed to download image from %s: %w", bestImage.URL, err)
	}

	return imagePath, nil
}

// setupF95ZoneImageExtraction sets up Colly handlers for F95Zone specific image extraction
func (s *Service) setupF95ZoneImageExtraction(c *colly.Collector, foundImages *[]ImageCandidate, sourceURL string) {
	fmt.Printf("DEBUG: Setting up F95Zone image extraction\n")

	// Target the thread starter post specifically
	c.OnHTML(".message-threadStarterPost", func(e *colly.HTMLElement) {
		fmt.Printf("DEBUG: Found thread starter post\n")

		// Look for lightbox images first (highest priority)
		e.ForEach(".lbContainer img[data-zoom-target]", func(i int, img *colly.HTMLElement) {
			s.processImageCandidate(img, foundImages, "thread-starter-lightbox", sourceURL, true)
		})

		// Look for lightbox zoomer divs that might have data-src
		e.ForEach(".lbContainer-zoomer", func(i int, div *colly.HTMLElement) {
			dataSrc := div.Attr("data-src")
			if dataSrc != "" {
				// Create a fake image candidate from the zoomer div
				s.processZoomerCandidate(div, foundImages, "thread-starter-zoomer", sourceURL, dataSrc)
			}
		})

		// Look for bbImage with zoom target
		e.ForEach(".bbImage[data-zoom-target]", func(i int, img *colly.HTMLElement) {
			s.processImageCandidate(img, foundImages, "thread-starter-bb-zoom", sourceURL, true)
		})

		// Look for wrapped images
		e.ForEach(".bbImageWrapper img", func(i int, img *colly.HTMLElement) {
			s.processImageCandidate(img, foundImages, "thread-starter-wrapped", sourceURL, false)
		})

		// Look for any bbImage in thread starter
		e.ForEach(".bbImage", func(i int, img *colly.HTMLElement) {
			s.processImageCandidate(img, foundImages, "thread-starter-bb", sourceURL, false)
		})

		// Look for any other images in message content
		e.ForEach(".message-userContent img", func(i int, img *colly.HTMLElement) {
			s.processImageCandidate(img, foundImages, "thread-starter-content", sourceURL, false)
		})

		// Also look for linked images (a[href] containing image URLs)
		e.ForEach("a[href]", func(i int, link *colly.HTMLElement) {
			href := link.Attr("href")
			if s.isImageURL(href) {
				s.processLinkCandidate(link, foundImages, "thread-starter-link", sourceURL, href)
			}
		})
	})
}

// setupGenericImageExtraction sets up Colly handlers for generic website image extraction
func (s *Service) setupGenericImageExtraction(c *colly.Collector, foundImages *[]ImageCandidate, sourceURL string) {
	fmt.Printf("DEBUG: Setting up generic image extraction\n")

	// Look for common image patterns
	selectors := []string{
		"img[alt*='preview']", "img[alt*='Preview']",
		"img[alt*='cover']", "img[alt*='Cover']",
		"img[alt*='game']", "img[alt*='Game']",
		"img[class*='cover']", "img[class*='preview']",
		"img[class*='thumbnail']", "img[class*='hero']",
		".preview img", ".cover img", ".thumbnail img",
	}

	for _, selector := range selectors {
		c.OnHTML(selector, func(e *colly.HTMLElement) {
			s.processImageCandidate(e, foundImages, "generic", sourceURL, false)
		})
	}

	// Fallback: any image
	c.OnHTML("img", func(e *colly.HTMLElement) {
		s.processImageCandidate(e, foundImages, "fallback", sourceURL, false)
	})
}

// processImageCandidate processes a found image element and adds it to candidates if suitable
func (s *Service) processImageCandidate(img *colly.HTMLElement, foundImages *[]ImageCandidate, context, sourceURL string, isLightbox bool) {
	// Get image URL from various attributes
	imgURL := img.Attr("data-url")
	if imgURL == "" {
		imgURL = img.Attr("src")
		if imgURL == "" {
			imgURL = img.Attr("data-src")
		}
	}

	if imgURL == "" {
		return
	}

	// Convert thumbnail URLs to full-size URLs
	imgURL = s.convertThumbnailToFullSize(imgURL)

	// Convert relative URLs to absolute
	if !strings.HasPrefix(imgURL, "http") {
		baseURL, err := url.Parse(sourceURL)
		if err == nil {
			if parsedImgURL, err := url.Parse(imgURL); err == nil {
				imgURL = baseURL.ResolveReference(parsedImgURL).String()
			}
		}
	}

	alt := img.Attr("alt")
	title := img.Attr("title")
	class := img.Attr("class")

	// Skip unwanted images
	if s.shouldSkipImage(imgURL, alt, class) {
		fmt.Printf("DEBUG: Skipping image: %s (reason: unwanted type)\n", imgURL)
		return
	}

	// Parse dimensions if available
	width, height := s.parseImageDimensions(img)

	// Determine if this is a screenshot
	isScreenshot := strings.Contains(strings.ToLower(imgURL), "screenshot") ||
		strings.Contains(strings.ToLower(alt), "screenshot")

	// Determine if this is a cover image
	isCover := strings.Contains(strings.ToLower(alt), "cover") ||
		strings.Contains(strings.ToLower(imgURL), "cover") ||
		strings.Contains(strings.ToLower(class), "cover")

	candidate := ImageCandidate{
		URL:          imgURL,
		Alt:          alt,
		Title:        title,
		Class:        class,
		Context:      context,
		Score:        s.calculateImageScore(imgURL, alt, class, context, isLightbox, isCover, isScreenshot, width, height),
		Width:        width,
		Height:       height,
		IsLightbox:   isLightbox,
		IsCover:      isCover,
		IsScreenshot: isScreenshot,
	}

	fmt.Printf("DEBUG: Found image candidate: %s (context: %s, score: %.2f, lightbox: %t, cover: %t, screenshot: %t)\n",
		imgURL, context, candidate.Score, isLightbox, isCover, isScreenshot)

	*foundImages = append(*foundImages, candidate)
}

// processZoomerCandidate processes a lightbox zoomer div that contains data-src
func (s *Service) processZoomerCandidate(div *colly.HTMLElement, foundImages *[]ImageCandidate, context, sourceURL, dataSrc string) {
	// Convert thumbnail URLs to full-size URLs
	imgURL := s.convertThumbnailToFullSize(dataSrc)

	// Convert relative URLs to absolute
	if !strings.HasPrefix(imgURL, "http") {
		baseURL, err := url.Parse(sourceURL)
		if err == nil {
			if parsedImgURL, err := url.Parse(imgURL); err == nil {
				imgURL = baseURL.ResolveReference(parsedImgURL).String()
			}
		}
	}

	// Skip unwanted images
	if s.shouldSkipImage(imgURL, "", "") {
		fmt.Printf("DEBUG: Skipping zoomer image: %s (reason: unwanted type)\n", imgURL)
		return
	}

	// Determine if this is a screenshot
	isScreenshot := strings.Contains(strings.ToLower(imgURL), "screenshot")

	// Determine if this is a cover image
	isCover := strings.Contains(strings.ToLower(imgURL), "cover")

	candidate := ImageCandidate{
		URL:          imgURL,
		Alt:          "",
		Title:        "",
		Class:        "",
		Context:      context,
		Score:        s.calculateImageScore(imgURL, "", "", context, true, isCover, isScreenshot, 0, 0), // Zoomer = lightbox
		Width:        0,
		Height:       0,
		IsLightbox:   true,
		IsCover:      isCover,
		IsScreenshot: isScreenshot,
	}

	fmt.Printf("DEBUG: Found zoomer candidate: %s (context: %s, score: %.2f, lightbox: %t, cover: %t, screenshot: %t)\n",
		imgURL, context, candidate.Score, true, isCover, isScreenshot)

	*foundImages = append(*foundImages, candidate)
}

// processLinkCandidate processes a link that points to an image
func (s *Service) processLinkCandidate(link *colly.HTMLElement, foundImages *[]ImageCandidate, context, sourceURL, href string) {
	// Convert thumbnail URLs to full-size URLs
	imgURL := s.convertThumbnailToFullSize(href)

	// Convert relative URLs to absolute
	if !strings.HasPrefix(imgURL, "http") {
		baseURL, err := url.Parse(sourceURL)
		if err == nil {
			if parsedImgURL, err := url.Parse(imgURL); err == nil {
				imgURL = baseURL.ResolveReference(parsedImgURL).String()
			}
		}
	}

	// Skip unwanted images
	if s.shouldSkipImage(imgURL, "", "") {
		fmt.Printf("DEBUG: Skipping linked image: %s (reason: unwanted type)\n", imgURL)
		return
	}

	// Determine if this is a screenshot
	isScreenshot := strings.Contains(strings.ToLower(imgURL), "screenshot")

	// Determine if this is a cover image
	isCover := strings.Contains(strings.ToLower(imgURL), "cover")

	candidate := ImageCandidate{
		URL:          imgURL,
		Alt:          "",
		Title:        "",
		Class:        "",
		Context:      context,
		Score:        s.calculateImageScore(imgURL, "", "", context, false, isCover, isScreenshot, 0, 0),
		Width:        0,
		Height:       0,
		IsLightbox:   false,
		IsCover:      isCover,
		IsScreenshot: isScreenshot,
	}

	fmt.Printf("DEBUG: Found link candidate: %s (context: %s, score: %.2f, lightbox: %t, cover: %t, screenshot: %t)\n",
		imgURL, context, candidate.Score, false, isCover, isScreenshot)

	*foundImages = append(*foundImages, candidate)
}

// isImageURL checks if a URL points to an image
func (s *Service) isImageURL(url string) bool {
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
	urlLower := strings.ToLower(url)

	for _, ext := range imageExtensions {
		if strings.Contains(urlLower, ext) {
			return true
		}
	}

	return false
}

// shouldSkipImage determines if an image should be skipped based on common patterns
func (s *Service) shouldSkipImage(imgURL, alt, class string) bool {
	skipPatterns := []string{
		"avatar", "icon", "emoji", "smilie", "data:image",
	}

	imgURLLower := strings.ToLower(imgURL)
	altLower := strings.ToLower(alt)
	classLower := strings.ToLower(class)

	for _, pattern := range skipPatterns {
		if strings.Contains(imgURLLower, pattern) ||
			strings.Contains(altLower, pattern) ||
			strings.Contains(classLower, pattern) {
			return true
		}
	}

	return false
}

// parseImageDimensions extracts width and height from image attributes
func (s *Service) parseImageDimensions(img *colly.HTMLElement) (width, height int) {
	if w := img.Attr("width"); w != "" {
		fmt.Sscanf(w, "%d", &width)
	}
	if h := img.Attr("height"); h != "" {
		fmt.Sscanf(h, "%d", &height)
	}
	return width, height
}

// calculateImageScore calculates a quality score for an image candidate
func (s *Service) calculateImageScore(imgURL, alt, class, context string, isLightbox, isCover, isScreenshot bool, width, height int) float64 {
	score := 0.0

	// Base score by context (where the image was found)
	switch context {
	case "thread-starter-lightbox":
		score += 100.0 // Highest priority for thread starter lightbox
	case "thread-starter-bb-zoom":
		score += 90.0
	case "thread-starter-wrapped":
		score += 80.0
	case "thread-starter-bb":
		score += 70.0
	case "thread-starter-content":
		score += 60.0
	case "generic":
		score += 30.0
	case "fallback":
		score += 10.0
	}

	// Bonus for lightbox images
	if isLightbox {
		score += 50.0
	}

	// Bonus for cover images
	if isCover {
		score += 40.0
	}

	// Heavy penalty for screenshots
	if isScreenshot {
		score -= 80.0
	}

	// Size bonus (larger is generally better for cover images)
	if width > 0 && height > 0 {
		area := width * height
		if area > 100000 { // Large image
			score += 20.0
		} else if area > 50000 { // Medium image
			score += 10.0
		} else if area < 10000 { // Small image penalty
			score -= 10.0
		}
	}

	// Bonus for images that appear to be covers based on filename
	imgURLLower := strings.ToLower(imgURL)
	if strings.Contains(imgURLLower, "cover") || strings.Contains(imgURLLower, "banner") {
		score += 30.0
	}

	// Penalty for thumbnails
	if strings.Contains(imgURLLower, "thumb") || strings.Contains(imgURLLower, "small") {
		score -= 20.0
	}

	return score
}

// selectBestImageCandidate selects the best image from candidates
func (s *Service) selectBestImageCandidate(candidates []ImageCandidate, pageTitle string) *ImageCandidate {
	if len(candidates) == 0 {
		return nil
	}

	fmt.Printf("DEBUG: Selecting best image from %d candidates\n", len(candidates))

	// Sort candidates by score (highest first)
	bestCandidate := &candidates[0]
	for i := 1; i < len(candidates); i++ {
		if candidates[i].Score > bestCandidate.Score {
			bestCandidate = &candidates[i]
		}
	}

	// Additional filtering: skip screenshots even if they have high scores
	for _, candidate := range candidates {
		if !candidate.IsScreenshot && candidate.Score > 50.0 {
			fmt.Printf("DEBUG: Selected non-screenshot candidate with score %.2f over screenshot with score %.2f\n",
				candidate.Score, bestCandidate.Score)
			return &candidate
		}
	}

	return bestCandidate
}

// sortCandidatesByScore sorts image candidates by score in descending order
func (s *Service) sortCandidatesByScore(candidates []ImageCandidate) []ImageCandidate {
	// Create a copy to avoid modifying the original slice
	sorted := make([]ImageCandidate, len(candidates))
	copy(sorted, candidates)

	// Simple bubble sort by score (descending)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Score < sorted[j+1].Score {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// convertThumbnailToFullSize converts thumbnail URLs to full-size image URLs
func (s *Service) convertThumbnailToFullSize(imgURL string) string {
	// F95Zone thumbnail pattern: .../thumb/filename -> .../filename
	if strings.Contains(imgURL, "/thumb/") {
		fullSizeURL := strings.Replace(imgURL, "/thumb/", "/", 1)
		fmt.Printf("DEBUG: Converting thumbnail URL: %s -> %s\n", imgURL, fullSizeURL)
		return fullSizeURL
	}
	return imgURL
}

// findBestImageFromPage finds the most suitable image for a game from a webpage
func (s *Service) findBestImageFromPage(doc *goquery.Document, sourceURL string) string {
	// Priority selectors for different types of sites
	var selectors []string

	if strings.Contains(sourceURL, "f95zone.to") {
		// F95Zone specific selectors - only target thread starter post
		selectors = []string{
			".message-threadStarterPost .bbImage[data-zoom-target]",         // Thread starter lightbox images with zoom (highest priority)
			".message-threadStarterPost .lbContainer img[data-zoom-target]", // Thread starter lightbox container images
			".message-threadStarterPost .bbImageWrapper img",                // Thread starter wrapped images
			".message-threadStarterPost img[data-zoom-target]",              // Thread starter images with zoom
			".message-threadStarterPost .bbImage",                           // Thread starter forum images
			".message-threadStarterPost .message-userContent img",           // Thread starter content images
			".message-threadStarterPost img",                                // Any images in thread starter post
		}
	} else {
		// Generic selectors for other sites
		selectors = []string{
			"img[alt*='preview']", "img[alt*='Preview']",
			"img[alt*='cover']", "img[alt*='Cover']",
			"img[alt*='game']", "img[alt*='Game']",
			"img[class*='cover']", "img[class*='preview']",
			"img[class*='thumbnail']", "img[class*='hero']",
			".preview img", ".cover img", ".thumbnail img",
			"img", // Fallback to any image
		}
	}

	// Try each selector in priority order
	for _, selector := range selectors {
		var bestImageURL string

		// For F95Zone lightbox selectors, prioritize the first valid image
		isLightboxSelector := strings.Contains(selector, "data-url") ||
			strings.Contains(selector, "bbImageWrapper") ||
			strings.Contains(selector, "data-zoom-target")

		if isLightboxSelector && strings.Contains(sourceURL, "f95zone.to") {
			// For lightbox images, take the first valid one
			fmt.Printf("DEBUG: Checking lightbox selector: %s\n", selector)
			doc.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
				// Get image URL from data-url attribute first (lightbox), then src
				imgURL, exists := s.Attr("data-url")
				if !exists || imgURL == "" {
					// data-url is empty or doesn't exist, try src
					imgURL, exists = s.Attr("src")
					if !exists || imgURL == "" {
						// Try data-src as another fallback
						imgURL, exists = s.Attr("data-src")
					}
				}
				if !exists || imgURL == "" {
					return true // Continue to next image
				}

				// Debug output
				alt, _ := s.Attr("alt")
				class, _ := s.Attr("class")
				zoomTarget, _ := s.Attr("data-zoom-target")
				fmt.Printf("DEBUG: Found image %d: %s (alt=%s, class=%s, zoom-target=%s)\n",
					i, imgURL, alt, class, zoomTarget)

				// Skip small icons, avatars, and common unwanted images
				if s.HasClass("avatar") || s.HasClass("icon") ||
					strings.Contains(imgURL, "avatar") || strings.Contains(imgURL, "icon") ||
					strings.Contains(imgURL, "emoji") || strings.Contains(imgURL, "smilie") ||
					strings.Contains(imgURL, "data:image") { // Skip data URIs
					return true // Continue to next image
				}

				// For lightbox images, also skip very small images based on URL patterns
				if strings.Contains(imgURL, "thumb") || strings.Contains(imgURL, "small") {
					return true // Continue to next image
				}

				// Skip screenshot images - prefer cover/banner images
				if strings.Contains(strings.ToLower(imgURL), "screenshot") ||
					strings.Contains(strings.ToLower(alt), "screenshot") {
					return true // Continue to next image
				}

				bestImageURL = imgURL
				fmt.Printf("DEBUG: Selected lightbox image: %s\n", imgURL)
				return false // Break - we found our first lightbox image
			})
		} else {
			// For non-lightbox selectors, use the existing logic (largest image)
			largestSize := 0

			doc.Find(selector).Each(func(i int, s *goquery.Selection) {
				// Get image URL from src or data-url attribute
				imgURL, exists := s.Attr("src")
				if !exists || imgURL == "" {
					// Try data-url for F95Zone click-to-expand images
					imgURL, exists = s.Attr("data-url")
					if !exists || imgURL == "" {
						// Try data-src as another fallback
						imgURL, exists = s.Attr("data-src")
					}
				}
				if !exists || imgURL == "" {
					return
				}

				// Skip small icons, avatars, and common unwanted images
				if s.HasClass("avatar") || s.HasClass("icon") ||
					strings.Contains(imgURL, "avatar") || strings.Contains(imgURL, "icon") ||
					strings.Contains(imgURL, "emoji") || strings.Contains(imgURL, "smilie") ||
					strings.Contains(imgURL, "data:image") { // Skip data URIs
					return
				}

				// Skip screenshot images - prefer cover/banner images
				altAttr, _ := s.Attr("alt")
				if strings.Contains(strings.ToLower(imgURL), "screenshot") ||
					strings.Contains(strings.ToLower(altAttr), "screenshot") {
					return // Skip screenshot images
				}

				// Try to get image dimensions
				width := 0
				height := 0
				if w, exists := s.Attr("width"); exists {
					fmt.Sscanf(w, "%d", &width)
				}
				if h, exists := s.Attr("height"); exists {
					fmt.Sscanf(h, "%d", &height)
				}

				// Calculate image size (use area as a rough measure)
				size := width * height

				// If no dimensions available, prioritize by position (first images are often better)
				if size == 0 {
					size = 1000 - i // Give preference to earlier images
				}

				// Skip very small images (likely icons)
				if (width > 0 && width < 50) || (height > 0 && height < 50) {
					return
				}

				// Update best image if this one is larger
				if size > largestSize {
					largestSize = size
					bestImageURL = imgURL
				}
			})
		}

		// Return the first good image we find
		if bestImageURL != "" {
			fmt.Printf("DEBUG: Final selected image URL: %s\n", bestImageURL)
			return bestImageURL
		}
	}

	fmt.Printf("DEBUG: No suitable image found on page\n")
	return ""
}

// downloadImage downloads an image from URL and saves it locally
func (s *Service) downloadImage(imageURL string) (string, error) {
	if imageURL == "" {
		return "", nil
	}

	// Create a filename from the URL
	urlParts := strings.Split(imageURL, "/")
	if len(urlParts) == 0 {
		return "", fmt.Errorf("invalid image URL")
	}

	filename := urlParts[len(urlParts)-1]
	// Clean the filename to remove query parameters
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	// Add extension if missing
	if !strings.Contains(filename, ".") {
		filename += ".jpg"
	}

	// Create full path
	imagePath := filepath.Join(s.imageDir, filename)

	// Check if image already exists and is valid
	if _, err := os.Stat(imagePath); err == nil {
		// Validate existing file
		if err := s.validateImageFile(imagePath); err == nil {
			return imagePath, nil // Image already exists and is valid
		}
		// If existing file is invalid, remove it and re-download
		os.Remove(imagePath)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}

	// Download the image with proper headers
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://f95zone.to/")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image, status: %d", resp.StatusCode)
	}

	// Check content type to ensure it's an image
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("response is not an image, content-type: %s", contentType)
	}

	// Create the file
	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	// Validate that the downloaded file is actually an image (this will convert AVIF to PNG if needed)
	if err := s.validateImageFile(imagePath); err != nil {
		fmt.Printf("DEBUG: Downloaded file validation failed for %s: %v\n", imagePath, err)
		// Remove the invalid file and return error
		os.Remove(imagePath)
		return "", fmt.Errorf("downloaded file is not a valid image: %w", err)
	}

	return imagePath, nil
}

// getBaseImageURL extracts the base URL without extension for trying alternative formats
func (s *Service) getBaseImageURL(imageURL string) string {
	// Remove the file extension
	lastDot := strings.LastIndex(imageURL, ".")
	if lastDot == -1 {
		return imageURL
	}

	// Also check for query parameters
	lastQuestion := strings.LastIndex(imageURL, "?")
	if lastQuestion > lastDot {
		return imageURL[:lastDot]
	}

	return imageURL[:lastDot]
}

// downloadImageWithValidation downloads and validates an image in one step
func (s *Service) downloadImageWithValidation(imageURL string) (string, error) {
	// Create a filename from the URL
	urlParts := strings.Split(imageURL, "/")
	if len(urlParts) == 0 {
		return "", fmt.Errorf("invalid image URL")
	}

	filename := urlParts[len(urlParts)-1]
	// Clean the filename to remove query parameters
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	// Add extension if missing
	if !strings.Contains(filename, ".") {
		filename += ".jpg"
	}

	// Create full path
	imagePath := filepath.Join(s.imageDir, filename)

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}

	// Download the image with proper headers
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://f95zone.to/")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image, status: %d", resp.StatusCode)
	}

	// Check content type to ensure it's an image
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("response is not an image, content-type: %s", contentType)
	}

	// Create the file
	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	// Validate the image
	if err := s.validateImageFile(imagePath); err != nil {
		os.Remove(imagePath) // Clean up invalid file
		return "", fmt.Errorf("downloaded image validation failed: %w", err)
	}

	return imagePath, nil
}

// DownloadImageForResult downloads the image for a specific search result
func (s *Service) DownloadImageForResult(result *SearchResult) error {
	if result.ImageURL == "" {
		return nil // No image to download
	}

	imagePath, err := s.downloadImage(result.ImageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	result.ImagePath = imagePath
	return nil
}

// validateImageFile checks if a file is a valid image
func (s *Service) validateImageFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first few bytes to check image signature
	buffer := make([]byte, 12)
	_, err = file.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Check for common image formats
	if bytes.HasPrefix(buffer, []byte{0x89, 0x50, 0x4E, 0x47}) { // PNG
		return nil
	}
	if bytes.HasPrefix(buffer, []byte{0xFF, 0xD8, 0xFF}) { // JPEG
		return nil
	}
	if bytes.HasPrefix(buffer, []byte{0x47, 0x49, 0x46}) { // GIF
		return nil
	}
	if bytes.HasPrefix(buffer, []byte{0x42, 0x4D}) { // BMP
		return nil
	}
	if bytes.HasPrefix(buffer, []byte{0x52, 0x49, 0x46, 0x46}) { // WebP (RIFF)
		return nil
	}

	// Check for AVIF format (ftypavif) and convert to PNG
	if bytes.Contains(buffer, []byte("ftyp")) && bytes.Contains(buffer, []byte("avif")) {
		fmt.Printf("DEBUG: AVIF file detected, converting to PNG: %s\n", filePath)
		err := s.convertAVIFToPNG(filePath)
		if err != nil {
			return fmt.Errorf("failed to convert AVIF to PNG: %w", err)
		}
		return nil // Successfully converted
	}

	return fmt.Errorf("file is not a valid image format")
}

// convertAVIFToPNG converts an AVIF file to PNG format in place
func (s *Service) convertAVIFToPNG(filePath string) error {
	// Read the AVIF file
	avifFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open AVIF file: %w", err)
	}
	defer avifFile.Close()

	// Decode AVIF image
	var img image.Image
	img, err = avif.Decode(avifFile)
	if err != nil {
		return fmt.Errorf("failed to decode AVIF image: %w", err)
	}

	// Create new PNG file path (replace extension)
	pngPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".png"

	// Create PNG file
	pngFile, err := os.Create(pngPath)
	if err != nil {
		return fmt.Errorf("failed to create PNG file: %w", err)
	}
	defer pngFile.Close()

	// Encode as PNG
	err = png.Encode(pngFile, img)
	if err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	// Remove the original AVIF file
	err = os.Remove(filePath)
	if err != nil {
		fmt.Printf("DEBUG: Warning - could not remove original AVIF file: %v\n", err)
		// Don't return error here, we successfully created the PNG
	}

	// Rename PNG file to original path (if extensions differ)
	if pngPath != filePath {
		err = os.Rename(pngPath, filePath)
		if err != nil {
			return fmt.Errorf("failed to rename PNG file: %w", err)
		}
	}

	fmt.Printf("DEBUG: Successfully converted AVIF to PNG: %s\n", filePath)
	return nil
}

// testFyneImageSupport tests what image formats Fyne can load
func (s *Service) testFyneImageSupport() {
	testImages := []string{
		"test.png",
		"test.jpg",
		"test.jpeg",
		"test.gif",
		"test.bmp",
		"test.webp",
	}

	fmt.Println("DEBUG: Testing Fyne image format support...")

	for _, format := range testImages {
		// Create a simple test image in each format
		testPath := filepath.Join(s.imageDir, format)
		if err := s.createTestImage(testPath, format); err != nil {
			fmt.Printf("DEBUG: Failed to create test %s: %v\n", format, err)
		} else {
			fmt.Printf("DEBUG: Created test image: %s\n", testPath)
		}
	}
}

// createTestImage creates a simple test image in the specified format
func (s *Service) createTestImage(filePath, format string) error {
	// This is a placeholder - in a real implementation, you'd create actual test images
	// For now, we'll just create empty files to test the file loading mechanism
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write a simple test pattern based on format
	switch format {
	case "test.png":
		// Write PNG header
		file.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	case "test.jpg":
		// Write JPEG header
		file.Write([]byte{0xFF, 0xD8, 0xFF})
	case "test.gif":
		// Write GIF header
		file.Write([]byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61})
	case "test.bmp":
		// Write BMP header
		file.Write([]byte{0x42, 0x4D})
	case "test.webp":
		// Write WebP header (RIFF)
		file.Write([]byte{0x52, 0x49, 0x46, 0x46})
	}

	return nil
}
