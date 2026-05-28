package docs

import (
	"context"
	"fmt"
	"strings"
)

// Generator handles PDF generation
// For hackathon: returns HTML representation
// Production: uses go-rod + Chromium
type Generator struct{}

// NewGenerator creates a new PDF generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateHTML renders document as HTML
func (g *Generator) GenerateHTML(
	doc *Document,
) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
  body {
    font-family: sans-serif;
    max-width: 800px;
    margin: 40px auto;
    padding: 0 40px;
    color: #333;
    line-height: 1.6;
  }
  h1 { color: #0052FF; }
  .block { margin-bottom: 16px; }
</style>
</head>
<body>`)

	sb.WriteString(
		fmt.Sprintf("<h1>%s</h1>", doc.Title),
	)

	for _, block := range doc.Content.Blocks {
		content := block.Content

		// Replace variables
		for key, val := range doc.Content.Variables {
			content = strings.ReplaceAll(
				content,
				"{{"+key+"}}",
				val,
			)
		}

		switch block.Type {
		case "heading":
			sb.WriteString(
				fmt.Sprintf(
					`<div class="block"><h2>%s</h2></div>`,
					content,
				),
			)
		case "text":
			sb.WriteString(
				fmt.Sprintf(
					`<div class="block"><p>%s</p></div>`,
					content,
				),
			)
		case "divider":
			sb.WriteString("<hr>")
		default:
			sb.WriteString(
				fmt.Sprintf(
					`<div class="block">%s</div>`,
					content,
				),
			)
		}
	}

	sb.WriteString("</body></html>")
	return sb.String()
}

// GeneratePDF generates PDF bytes from document
// For hackathon: returns HTML as bytes
// TODO: Integrate go-rod for real PDF
func (g *Generator) GeneratePDF(
	ctx context.Context,
	doc *Document,
) ([]byte, string, error) {
	html := g.GenerateHTML(doc)
	// Return HTML for now
	// Production: launch Chromium via go-rod
	return []byte(html), "text/html", nil
}
