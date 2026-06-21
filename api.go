package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	_ "golang.org/x/image/webp"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"github.com/mattn/go-sixel"
)

// SearchResult represents a single search result from Fandom/MediaWiki
type SearchResult struct {
	Title   string `json:"title"`
	PageID  int    `json:"pageid"`
	Snippet string `json:"snippet"`
}

type searchResponse struct {
	Query struct {
		Search []SearchResult `json:"search"`
	} `json:"query"`
}

type parseResponse struct {
	Parse struct {
		Title string `json:"title"`
		Text  struct {
			HTML string `json:"*"`
		} `json:"text"`
	} `json:"parse"`
}

type siteinfoResponse struct {
	Query struct {
		General struct {
			Logo     string `json:"logo"`
			MainPage string `json:"mainpage"`
		} `json:"general"`
	} `json:"query"`
}

// SearchArticles queries the specified fandom wiki for articles matching the query.
func SearchArticles(wiki string, query string) ([]SearchResult, error) {
	apiURL := fmt.Sprintf("https://%s.fandom.com/api.php", wiki)
	params := url.Values{}
	params.Add("action", "query")
	params.Add("list", "search")
	params.Add("srsearch", query)
	params.Add("format", "json")

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var searchResp searchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return searchResp.Query.Search, nil
}

// FetchArticleContent retrieves the parsed HTML content of a wiki page.
func FetchArticleContent(wiki string, title string) (string, error) {
	apiURL := fmt.Sprintf("https://%s.fandom.com/api.php", wiki)
	params := url.Values{}
	params.Add("action", "parse")
	params.Add("page", title)
	params.Add("prop", "text")
	params.Add("format", "json")
	params.Add("redirects", "true")

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("failed to fetch article: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var parseResp parseResponse
	if err := json.Unmarshal(body, &parseResp); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return parseResp.Parse.Text.HTML, nil
}

// FetchWikiSiteDetails retrieves the logo and main page title from siteinfo
func FetchWikiSiteDetails(wiki string) (string, string, error) {
	apiURL := fmt.Sprintf("https://%s.fandom.com/api.php", wiki)
	params := url.Values{}
	params.Add("action", "query")
	params.Add("meta", "siteinfo")
	params.Add("siprop", "general")
	params.Add("format", "json")

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("http error: %d", resp.StatusCode)
	}

	var info siteinfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", "", err
	}

	return info.Query.General.Logo, info.Query.General.MainPage, nil
}

// FetchDominantColor downloads the logo and calculates the dominant color.
func FetchDominantColor(logoURL string) (string, error) {
	resp, err := http.Get(logoURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download logo: %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	colorCounts := make(map[string]int)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			r, g, b, a := img.At(x, y).RGBA()
			r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)

			if a8 < 150 {
				continue
			}

			if r8 < 20 && g8 < 20 && b8 < 20 {
				continue
			}
			if r8 > 230 && g8 > 230 && b8 > 230 {
				continue
			}

			maxVal := math.Max(float64(r8), math.Max(float64(g8), float64(b8)))
			minVal := math.Min(float64(r8), math.Min(float64(g8), float64(b8)))
			if (maxVal - minVal) < 15 {
				continue
			}

			rBin := (r8 / 16) * 16
			gBin := (g8 / 16) * 16
			bBin := (b8 / 16) * 16

			hexKey := fmt.Sprintf("#%02X%02X%02X", rBin, gBin, bBin)
			colorCounts[hexKey]++
		}
	}

	maxCount := 0
	dominantHex := ""
	for hex, count := range colorCounts {
		if count > maxCount {
			maxCount = count
			dominantHex = hex
		}
	}

	if dominantHex == "" {
		return "", fmt.Errorf("no colorful pixels found in logo")
	}

	return dominantHex, nil
}

func resizeImage(img image.Image, width, height int) image.Image {
	minX := img.Bounds().Min.X
	minY := img.Bounds().Min.Y
	maxX := img.Bounds().Max.X
	maxY := img.Bounds().Max.Y
	
	oldWidth := maxX - minX
	oldHeight := maxY - minY
	
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			origX := minX + (x * oldWidth / width)
			origY := minY + (y * oldHeight / height)
			newImg.Set(x, y, img.At(origX, origY))
		}
	}
	return newImg
}

// FetchSixelLogo downloads the logo and encodes it to Sixel format, padded with newlines to reserve TUI layout rows.
func FetchSixelLogo(logoURL string) (string, error) {
	resp, err := http.Get(logoURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download logo: %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("invalid image dimensions")
	}

	// Fit inside sidebar pane (typically 30 chars wide, let's use 24 chars max)
	// Assume 1 cell = 8x16 pixels
	charWidth := 24
	pixelWidth := charWidth * 8
	pixelHeight := (h * pixelWidth) / w
	
	// Limit height so it doesn't occupy too much space
	if pixelHeight > 96 {
		pixelHeight = 96
		pixelWidth = (w * pixelHeight) / h
	}

	resized := resizeImage(img, pixelWidth, pixelHeight)

	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	err = enc.Encode(resized)
	if err != nil {
		return "", err
	}

	linesNeeded := (pixelHeight + 15) / 16
	return buf.String() + strings.Repeat("\n", linesNeeded), nil
}

// FetchSixelImage downloads an image and encodes it to Sixel format fit to max width/height characters.
func FetchSixelImage(imageURL string, maxCharWidth, maxCharHeight int) (string, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("invalid image dimensions")
	}

	// Calculate target size
	pixelWidth := maxCharWidth * 8
	pixelHeight := (h * pixelWidth) / w
	
	maxPixelHeight := maxCharHeight * 16
	if pixelHeight > maxPixelHeight {
		pixelHeight = maxPixelHeight
		pixelWidth = (w * pixelHeight) / h
	}

	resized := resizeImage(img, pixelWidth, pixelHeight)

	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	err = enc.Encode(resized)
	if err != nil {
		return "", err
	}

	linesNeeded := (pixelHeight + 15) / 16
	return buf.String() + strings.Repeat("\n", linesNeeded), nil
}
