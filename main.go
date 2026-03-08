package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type searchCancelledMsg struct{}
type debounceMsg struct {
	tag   int
	query string
}
type model struct {
	articles    wikiSearchResponse
	cursor      int
	textInput   textinput.Model
	cancel      context.CancelFunc
	debounceTag int
}

func debounceCmd(tag int, query string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(50 * time.Millisecond)
		return debounceMsg{tag: tag, query: query}
	}
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Start searching..."
	ti.SetVirtualCursor(false)
	ti.Focus()
	ti.CharLimit = 156
	ti.SetWidth(20)
	return model{
		cursor:    0,
		textInput: ti,
	}
}
func (m model) Init() tea.Cmd {
	return nil
}

var wikiClient = &http.Client{
	Timeout: 15 * time.Second,
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {

	// Handling a built-in key message
	case debounceMsg:
		// Only fire the search if the user hasn't typed anything new
		// since this specific debounce command was triggered.
		if msg.tag == m.debounceTag {
			if m.cancel != nil {
				m.cancel()
			}
			var ctx context.Context
			ctx, m.cancel = context.WithCancel(context.Background())
			return m, fetchSearchResults(ctx, msg.query)
		}
		return m, nil
	case wikiSearchResponse:
		m.articles = msg
		m.cursor = 0
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit

		case "up":
			m.cursor--
		case "down":
			m.cursor++
		}

		// --- THE CLAMP ---
		maxIndex := max(0, len(m.articles.Pages)-1)
		m.cursor = max(0, min(maxIndex, m.cursor))
	}
	// --- THE LIVE SEARCH LOGIC ---

	// 1. Record the value of the input before the keystroke is processed
	oldValue := m.textInput.Value()

	// 2. Pass the message to the text input so it can process the keystroke
	var tiCmd tea.Cmd
	m.textInput, tiCmd = m.textInput.Update(msg)
	cmds = append(cmds, tiCmd)

	// 3. Check if the keystroke actually changed the text
	newValue := m.textInput.Value()
	if oldValue != newValue {
		// The text changed!
		m.debounceTag++
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		if strings.TrimSpace(newValue) == "" {
			// If they backspaced everything, clear the results
			m.articles.Pages = nil
		} else {
			// Queue up a new debounce command
			cmds = append(cmds, debounceCmd(m.debounceTag, newValue))
		}
	}

	// tea.Batch runs all queued commands concurrently
	return m, tea.Batch(cmds...)
}
func (m model) View() tea.View {
	str :=
		m.textInput.View() +
			m.headerView() +
			m.listView() +
			m.footerView()

	return tea.NewView(str)
}

// Helper for the title
func (m model) headerView() string {
	return "\n  Wikipedia Search Results:\n"
}

// Helper for the interactive list logic
func (m model) listView() string {
	var b strings.Builder
	for i, page := range m.articles.Pages {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, page.Title)
	}
	return b.String()
}

// Helper for navigation hints
func (m model) footerView() string {
	return "\n  Press 'up'/'down' to move, 'Esc' to quit.\n"
}

func fetchSearchResults(ctx context.Context, searchQuery string) tea.Cmd {
	return func() tea.Msg {
		var searchResults wikiSearchResponse
		limit := 10
		baseSearchURL := "https://en.wikipedia.org/w/rest.php/v1/search/page?"

		u, err := url.Parse(baseSearchURL)
		if err != nil {
			return searchResults
		}
		params := url.Values{}
		params.Add("q", searchQuery)
		params.Add("limit", strconv.Itoa(limit))

		u.RawQuery = params.Encode()
		fullSearchURL := u.String()

		req, err := http.NewRequestWithContext(ctx, "GET", fullSearchURL, nil)
		if err != nil {
			return searchResults
		}
		req.Header.Set("User-Agent", "MyGoWikiTool/1.0 (contact: user@example.com)")
		resp, err := wikiClient.Do(req)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return searchCancelledMsg{}
			}
			return searchResults
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return searchResults
		}
		if err := json.Unmarshal(body, &searchResults); err != nil {
			return searchResults
		}
		return searchResults
	}
}
func main() {

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
