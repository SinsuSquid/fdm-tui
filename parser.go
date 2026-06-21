package main

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ArticleLink represents a link to another wiki page
type ArticleLink struct {
	Text   string
	Target string
}

var (
	// Tags to discard entirely along with their content
	tagsToDiscard = []string{"script", "style", "noscript", "table", "aside"}

	// Regexp to extract internal links
	linkRegex = regexp.MustCompile(`(?i)<a[^>]*href="/wiki/([^"?#]+)"[^>]*>(.*?)</a>`)
)

// stripHTML is a robust state-based HTML tag stripper.
// It safely handles quotes, long attributes, and embedded '>' characters.
func stripHTML(s string) string {
	var builder strings.Builder
	inTag := false
	inQuote := false
	var quoteChar rune

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if inTag {
			if inQuote {
				if r == quoteChar {
					inQuote = false
				}
			} else {
				if r == '"' || r == '\'' {
					inQuote = true
					quoteChar = r
				} else if r == '>' {
					inTag = false
				}
			}
		} else {
			if r == '<' {
				inTag = true
				inQuote = false
			} else {
				builder.WriteRune(r)
			}
		}
	}
	return builder.String()
}

// GetHintForIndex converts a 1-based index into a Vimium-style letter hint.
func GetHintForIndex(index int) string {
	chars := "asdfghjklqwertyuiopzxcvbnm"
	if index <= len(chars) {
		return string(chars[index-1])
	}
	first := (index - 1) / len(chars)
	second := (index - 1) % len(chars)
	if first <= len(chars) {
		return string(chars[first-1]) + string(chars[second])
	}
	return fmt.Sprintf("%d", index)
}

// WikiHeader represents a parsed header section in the article.
type WikiHeader struct {
	Text  string
	Level int
}

func extractHeaders(rawHTML string) []WikiHeader {
	var headers []WikiHeader
	headerRegex := regexp.MustCompile(`(?i)<(h[1-6])[^>]*>(.*?)</h[1-6]>`)
	matches := headerRegex.FindAllStringSubmatch(rawHTML, -1)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		tag := strings.ToLower(m[1])
		level := int(tag[1] - '0')
		text := stripHTML(m[2])
		text = html.UnescapeString(strings.TrimSpace(text))
		if text == "" || len(text) > 100 {
			continue
		}
		headers = append(headers, WikiHeader{
			Text:  text,
			Level: level,
		})
	}
	return headers
}

// CleanHTML converts raw wiki HTML into readable plaintext formatted for the terminal,
// extracts all internal links, and extracts header outlines.
func CleanHTML(rawHTML string, themeColor lipgloss.Color) (string, []ArticleLink, []WikiHeader) {
	// Extract outline headers before modifying HTML
	headers := extractHeaders(rawHTML)

	// 1. Remove comments
	commentRegex := regexp.MustCompile(`(?s)`)
	h := commentRegex.ReplaceAllString(rawHTML, "")

	// 2. Remove discarded tags and their contents
	for _, tag := range tagsToDiscard {
		discardRegex := regexp.MustCompile("(?si)<" + tag + "[^>]*>.*?</" + tag + ">")
		h = discardRegex.ReplaceAllString(h, "")
	}

	// 3. Extract and style links before stripping tags
	var links []ArticleLink
	linkStyle := lipgloss.NewStyle().Foreground(themeColor).Underline(true)
	
	h = linkRegex.ReplaceAllStringFunc(h, func(match string) string {
		submatches := linkRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		
		targetEncoded := submatches[1]
		linkTextRaw := submatches[2]
		
		target, err := url.QueryUnescape(targetEncoded)
		if err != nil {
			target = targetEncoded
		}
		target = strings.ReplaceAll(target, "_", " ")
		
		linkText := stripHTML(linkTextRaw)
		linkText = html.UnescapeString(linkText)
		linkText = strings.TrimSpace(linkText)
		
		if linkText == "" || target == "" || strings.Contains(target, ":") {
			return linkTextRaw
		}
		
		// Deduplicate and get index
		linkIndex := -1
		for i, l := range links {
			if l.Target == target {
				linkIndex = i + 1
				break
			}
		}
		
		if linkIndex == -1 {
			links = append(links, ArticleLink{Text: linkText, Target: target})
			linkIndex = len(links)
		}
		
		hint := GetHintForIndex(linkIndex)
		indexStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).SetString(fmt.Sprintf("[%s]", hint))
		return fmt.Sprintf("%s%s", linkStyle.Render(linkText), indexStyle.Render())
	})

	// 4. Style specific formatted tags inside the content (Bold, Italics, Blockquotes)
	
	// Bold tags <b> / <strong>
	boldRegex := regexp.MustCompile(`(?i)<(?:strong|b)[^>]*>(.*?)</(?:strong|b)>`)
	boldStyle := lipgloss.NewStyle().Bold(true)
	h = boldRegex.ReplaceAllStringFunc(h, func(match string) string {
		sub := boldRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		cleanText := stripHTML(sub[1])
		return boldStyle.Render(html.UnescapeString(cleanText))
	})

	// Italic tags <i> / <em>
	italicRegex := regexp.MustCompile(`(?i)<(?:em|i)[^>]*>(.*?)</(?:em|i)>`)
	italicStyle := lipgloss.NewStyle().Italic(true)
	h = italicRegex.ReplaceAllStringFunc(h, func(match string) string {
		sub := italicRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		cleanText := stripHTML(sub[1])
		return italicStyle.Render(html.UnescapeString(cleanText))
	})

	// Blockquotes <blockquote>
	quoteRegex := regexp.MustCompile(`(?i)<blockquote[^>]*>(.*?)</blockquote>`)
	h = quoteRegex.ReplaceAllStringFunc(h, func(match string) string {
		sub := quoteRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		cleanText := stripHTML(sub[1])
		cleanText = html.UnescapeString(strings.TrimSpace(cleanText))
		
		quoteStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9399B2")).
			Italic(true).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(themeColor).
			PaddingLeft(2)
			
		return fmt.Sprintf("\n\n%s\n\n", quoteStyle.Render(cleanText))
	})

	// Citation/reference tags <sup> (like [1])
	supRegex := regexp.MustCompile(`(?i)<sup[^>]*>(.*?)</sup>`)
	supStyle := lipgloss.NewStyle().Foreground(themeColor)
	h = supRegex.ReplaceAllStringFunc(h, func(match string) string {
		sub := supRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		cleanText := stripHTML(sub[1])
		return supStyle.Render(html.UnescapeString(cleanText))
	})

	// 5. Stylize Headers with customized lipgloss rules and underlines
	h1Regex := regexp.MustCompile(`(?i)<h[1-3][^>]*>(.*?)</h[1-3]>`)
	h1Style := lipgloss.NewStyle().Bold(true).Foreground(themeColor)
	h = h1Regex.ReplaceAllStringFunc(h, func(match string) string {
		sub := h1Regex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		cleanText := stripHTML(sub[1])
		cleanText = html.UnescapeString(strings.TrimSpace(cleanText))
		borderLine := strings.Repeat("━", len(cleanText))
		borderStyle := lipgloss.NewStyle().Foreground(themeColor)
		return fmt.Sprintf("\n\n%s\n%s\n", h1Style.Render(strings.ToUpper(cleanText)), borderStyle.Render(borderLine))
	})

	h4Regex := regexp.MustCompile(`(?i)<h[4-6][^>]*>(.*?)</h[4-6]>`)
	h4Style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4"))
	h = h4Regex.ReplaceAllStringFunc(h, func(match string) string {
		sub := h4Regex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		cleanText := stripHTML(sub[1])
		cleanText = html.UnescapeString(strings.TrimSpace(cleanText))
		borderLine := strings.Repeat("─", len(cleanText))
		borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475A"))
		return fmt.Sprintf("\n\n%s\n%s\n", h4Style.Render(cleanText), borderStyle.Render(borderLine))
	})

	// 6. Style bullet points
	bulletStyle := lipgloss.NewStyle().Foreground(themeColor).Bold(true).SetString("• ")
	h = regexp.MustCompile(`(?i)<li[^>]*>`).ReplaceAllString(h, "\n"+bulletStyle.Render())
	h = regexp.MustCompile(`(?i)</li>`).ReplaceAllString(h, "")

	// 7. Strip out all remaining HTML tags using state-based stripper
	h = stripHTML(h)

	// 8. Decode HTML entities
	h = html.UnescapeString(h)

	// 9. Clean up multiple newlines and spaces
	lines := strings.Split(h, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(cleanedLines) > 0 && cleanedLines[len(cleanedLines)-1] != "" {
				cleanedLines = append(cleanedLines, "")
			}
		} else {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n")), links, headers
}
