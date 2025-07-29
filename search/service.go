package search

import (
	"bytes"
	"encoding/xml"
	"fmt"
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
			// Extract image URL from description (but don't download yet)
			imageURL := s.ExtractImageURL(item.Description)

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
			// Extract image URL from description (but don't download yet)
			imageURL := s.ExtractImageURL(item.Description)

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

	// Validate that the downloaded file is actually an image
	if err := s.validateImageFile(imagePath); err != nil {
		// For now, just log the error but don't delete the file
		// This will help us debug what's being downloaded
		fmt.Printf("DEBUG: Downloaded file validation failed for %s: %v\n", imagePath, err)
		// Let's examine what was actually downloaded
		if content, readErr := os.ReadFile(imagePath); readErr == nil {
			length := 200
			if len(content) < length {
				length = len(content)
			}
			fmt.Printf("DEBUG: Downloaded file content (first %d bytes): %s\n", length, string(content[:length]))
		}
		// Don't delete the file yet - let's see what it contains
		// os.Remove(imagePath)
		// return "", fmt.Errorf("downloaded file is not a valid image: %w", err)
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
	buffer := make([]byte, 8)
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

	return fmt.Errorf("file is not a valid image format")
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
