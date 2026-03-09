package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
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
type sessionState int
type searchModel struct {
	articles    wikiSearchResponse
	cursor      int
	textInput   textinput.Model
	cancel      context.CancelFunc
	debounceTag int
}
type articleModel struct {
	article wikiPage
}
type model struct {
	state   sessionState
	search  searchModel
	article articleModel
}

const (
	searchState sessionState = iota
	articleState
)

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
		search: searchModel{
			cursor:    0,
			textInput: ti,
		},
	}
}
func (m model) Init() tea.Cmd {
	return nil
}
func (m searchModel) Init() tea.Cmd {
	return nil
}
func (m articleModel) Init() tea.Cmd {
	return nil
}

var wikiClient = &http.Client{
	Timeout: 15 * time.Second,
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func (m articleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.state {
	case searchState:
		newSearch, searchCmd := m.search.Update(msg)
		// 2. Cast it back to our specific type
		m.search = newSearch.(searchModel)
		cmd = searchCmd
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "enter" {
				if len(m.search.articles.Pages) > 0 {
					// 2. Grab the specific article based on the search cursor
					selectedArticle := m.search.articles.Pages[m.search.cursor]

					// 3. Assign it to the article model
					m.article.article = selectedArticle

					// 4. Switch the state
					m.state = articleState

					// 5. Important: Return immediately to stop the search model
					// from processing 'enter' further
					return m, nil
				}
			}
		}
	case articleState:
		newArticle, articleCmd := m.article.Update(msg)
		m.article = newArticle.(articleModel)
		cmd = articleCmd

		// Handle going back to search
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.state = searchState
			return m, nil
		}
	}
	return m, cmd
}
func (m model) View() tea.View {
	switch m.state {
	case articleState:
		return m.article.View()
	default:
		return m.search.View()
	}
}
func (m articleModel) View() tea.View {
	return tea.NewView(m.article.Title)
}

func (m searchModel) View() tea.View {

	str :=
		m.textInput.View() +
			m.headerView() +
			m.listView() +
			m.footerView()

	return tea.NewView(str)
}

// Helper for the title
func (m searchModel) headerView() string {
	return "\n  Wikipedia Search Results:\n"
}

// Helper for the interactive list logic
func (m searchModel) listView() string {
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
func (m searchModel) footerView() string {
	return "\n  Press 'up'/'down' to move, 'Esc' to quit.\n"
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func fetchSearchResults(ctx context.Context, searchQuery string) tea.Cmd {
	return func() tea.Msg {
		limit := 10

		// Call the abstracted API function
		results, err := searchWikipedia(ctx, wikiClient, searchQuery, limit)
		if err != nil {
			// Handle the specific cancellation case
			if errors.Is(err, context.Canceled) {
				return searchCancelledMsg{}
			}
			// Return a dedicated error message instead of an empty struct
			return errMsg{err}
		}

		// Return the successful payload
		return results
	}
}
func main() {

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}