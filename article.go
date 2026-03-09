package main

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"golang.org/x/net/html"
)

// ── Glamour Custom Styles ─────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool  { return &b }
func uintPtr(u uint) *uint  { return &u }

var articleRenderStyles = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "",
			BlockSuffix: "",
			Color:       strPtr("#F0F0F5"),
		},
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: strPtr("#E4E4EB"),
		},
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix:     "",
			BlockSuffix:     "\n",
			Prefix:          "│ ",
			Color:           strPtr("#BF5AF2"),
			BackgroundColor: strPtr("#2E3240"),
		},
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "",
			BlockSuffix: "\n",
			Color:       strPtr("#F7F7FF"),
			Bold:        boolPtr(true),
		},
		Margin: uintPtr(1),
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "",
			BlockSuffix: "\n",
			Color:       strPtr("#64D2FF"),
			Bold:        boolPtr(true),
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "",
			BlockSuffix: "\n",
			Color:       strPtr("#BF5AF2"),
			Bold:        boolPtr(true),
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "",
			BlockSuffix: "\n",
			Color:       strPtr("#FF9F0A"),
			Bold:        boolPtr(true),
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix: "",
			BlockSuffix: "\n",
			Color:       strPtr("#30D158"),
			Bold:        boolPtr(true),
		},
	},
	Link: ansi.StylePrimitive{
		Color:     strPtr("#64D2FF"),
		Underline: boolPtr(true),
	},
	LinkText: ansi.StylePrimitive{
		Color:     strPtr("#64D2FF"),
		Underline: boolPtr(true),
	},
	Strong: ansi.StylePrimitive{
		Bold:  boolPtr(true),
		Color: strPtr("#FFD760"),
	},
	Emph: ansi.StylePrimitive{
		Italic: boolPtr(true),
		Color:  strPtr("#A5B4FF"),
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("#F2F2F7"),
			},
		},
		LevelIndent: 2,
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           strPtr("#F2F2F7"),
				BackgroundColor: strPtr("#2E3240"),
			},
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           strPtr("#E2E2E8"),
				BackgroundColor: strPtr("#18191E"),
			},
		},
		CenterSeparator: strPtr("╂"),
		ColumnSeparator: strPtr("│"),
		RowSeparator:    strPtr("═"),
	},
	HorizontalRule: ansi.StylePrimitive{
		Format: strings.Repeat("─", 36),
		Color:  strPtr("#2F3440"),
	},
	DefinitionTerm: ansi.StylePrimitive{
		Color: strPtr("#64D2FF"),
		Bold:  boolPtr(true),
	},
	DefinitionDescription: ansi.StylePrimitive{
		Color: strPtr("#E4E4EA"),
	},
}

// ── Wikipedia API ─────────────────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 15 * time.Second}


// ── HTML pre-cleaner ──────────────────────────────────────────────────────────

var sectionsToTrim = map[string]bool{
	"References": true, "Notes": true, "Footnotes": true,
	"External_links": true, "Further_reading": true,
	"See_also": true, "Bibliography": true, "Citations": true,
}

var classesToStrip = map[string]bool{
	"mw-editsection": true, "mw-references-wrap": true, "reflist": true,
	"infobox": true, "infobox-table": true,
	"navbox": true, "navbox-inner": true, "navbox-list": true,
	"sidebar": true, "vertical-navbox": true,
	"mbox": true, "ambox": true, "ombox": true, "tmbox": true, "fmbox": true,
	"thumb": true, "thumbinner": true,
	"mw-empty-elt": true, "hatnote": true,
	"navigation-not-searchable": true, "noprint": true,
}

func cleanHTML(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}
	stripDOM(doc)
	var sb strings.Builder
	html.Render(&sb, doc)
	return sb.String()
}

func stripDOM(n *html.Node) {
	var toRemove []*html.Node
	skipAfter := false

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if skipAfter {
			toRemove = append(toRemove, c)
			continue
		}
		if domShouldStrip(c) {
			toRemove = append(toRemove, c)
			continue
		}
		if domIsTrimSection(c) {
			toRemove = append(toRemove, c)
			skipAfter = true
			continue
		}
		stripDOM(c)
	}
	for _, r := range toRemove {
		n.RemoveChild(r)
	}
}

func domShouldStrip(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.Data {
	case "figure", "style", "script", "sup", "link", "meta", "img":
		return true
	}
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, cls := range strings.Fields(a.Val) {
				if classesToStrip[cls] {
					return true
				}
			}
		}
	}
	return false
}

func domIsTrimSection(n *html.Node) bool {
	if n.Type != html.ElementNode || n.Data != "h2" {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == "id" && sectionsToTrim[a.Val] {
			return true
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "span" {
			for _, a := range c.Attr {
				if a.Key == "id" && sectionsToTrim[a.Val] {
					return true
				}
			}
		}
	}
	return false
}

// ── Markdown cleanup ──────────────────────────────────────────────────────────

var (
	citationRe   = regexp.MustCompile(`\[\d+\]|\[note\s*\d+\]|\[citation needed\]|\[nb \d+\]`)
	mdLinkRe     = regexp.MustCompile(`\[([^\]\n]+)\]\([^)\n]*\)`)
	mdImageRe    = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	editWordRe   = regexp.MustCompile(`(?m)^\s*edit\s*$`)
	blankLinesRe = regexp.MustCompile(`\n{3,}`)
)

func cleanMarkdown(md string) string {
	md = mdImageRe.ReplaceAllString(md, "") 
	md = citationRe.ReplaceAllString(md, "")
	md = mdLinkRe.ReplaceAllString(md, "[$1]()") // Format links as dummies for highlighting
	md = editWordRe.ReplaceAllString(md, "")
	md = blankLinesRe.ReplaceAllString(md, "\n\n")
	return strings.TrimSpace(md)
}

func markdownFromHTML(htmlContent string) (string, error) {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
		),
	)
	conv.Register.Plugin(
		table.NewTablePlugin(
			table.WithNewlineBehavior(table.NewlineBehaviorPreserve),
			table.WithSpanCellBehavior(table.SpanBehaviorMirror),
		),
	)
	return conv.ConvertString(htmlContent)
}

// ── Main Execution ────────────────────────────────────────────────────────────
func fetchAndParseArticle(articleKey string) (string, error) {
	htmlContent, err := fetchArticle(articleKey)
	if err != nil {
		return "", err
	}
	cleanedHTML := cleanHTML(htmlContent)
	markdown, err := markdownFromHTML(cleanedHTML)
	if err != nil {
		return "", err
	}
	markdown = cleanMarkdown(markdown)
	return markdown, nil

}

func applyGlamour(markdown string, terminalWidth int) (string) {

	// 5. Render with Glamour using the custom style struct
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(articleRenderStyles),
		glamour.WithWordWrap(terminalWidth),
	)
	if err != nil {
		log.Fatalf("Failed to setup renderer: %v", err)
	}

	rendered, err := r.Render(markdown)
	if err != nil {
		log.Fatalf("Failed to render markdown: %v", err)
	}

	return rendered
}