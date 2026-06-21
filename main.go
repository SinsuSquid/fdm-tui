package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appState int

const (
	stateWelcome appState = iota
	stateDashboard
)

type focusState int

const (
	focusSearch focusState = iota
	focusList
	focusOutline
	focusReader
	focusFollow
	focusGallery
)

type sidebarTabState int

const (
	tabSearch sidebarTabState = iota
	tabOutline
)

// Msg types for async operations
type searchResultMsg []SearchResult
type errMsg error
type articleContentMsg struct {
	Text    string
	Links   []ArticleLink
	Headers []WikiHeader
	Images  []ArticleImage
}
type wikiLandingMsg struct {
	ThemeColor lipgloss.Color
	Text       string
	Links      []ArticleLink
	Headers    []WikiHeader
	Title      string
	LogoSixel  string
	Images     []ArticleImage
}

func searchArticlesCmd(wiki, query string) tea.Cmd {
	return func() tea.Msg {
		results, err := SearchArticles(wiki, query)
		if err != nil {
			return errMsg(err)
		}
		return searchResultMsg(results)
	}
}

func fetchArticleCmd(wiki, title string, themeColor lipgloss.Color) tea.Cmd {
	return func() tea.Msg {
		html, err := FetchArticleContent(wiki, title)
		if err != nil {
			return errMsg(err)
		}
		cleaned, links, headers, images := CleanHTML(html, themeColor)
		return articleContentMsg{Text: cleaned, Links: links, Headers: headers, Images: images}
	}
}

func fetchWikiLandingCmd(wiki string) tea.Cmd {
	return func() tea.Msg {
		logo, mainPage, err := FetchWikiSiteDetails(wiki)
		var themeColor lipgloss.Color
		var logoSixel string
		if err != nil {
			themeColor = getAccentColor(wiki)
			mainPage = "Main Page"
		} else {
			hex, err := FetchDominantColor(logo)
			if err != nil {
				themeColor = getAccentColor(wiki)
			} else {
				themeColor = lipgloss.Color(hex)
			}
			// Fetch Sixel logo
			logoSixel, _ = FetchSixelLogo(logo)
		}

		html, err := FetchArticleContent(wiki, mainPage)
		if err != nil {
			return wikiLandingMsg{
				ThemeColor: themeColor,
				Text:       fmt.Sprintf("Welcome to the %s wiki!\n\nUse the sidebar search input (🔍) to find articles.", wiki),
				Links:      nil,
				Headers:    nil,
				Title:      "Main Page",
				LogoSixel:  logoSixel,
				Images:     nil,
			}
		}

		cleaned, links, headers, images := CleanHTML(html, themeColor)
		return wikiLandingMsg{
			ThemeColor: themeColor,
			Text:       cleaned,
			Links:      links,
			Headers:    headers,
			Title:      mainPage,
			LogoSixel:  logoSixel,
			Images:     images,
		}
	}
}

type model struct {
	state         appState
	focus         focusState
	wiki          string
	wikiInput     textinput.Model
	searchInput   textinput.Model
	followInput   textinput.Model
	viewport      viewport.Model
	loading       bool
	err           error
	width         int
	height        int
	themeColor    lipgloss.Color
	hideSidebar   bool
	
	// Search state
	searchResults []SearchResult
	cursor        int // selected search result
	
	// Article viewer state
	articleRawText string // Holds the unwrapped parsed text for dynamic re-wrapping
	articleLinks   []ArticleLink
	inReaderMode   bool

	// Headers / Outline state
	headers       []WikiHeader
	outlineCursor int
	sidebarTab    sidebarTabState

	// History stack
	history      []string
	currentTitle string

	// Sixel logo string & line count
	logoSixel string
	logoLines int

	// Gallery viewer state
	articleImages     []ArticleImage
	imageIndex        int
	currentImageSixel string
	imageLoading      bool
}

type imageSixelMsg string

func fetchSixelImageCmd(imageURL string, width, height int) tea.Cmd {
	return func() tea.Msg {
		sixelStr, err := FetchSixelImage(imageURL, width, height)
		if err != nil {
			return imageSixelMsg("")
		}
		return imageSixelMsg(sixelStr)
	}
}

func initialModel() model {
	wi := textinput.New()
	wi.Placeholder = "genshin-impact"
	wi.Prompt = " 🌐 wiki: "
	wi.CharLimit = 50
	wi.Width = 30
	wi.Focus()

	si := textinput.New()
	si.Placeholder = "Search articles..."
	si.Prompt = " 🔍 "
	si.CharLimit = 100
	si.Width = 20

	fi := textinput.New()
	fi.Placeholder = "asdf"
	fi.Prompt = " Follow link (letters): "
	fi.CharLimit = 5
	fi.Width = 12

	vp := viewport.New(80, 20)

	return model{
		state:       stateWelcome,
		wiki:        "genshin-impact",
		wikiInput:   wi,
		searchInput: si,
		followInput: fi,
		viewport:    vp,
		focus:       focusSearch,
		themeColor:  lipgloss.Color("#A855F7"),
		hideSidebar: false,
		sidebarTab:  tabSearch,
	}
}

func getAccentColor(wiki string) lipgloss.Color {
	name := strings.ToLower(strings.TrimSpace(wiki))
	if name == "" {
		return lipgloss.Color("#A855F7")
	}

	switch {
	case strings.Contains(name, "genshin"):
		return lipgloss.Color("#89B4FA")
	case strings.Contains(name, "elden") || strings.Contains(name, "ring") || strings.Contains(name, "ds") || strings.Contains(name, "souls"):
		return lipgloss.Color("#F9E2AF")
	case strings.Contains(name, "mine") || strings.Contains(name, "craft") || strings.Contains(name, "terraria"):
		return lipgloss.Color("#A6E3A1")
	case strings.Contains(name, "anime") || strings.Contains(name, "manga") || strings.Contains(name, "fandom"):
		return lipgloss.Color("#F5C2E7")
	case strings.Contains(name, "cyber") || strings.Contains(name, "punk") || strings.Contains(name, "hacker"):
		return lipgloss.Color("#FFE082")
	case strings.Contains(name, "starwars") || strings.Contains(name, "star-wars") || strings.Contains(name, "sith") || strings.Contains(name, "jedi"):
		return lipgloss.Color("#F38BA8")
	case strings.Contains(name, "wow") || strings.Contains(name, "warcraft") || strings.Contains(name, "diablo"):
		return lipgloss.Color("#E06C75")
	case strings.Contains(name, "wiki") || strings.Contains(name, "meta"):
		return lipgloss.Color("#94E2D5")
	default:
		hash := 0
		for _, char := range name {
			hash = int(char) + (hash << 5) - hash
		}
		colors := []string{
			"#A855F7",
			"#F38BA8",
			"#89B4FA",
			"#A6E3A1",
			"#F9E2AF",
			"#CBA6F7",
			"#FAB387",
			"#94E2D5",
		}
		index := idx(hash, len(colors))
		return lipgloss.Color(colors[index])
	}
}

func idx(val, limit int) int {
	if val < 0 {
		val = -val
	}
	return val % limit
}

// truncate safe-cuts a string at maxLen runes, adding an ellipsis if exceeded
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		if maxLen > 3 {
			return string(runes[:maxLen-3]) + "..."
		}
		return string(runes[:maxLen])
	}
	return s
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if m.state == stateDashboard {
			if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}

			if msg.Type == tea.MouseRelease && msg.Button == tea.MouseButtonLeft {
				if !m.hideSidebar && msg.X >= 1 && msg.X <= 31 {
					if m.sidebarTab == tabSearch {
						// Search results list starts at terminal line 13 + logo lines
						clickedIndex := msg.Y - (13 + m.logoLines)
						if clickedIndex >= 0 && clickedIndex < len(m.searchResults) && !m.loading {
							m.cursor = clickedIndex
							m.focus = focusList
							m.loading = true
							m.err = nil
							selected := m.searchResults[m.cursor].Title
							if m.currentTitle != "" {
								m.history = append(m.history, m.currentTitle)
							}
							m.currentTitle = selected
							cmds = append(cmds, fetchArticleCmd(m.wiki, selected, m.themeColor))
						}
					} else {
						// Outline list starts at terminal line 11 + logo lines
						clickedIndex := msg.Y - (11 + m.logoLines)
						if clickedIndex >= 0 && clickedIndex < len(m.headers) {
							m.outlineCursor = clickedIndex
							m.focus = focusOutline
							
							headerText := m.headers[m.outlineCursor].Text
							wrappedWidth := m.viewport.Width
							if wrappedWidth < 10 {
								wrappedWidth = 10
							}
							wrapped := lipgloss.NewStyle().Width(wrappedWidth).Render(m.articleRawText)
							lines := strings.Split(wrapped, "\n")
							var ansiRegex = regexp.MustCompile(`\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
							for i, line := range lines {
								cleanLine := ansiRegex.ReplaceAllString(line, "")
								if strings.Contains(strings.ToLower(cleanLine), strings.ToLower(headerText)) {
									m.viewport.YOffset = i
									break
								}
							}
							m.focus = focusReader
						}
					}
				}
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+b":
			if m.state == stateDashboard {
				m.hideSidebar = !m.hideSidebar
				
				var rightWidth int
				if m.hideSidebar {
					rightWidth = m.width - 2
				} else {
					rightWidth = m.width - 33
				}
				if rightWidth < 10 {
					rightWidth = 10
				}
				m.viewport.Width = rightWidth
				
				if m.inReaderMode {
					wrapped := lipgloss.NewStyle().Width(rightWidth).Render(m.articleRawText)
					m.viewport.SetContent(wrapped)
				}

				if m.hideSidebar {
					m.searchInput.Blur()
					m.focus = focusReader
				}
				return m, nil
			}
		case "ctrl+t":
			if m.state == stateDashboard && !m.hideSidebar {
				if m.sidebarTab == tabSearch {
					m.sidebarTab = tabOutline
					m.searchInput.Blur()
					if len(m.headers) > 0 {
						m.focus = focusOutline
					} else {
						m.focus = focusReader
					}
				} else {
					m.sidebarTab = tabSearch
					m.focus = focusSearch
					m.searchInput.Focus()
				}
				return m, nil
			}
		case "tab":
			if m.state == stateDashboard {
				if m.hideSidebar {
					m.focus = focusReader
					return m, nil
				}
				
				switch m.focus {
				case focusSearch:
					m.searchInput.Blur()
					if m.sidebarTab == tabOutline && len(m.headers) > 0 {
						m.focus = focusOutline
					} else if m.sidebarTab == tabSearch && len(m.searchResults) > 0 {
						m.focus = focusList
					} else if m.inReaderMode {
						m.focus = focusReader
					} else {
						m.focus = focusSearch
						m.searchInput.Focus()
					}
				case focusList:
					if m.inReaderMode {
						m.focus = focusReader
					} else {
						m.focus = focusSearch
						m.searchInput.Focus()
					}
				case focusOutline:
					if m.inReaderMode {
						m.focus = focusReader
					} else {
						m.focus = focusSearch
						m.searchInput.Focus()
					}
				case focusReader:
					if m.sidebarTab == tabOutline && len(m.headers) > 0 {
						m.focus = focusOutline
					} else {
						m.focus = focusSearch
						m.searchInput.Focus()
					}
				case focusFollow:
					m.followInput.Blur()
					m.focus = focusReader
				}
				return m, nil
			}
		case "ctrl+w":
			if m.state == stateDashboard {
				m.state = stateWelcome
				m.wikiInput.Focus()
				m.wikiInput.SetValue(m.wiki)
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		contentHeight := m.height - 6
		if contentHeight < 1 {
			contentHeight = 1
		}
		
		var rightWidth int
		if m.hideSidebar {
			rightWidth = m.width - 2
		} else {
			rightWidth = m.width - 33
		}
		if rightWidth < 10 {
			rightWidth = 10
		}
		
		m.viewport.Width = rightWidth
		m.viewport.Height = contentHeight
		
		if m.focus == focusGallery && len(m.articleImages) > 0 {
			m.imageLoading = true
			m.currentImageSixel = ""
			return m, fetchSixelImageCmd(m.articleImages[m.imageIndex].URL, rightWidth-4, contentHeight-4)
		} else if m.inReaderMode {
			wrapped := lipgloss.NewStyle().Width(rightWidth).Render(m.articleRawText)
			m.viewport.SetContent(wrapped)
		}
		
		return m, nil

	case wikiLandingMsg:
		m.loading = false
		m.themeColor = msg.ThemeColor
		m.articleRawText = msg.Text
		m.articleLinks = msg.Links
		m.headers = msg.Headers
		m.outlineCursor = 0
		m.currentTitle = msg.Title
		m.history = nil
		m.logoSixel = msg.LogoSixel
		m.logoLines = strings.Count(m.logoSixel, "\n")
		m.articleImages = msg.Images
		m.inReaderMode = true
		
		wrappedWidth := m.viewport.Width
		if wrappedWidth < 10 {
			wrappedWidth = 10
		}
		wrapped := lipgloss.NewStyle().Width(wrappedWidth).Render(m.articleRawText)
		m.viewport.SetContent(wrapped)
		m.viewport.YOffset = 0
		
		m.focus = focusReader
		m.searchInput.Blur()
		return m, nil

	case searchResultMsg:
		m.loading = false
		m.searchResults = msg
		m.cursor = 0
		if len(m.searchResults) > 0 {
			m.focus = focusList
		} else {
			m.focus = focusSearch
			m.searchInput.Focus()
		}
		return m, nil

	case articleContentMsg:
		m.loading = false
		m.articleRawText = msg.Text
		m.articleLinks = msg.Links
		m.headers = msg.Headers
		m.outlineCursor = 0
		m.articleImages = msg.Images
		m.inReaderMode = true
		
		wrappedWidth := m.viewport.Width
		if wrappedWidth < 10 {
			wrappedWidth = 10
		}
		wrapped := lipgloss.NewStyle().Width(wrappedWidth).Render(m.articleRawText)
		
		m.viewport.SetContent(wrapped)
		m.viewport.YOffset = 0
		m.focus = focusReader
		return m, nil

	case imageSixelMsg:
		m.imageLoading = false
		m.currentImageSixel = string(msg)
		if m.currentImageSixel == "" {
			m.currentImageSixel = "\n  [Failed to load image. Press n/p or Arrow keys to try another]"
		}
		m.viewport.SetContent(m.currentImageSixel)
		m.viewport.YOffset = 0
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg
		return m, nil
	}

	// Welcome State Update Loop
	if m.state == stateWelcome {
		m.wikiInput, cmd = m.wikiInput.Update(msg)
		cmds = append(cmds, cmd)

		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			wiki := m.wikiInput.Value()
			if wiki == "" {
				wiki = "genshin-impact"
			}
			m.wiki = wiki
			m.state = stateDashboard
			m.inReaderMode = false
			m.loading = true
			cmds = append(cmds, fetchWikiLandingCmd(wiki))
		}
		return m, tea.Batch(cmds...)
	}

	// Dashboard State Update Loop
	switch m.focus {
	case focusSearch:
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)

		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
			if m.searchInput.Value() != "" && !m.loading {
				m.loading = true
				m.err = nil
				m.searchResults = nil
				cmds = append(cmds, searchArticlesCmd(m.wiki, m.searchInput.Value()))
			}
		}

	case focusList:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.searchResults)-1 {
					m.cursor++
				}
			case "enter":
				if len(m.searchResults) > 0 && !m.loading {
					m.loading = true
					m.err = nil
					selected := m.searchResults[m.cursor].Title
					if m.currentTitle != "" {
						m.history = append(m.history, m.currentTitle)
					}
					m.currentTitle = selected
					cmds = append(cmds, fetchArticleCmd(m.wiki, selected, m.themeColor))
				}
			case "escape":
				m.focus = focusSearch
				m.searchInput.Focus()
			}
		}

	case focusOutline:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up", "k":
				if m.outlineCursor > 0 {
					m.outlineCursor--
				}
			case "down", "j":
				if m.outlineCursor < len(m.headers)-1 {
					m.outlineCursor++
				}
			case "enter":
				if len(m.headers) > 0 {
					headerText := m.headers[m.outlineCursor].Text
					wrappedWidth := m.viewport.Width
					if wrappedWidth < 10 {
						wrappedWidth = 10
					}
					wrapped := lipgloss.NewStyle().Width(wrappedWidth).Render(m.articleRawText)
					lines := strings.Split(wrapped, "\n")
					var ansiRegex = regexp.MustCompile(`\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
					for i, line := range lines {
						cleanLine := ansiRegex.ReplaceAllString(line, "")
						if strings.Contains(strings.ToLower(cleanLine), strings.ToLower(headerText)) {
							m.viewport.YOffset = i
							break
						}
					}
					m.focus = focusReader
				}
			case "escape":
				m.focus = focusSearch
				m.searchInput.Focus()
			}
		}

	case focusReader:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "escape":
				if !m.hideSidebar {
					if m.sidebarTab == tabOutline && len(m.headers) > 0 {
						m.focus = focusOutline
					} else if len(m.searchResults) > 0 {
						m.focus = focusList
					} else {
						m.focus = focusSearch
						m.searchInput.Focus()
					}
				}
				return m, nil
			case "ctrl+o", "backspace", "H":
				if len(m.history) > 0 {
					prevTitle := m.history[len(m.history)-1]
					m.history = m.history[:len(m.history)-1]
					m.currentTitle = prevTitle
					m.loading = true
					m.err = nil
					cmds = append(cmds, fetchArticleCmd(m.wiki, prevTitle, m.themeColor))
					return m, tea.Batch(cmds...)
				}
			case "f":
				if m.inReaderMode && len(m.articleLinks) > 0 {
					m.focus = focusFollow
					m.followInput.Reset()
					m.followInput.Focus()
					return m, nil
				}
			case "j", "down":
				m.viewport.LineDown(1)
				return m, nil
			case "k", "up":
				m.viewport.LineUp(1)
				return m, nil
			case "ctrl+d", "d":
				m.viewport.HalfPageDown()
				return m, nil
			case "ctrl+u", "u":
				m.viewport.HalfPageUp()
				return m, nil
			case "g":
				m.viewport.GotoTop()
				return m, nil
			case "G":
				m.viewport.GotoBottom()
				return m, nil
			case "i":
				if m.inReaderMode && len(m.articleImages) > 0 {
					m.focus = focusGallery
					m.imageIndex = 0
					m.currentImageSixel = ""
					m.imageLoading = true
					return m, fetchSixelImageCmd(m.articleImages[m.imageIndex].URL, m.viewport.Width-4, m.viewport.Height-4)
				}
			}
		}
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

	case focusFollow:
		m.followInput, cmd = m.followInput.Update(msg)
		cmds = append(cmds, cmd)

		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "escape":
				m.focus = focusReader
				m.followInput.Blur()
				return m, nil
			}

			val := strings.ToLower(m.followInput.Value())
			if val != "" {
				matched := false
				for i := range m.articleLinks {
					hint := GetHintForIndex(i + 1)
					if val == hint {
						matched = true
						m.focus = focusReader
						m.followInput.Blur()
						m.loading = true
						m.err = nil
						selected := m.articleLinks[i].Target
						if m.currentTitle != "" {
							m.history = append(m.history, m.currentTitle)
						}
						m.currentTitle = selected
						cmds = append(cmds, fetchArticleCmd(m.wiki, selected, m.themeColor))
						break
					}
				}

				if !matched {
					anyPrefix := false
					for i := range m.articleLinks {
						if strings.HasPrefix(GetHintForIndex(i+1), val) {
							anyPrefix = true
							break
						}
					}
					if !anyPrefix {
						m.followInput.Reset()
					}
				}
			}
		}

	case focusGallery:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "escape", "i", "q":
				m.focus = focusReader
				wrappedWidth := m.viewport.Width
				if wrappedWidth < 10 {
					wrappedWidth = 10
				}
				wrapped := lipgloss.NewStyle().Width(wrappedWidth).Render(m.articleRawText)
				m.viewport.SetContent(wrapped)
				m.viewport.YOffset = 0
				return m, nil
			case "right", "j", "n":
				if len(m.articleImages) > 0 {
					m.imageIndex = (m.imageIndex + 1) % len(m.articleImages)
					m.currentImageSixel = ""
					m.imageLoading = true
					return m, fetchSixelImageCmd(m.articleImages[m.imageIndex].URL, m.viewport.Width-4, m.viewport.Height-4)
				}
			case "left", "k", "p":
				if len(m.articleImages) > 0 {
					m.imageIndex = (m.imageIndex - 1 + len(m.articleImages)) % len(m.articleImages)
					m.currentImageSixel = ""
					m.imageLoading = true
					return m, fetchSixelImageCmd(m.articleImages[m.imageIndex].URL, m.viewport.Width-4, m.viewport.Height-4)
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing full-screen TUI..."
	}

	// Determine active theme color
	var themeColor lipgloss.Color
	if m.state == stateWelcome {
		activeWiki := m.wikiInput.Value()
		if activeWiki == "" {
			activeWiki = "genshin-impact"
		}
		themeColor = getAccentColor(activeWiki)
	} else {
		themeColor = m.themeColor
	}

	// Common Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(themeColor).
		Background(lipgloss.Color("#1E1E2E")).
		Padding(0, 1)

	// WELCOME SCREEN VIEW
	if m.state == stateWelcome {
		logo := `
    ______ ____  __  ___      ______ _  __ ____
   / ____// __ \/  |/  /     /_  __// / / //  _/
  / /_   / / / / /|_/ /       / /  / /_/ / / /  
 / __/  / /_/ / /  / /       / /  / __  /_/ /   
/_/    /_____/_/  /_/       /_/  /_/ /_//___/   
`
		styledLogo := lipgloss.NewStyle().
			Foreground(themeColor).
			Bold(true).
			Render(logo)

		subtitle := lipgloss.NewStyle().
			Foreground(themeColor).
			Italic(true).
			Render("Fast Deployment & Monitoring — Fandom Wiki Explorer")

		question := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")).
			Bold(true).
			Render("Which wiki subdomain do you want to dive in?")

		hint := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Render("e.g. 'eldenring', 'genshin-impact', or 'minecraft'")

		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(themeColor).
			Padding(0, 2).
			Render(m.wikiInput.View())

		footer := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#45475A")).
			Render("[Enter to Submit • Ctrl+C to Exit]")

		uiContent := lipgloss.JoinVertical(
			lipgloss.Center,
			styledLogo,
			subtitle,
			"",
			question,
			hint,
			"",
			inputBox,
			"",
			footer,
		)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			uiContent,
		)
	}

	// DASHBOARD VIEW
	borderActive := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(themeColor)

	borderInactive := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#313244"))

	// Header
	headerLeft := titleStyle.Render("⚡ Fdm-TUI")
	headerCenter := fmt.Sprintf("Wiki: %s (Ctrl+W to switch)", lipgloss.NewStyle().Foreground(themeColor).Render(m.wiki))
	
	header := lipgloss.JoinHorizontal(lipgloss.Center, headerLeft, "  |  ", headerCenter)
	if m.err != nil {
		header += fmt.Sprintf("  |  %s", lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Render(fmt.Sprintf("Error: %v", m.err)))
	}
	if m.loading {
		header += fmt.Sprintf("  |  %s", lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF")).Render("Loading..."))
	}

	innerContentHeight := m.height - 6

	// Left Pane (Search + Results or Outline)
	var leftView string
	if !m.hideSidebar {
		var leftContent string
		
		if m.logoSixel != "" {
			leftContent += m.logoSixel
		}

		// Draw tabs at the top of the sidebar
		var searchTabStr, outlineTabStr string
		if m.sidebarTab == tabSearch {
			searchTabStr = lipgloss.NewStyle().Foreground(themeColor).Bold(true).Underline(true).Render("1:Search")
			outlineTabStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render("2:Outline")
		} else {
			searchTabStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render("1:Search")
			outlineTabStr = lipgloss.NewStyle().Foreground(themeColor).Bold(true).Underline(true).Render("2:Outline")
		}
		tabs := lipgloss.JoinHorizontal(lipgloss.Center, " ", searchTabStr, " | ", outlineTabStr)
		leftContent += tabs + "\n\n"

		if m.sidebarTab == tabSearch {
			searchBoxStyle := lipgloss.NewStyle().Padding(0, 1)
			if m.focus == focusSearch {
				searchBoxStyle = searchBoxStyle.Border(lipgloss.NormalBorder()).BorderForeground(themeColor)
			} else {
				searchBoxStyle = searchBoxStyle.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#45475A"))
			}
			
			leftContent += searchBoxStyle.Render(m.searchInput.View()) + "\n"

			leftContent += "\nResults:\n"
			if len(m.searchResults) > 0 {
				for i, res := range m.searchResults {
					cursorStr := " "
					style := lipgloss.NewStyle()
					if i == m.cursor {
						cursorStr = ">"
						if m.focus == focusList {
							style = style.Foreground(themeColor).Bold(true)
						} else {
							style = style.Foreground(lipgloss.Color("#6C7086")).Italic(true)
						}
					}
					
					maxTextLen := 30 - 4
					truncatedText := truncate(res.Title, maxTextLen)
					
					leftContent += fmt.Sprintf("%s %s\n", cursorStr, style.Render(truncatedText))
				}
			} else {
				leftContent += "\n(No results)"
			}
		} else {
			leftContent += "Article Outline:\n\n"
			if len(m.headers) > 0 {
				for i, h := range m.headers {
					cursorStr := " "
					style := lipgloss.NewStyle()
					if i == m.outlineCursor {
						cursorStr = ">"
						if m.focus == focusOutline {
							style = style.Foreground(themeColor).Bold(true)
						} else {
							style = style.Foreground(lipgloss.Color("#6C7086")).Italic(true)
						}
					}
					
					indentLen := h.Level - 1
					if indentLen < 0 {
						indentLen = 0
					}
					if indentLen > 3 {
						indentLen = 3
					}
					indent := strings.Repeat("  ", indentLen)
					maxTextLen := 30 - 4 - len(indent)
					truncatedText := truncate(h.Text, maxTextLen)
					
					leftContent += fmt.Sprintf("%s %s%s\n", cursorStr, indent, style.Render(truncatedText))
				}
			} else {
				leftContent += "\n(No headers found)"
			}
		}

		if m.focus == focusSearch || m.focus == focusList || m.focus == focusOutline {
			leftView = borderActive.Width(30).Height(innerContentHeight).Render(leftContent)
		} else {
			leftView = borderInactive.Width(30).Height(innerContentHeight).Render(leftContent)
		}
	}

	// Right Pane (Reader or Gallery)
	var rightView string
	rightContent := m.viewport.View()
	
	if m.focus == focusReader || m.focus == focusGallery {
		rightView = borderActive.Width(m.viewport.Width).Height(m.viewport.Height).Render(rightContent)
	} else {
		rightView = borderInactive.Width(m.viewport.Width).Height(m.viewport.Height).Render(rightContent)
	}

	// Assemble Body
	var body string
	if m.hideSidebar {
		body = rightView
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftView, " ", rightView)
	}

	// Footer help text
	var helpText string
	switch m.focus {
	case focusSearch:
		helpText = "Enter: Search Wiki • Ctrl+T: Toggle Tab • Ctrl+B: Toggle Sidebar • Ctrl+W: Change Wiki • Tab: Focus Reader • Ctrl+C: Quit"
	case focusList:
		helpText = "j/k: Navigate • Enter: Open • Esc: Search Bar • Ctrl+T: Toggle Tab • Ctrl+B: Hide Sidebar • Tab: Focus Reader"
	case focusOutline:
		helpText = "j/k: Navigate • Enter: Scroll To • Esc: Search Bar • Ctrl+T: Toggle Tab • Ctrl+B: Hide Sidebar • Tab: Focus Reader"
	case focusReader:
		backHelp := ""
		if len(m.history) > 0 {
			backHelp = "Ctrl+O: Back • "
		}
		galleryHelp := ""
		if len(m.articleImages) > 0 {
			galleryHelp = fmt.Sprintf("i: Gallery (%d imgs) • ", len(m.articleImages))
		}
		if m.hideSidebar {
			helpText = fmt.Sprintf("j/k: Scroll • d/u: Half-Page • g/G: Top/Bottom • f: Follow Link • %s%sCtrl+B: Show Sidebar • Ctrl+C: Quit", galleryHelp, backHelp)
		} else {
			helpText = fmt.Sprintf("j/k: Scroll • d/u: Half-Page • g/G: Top/Bottom • f: Follow Link • %s%sEsc: Sidebar • Ctrl+T: Toggle Tab • Ctrl+B: Hide Sidebar • Tab: Focus Sidebar", galleryHelp, backHelp)
		}
	case focusFollow:
		helpText = m.followInput.View() + " [Esc: Cancel]"
	case focusGallery:
		caption := "Image"
		if len(m.articleImages) > 0 && m.imageIndex < len(m.articleImages) {
			caption = m.articleImages[m.imageIndex].Caption
		}
		loadingStr := ""
		if m.imageLoading {
			loadingStr = " [Loading...] "
		}
		helpText = fmt.Sprintf("Gallery: [Image %d of %d] %s%s • j/k (or Left/Right or n/p): Prev/Next • Esc / i: Return to Article", m.imageIndex+1, len(m.articleImages), loadingStr, caption)
	}
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Background(lipgloss.Color("#1E1E2E")).
		Padding(0, 1).
		Render(helpText)

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, body, footer)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, an error occurred: %v", err)
		os.Exit(1)
	}
}
