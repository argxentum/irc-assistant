package summary

import (
	"assistant/pkg/api/text"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const defaultMaxLength = 256

// Sanitize normalizes and truncates content for summary display.
func Sanitize(s string) string {
	return text.SanitizeToMaxLength(s, defaultMaxLength)
}

// PageMetadata holds the title and description extracted from an HTML document.
type PageMetadata struct {
	Title       string
	Description string
}

// ExtractMetadata extracts a page title and description from an HTML document.
// Priority order: OG tags → Twitter Card → Schema.org → standard meta → HTML elements.
// CSS-like content in <title> and <h1> is discarded.
func ExtractMetadata(doc *goquery.Document) PageMetadata {
	title := text.Coalesce(
		metaContent(doc, "og:title"),
		metaContent(doc, "twitter:title"),
		metaAttr(doc, "itemprop", "name"),
		metaContent(doc, "title"),
		cleanHTMLText(doc, "title"),
		cleanHTMLText(doc, "html body h1"),
	)

	description := text.Coalesce(
		metaContent(doc, "og:description"),
		metaContent(doc, "twitter:description"),
		metaAttr(doc, "itemprop", "description"),
		metaContent(doc, "description"),
	)

	return PageMetadata{
		Title:       Sanitize(title),
		Description: Sanitize(description),
	}
}

func cleanHTMLText(doc *goquery.Document, selector string) string {
	v := strings.TrimSpace(doc.Find(selector).First().Text())
	cssIndicators := []string{"{", ":", ";", "}"}
	if text.ContainsAll(v, cssIndicators) {
		return ""
	}
	return v
}

func metaContent(doc *goquery.Document, name string) string {
	if val, exists := doc.Find("meta[property='" + name + "']").First().Attr("content"); exists {
		if v := strings.TrimSpace(val); v != "" {
			return v
		}
	}
	if val, exists := doc.Find("meta[name='" + name + "']").First().Attr("content"); exists {
		if v := strings.TrimSpace(val); v != "" {
			return v
		}
	}
	return ""
}

func metaAttr(doc *goquery.Document, attr, value string) string {
	if val, exists := doc.Find("meta[" + attr + "='" + value + "']").First().Attr("content"); exists {
		if v := strings.TrimSpace(val); v != "" {
			return v
		}
	}
	return ""
}
