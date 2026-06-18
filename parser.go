package main

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss"
	nethtml "golang.org/x/net/html"
)

// ArticleLink represents a link to another wiki page
type ArticleLink struct {
	Text   string
	Target string
}

type styleContext struct {
	bold    bool
	italic  bool
	inQuote bool
}

// extractText extracts all plain text recursively from a node
func extractText(n *nethtml.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == nethtml.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}

// CleanHTML converts raw wiki HTML into readable plaintext formatted for the terminal,
// and extracts all internal links using golang.org/x/net/html.
func CleanHTML(rawHTML string, themeColor lipgloss.Color) (string, []ArticleLink) {
	doc, err := nethtml.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return rawHTML, nil
	}

	var links []ArticleLink
	
	// Helper to track and deduplicate internal wiki links
	addLink := func(target, text string) int {
		target = strings.TrimSpace(target)
		target = strings.ReplaceAll(target, "_", " ")
		text = strings.TrimSpace(text)
		
		if text == "" || target == "" || strings.Contains(target, ":") {
			return -1
		}
		
		for i, l := range links {
			if l.Target == target {
				return i + 1
			}
		}
		links = append(links, ArticleLink{Text: text, Target: target})
		return len(links)
	}

	var builder strings.Builder

	// Style declarations
	linkStyle := lipgloss.NewStyle().Foreground(themeColor).Underline(true)
	indexStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	quoteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9399B2")).
		Italic(true).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(themeColor).
		PaddingLeft(2)

<<<<<<< HEAD
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
||||||| parent of 1e21417 (feat:native HTML parser, UX upgrade, History navigation added.)
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

	// 5. Stylize Headers with customized lipgloss rules and underlines
	h1Regex := regexp.MustCompile(`(?i)<h[1-3][^>]*>(.*?)</h[1-3]>`)
=======
>>>>>>> 1e21417 (feat:native HTML parser, UX upgrade, History navigation added.)
	h1Style := lipgloss.NewStyle().Bold(true).Foreground(themeColor)
	h4Style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4"))
	bulletStyle := lipgloss.NewStyle().Foreground(themeColor).Bold(true).SetString("• ")

	var walk func(n *nethtml.Node, ctx styleContext)
	walk = func(n *nethtml.Node, ctx styleContext) {
		if n == nil {
			return
		}

		if n.Type == nethtml.ElementNode {
			tagName := strings.ToLower(n.Data)

			// Discard scripts, styles, noscript, tables, and asides
			if tagName == "script" || tagName == "style" || tagName == "noscript" || tagName == "table" || tagName == "aside" {
				return
			}

			// Handle links (<a>)
			if tagName == "a" {
				var href string
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						href = attr.Val
						break
					}
				}

				// If it's an internal /wiki/ link
				u, err := url.Parse(href)
				if err == nil && strings.HasPrefix(u.Path, "/wiki/") {
					targetEncoded := strings.TrimPrefix(u.Path, "/wiki/")
					target, err := url.QueryUnescape(targetEncoded)
					if err != nil {
						target = targetEncoded
					}
					
					linkText := html.UnescapeString(extractText(n))
					linkText = strings.TrimSpace(linkText)
					
					idx := addLink(target, linkText)
					if idx != -1 {
						renderedText := linkStyle.Render(linkText)
						renderedIdx := indexStyle.Render(fmt.Sprintf("[%d]", idx))
						builder.WriteString(renderedText + renderedIdx)
						return // Skip visiting children since we rendered the link fully
					}
				}
			}

			// Handle Headers
			if tagName == "h1" || tagName == "h2" || tagName == "h3" {
				headerText := html.UnescapeString(extractText(n))
				headerText = strings.TrimSpace(headerText)
				if headerText != "" {
					rendered := h1Style.Render(strings.ToUpper(headerText))
					borderLine := lipgloss.NewStyle().Foreground(themeColor).Render(strings.Repeat("━", len(headerText)))
					builder.WriteString("\n\n" + rendered + "\n" + borderLine + "\n")
				}
				return // Skip visiting children
			}

			if tagName == "h4" || tagName == "h5" || tagName == "h6" {
				headerText := html.UnescapeString(extractText(n))
				headerText = strings.TrimSpace(headerText)
				if headerText != "" {
					rendered := h4Style.Render(headerText)
					borderLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475A")).Render(strings.Repeat("─", len(headerText)))
					builder.WriteString("\n\n" + rendered + "\n" + borderLine + "\n")
				}
				return // Skip visiting children
			}

			// Handle blockquote
			if tagName == "blockquote" {
				quoteText := html.UnescapeString(extractText(n))
				quoteText = strings.TrimSpace(quoteText)
				if quoteText != "" {
					builder.WriteString("\n\n" + quoteStyle.Render(quoteText) + "\n\n")
				}
				return // Skip visiting children
			}

			// Adjust context for inline formatting
			switch tagName {
			case "strong", "b":
				ctx.bold = true
			case "em", "i":
				ctx.italic = true
			case "li":
				builder.WriteString("\n" + bulletStyle.Render())
			case "br":
				builder.WriteString("\n")
			case "p", "div":
				builder.WriteString("\n")
			}
		}

		if n.Type == nethtml.TextNode {
			text := n.Data
			
			// Simple space normalization for text blocks (but preserve line breaks in structure)
			text = strings.ReplaceAll(text, "\t", " ")
			text = strings.ReplaceAll(text, "\n", " ")
			
			if text != "" {
				styled := text
				if ctx.bold || ctx.italic {
					s := lipgloss.NewStyle()
					if ctx.bold {
						s = s.Bold(true)
					}
					if ctx.italic {
						s = s.Italic(true)
					}
					styled = s.Render(text)
				}
				builder.WriteString(styled)
			}
		}

		// Recurse children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, ctx)
		}

		// Block closures
		if n.Type == nethtml.ElementNode {
			tagName := strings.ToLower(n.Data)
			if tagName == "p" || tagName == "div" {
				builder.WriteString("\n")
			}
		}
	}

	walk(doc, styleContext{})

	// Post-processing line-trimming logic
	lines := strings.Split(builder.String(), "\n")
	var cleanedLines []string
	for _, line := range lines {
		// Collapse multiple spaces but keep general formatting intact
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(cleanedLines) > 0 && cleanedLines[len(cleanedLines)-1] != "" {
				cleanedLines = append(cleanedLines, "")
			}
		} else {
			// Replace multiple consecutive spaces with a single space
			spaceRegex := strings.Fields(line)
			reconstructed := strings.Join(spaceRegex, " ")
			// If it was a list item or started with a bullet point, let's keep indentation/bullet spacing
			if strings.HasPrefix(trimmed, "•") {
				// Bullet points should keep their indent
				cleanedLines = append(cleanedLines, line)
			} else {
				cleanedLines = append(cleanedLines, reconstructed)
			}
		}
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n")), links
}
