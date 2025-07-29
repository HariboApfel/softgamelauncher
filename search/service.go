package search

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
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
}

// Service handles game searching functionality
type Service struct {
	baseURL    string
	httpClient *http.Client
}

// NewService creates a new search service
func NewService() *Service {
	return &Service{
		baseURL: "https://f95zone.to/sam/latest_alpha/latest_data.php",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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
			results = append(results, SearchResult{
				Title:       item.Title,
				Link:        item.Link,
				Description: item.Description,
				PubDate:     item.PubDate,
				Category:    item.Category,
				MatchScore:  matchScore,
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
			results = append(results, SearchResult{
				Title:       item.Title,
				Link:        item.Link,
				Description: item.Description,
				PubDate:     item.PubDate,
				Category:    item.Category,
				MatchScore:  matchScore,
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
