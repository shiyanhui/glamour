package ansi

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/alecthomas/chroma/v2/styles"
	"charm.land/lipgloss/v2"
)

// CodeBlockElementWithBackground is an optimized version of CodeBlockElement
// that properly supports background colors.
//
// The standard CodeBlockElement has background color issues because:
// 1. Chroma's terminal formatters don't support background colors
// 2. Indentation uses parent element's style instead of CodeBlock's style
//
// This version fixes both issues by:
// 1. Using rules.StylePrimitive for indentation (contains CodeBlock's background color)
// 2. Wrapping Chroma output line-by-line with background color
type CodeBlockElementWithBackground struct {
	Code     string
	Language string
}

// Render renders a CodeBlockElementWithBackground with proper background color support.
func (e *CodeBlockElementWithBackground) Render(w io.Writer, ctx RenderContext) error {
	var indentation uint
	var margin uint
	formatter := chromaFormatter
	rules := ctx.options.Styles.CodeBlock
	if rules.Indent != nil {
		indentation = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}
	if len(ctx.options.ChromaFormatter) > 0 {
		formatter = ctx.options.ChromaFormatter
	}
	theme := rules.Theme

	if rules.Chroma != nil {
		// Generate unique theme name based on configuration content
		// This allows different color schemes to coexist without conflicts
		theme = generateChromaThemeName(rules.Chroma)
		mutex.Lock()
		// Don't register the style if it's already registered.
		_, ok := styles.Registry[theme]
		if !ok {
			styles.Register(chroma.MustNewStyle(theme,
				chroma.StyleEntries{
					chroma.Text:                chromaStyle(rules.Chroma.Text),
					chroma.Error:               chromaStyle(rules.Chroma.Error),
					chroma.Comment:             chromaStyle(rules.Chroma.Comment),
					chroma.CommentPreproc:      chromaStyle(rules.Chroma.CommentPreproc),
					chroma.Keyword:             chromaStyle(rules.Chroma.Keyword),
					chroma.KeywordReserved:     chromaStyle(rules.Chroma.KeywordReserved),
					chroma.KeywordNamespace:    chromaStyle(rules.Chroma.KeywordNamespace),
					chroma.KeywordType:         chromaStyle(rules.Chroma.KeywordType),
					chroma.Operator:            chromaStyle(rules.Chroma.Operator),
					chroma.Punctuation:         chromaStyle(rules.Chroma.Punctuation),
					chroma.Name:                chromaStyle(rules.Chroma.Name),
					chroma.NameBuiltin:         chromaStyle(rules.Chroma.NameBuiltin),
					chroma.NameTag:             chromaStyle(rules.Chroma.NameTag),
					chroma.NameAttribute:       chromaStyle(rules.Chroma.NameAttribute),
					chroma.NameClass:           chromaStyle(rules.Chroma.NameClass),
					chroma.NameConstant:        chromaStyle(rules.Chroma.NameConstant),
					chroma.NameDecorator:       chromaStyle(rules.Chroma.NameDecorator),
					chroma.NameException:       chromaStyle(rules.Chroma.NameException),
					chroma.NameFunction:        chromaStyle(rules.Chroma.NameFunction),
					chroma.NameOther:           chromaStyle(rules.Chroma.NameOther),
					chroma.Literal:             chromaStyle(rules.Chroma.Literal),
					chroma.LiteralNumber:       chromaStyle(rules.Chroma.LiteralNumber),
					chroma.LiteralDate:         chromaStyle(rules.Chroma.LiteralDate),
					chroma.LiteralString:       chromaStyle(rules.Chroma.LiteralString),
					chroma.LiteralStringEscape: chromaStyle(rules.Chroma.LiteralStringEscape),
					chroma.GenericDeleted:      chromaStyle(rules.Chroma.GenericDeleted),
					chroma.GenericEmph:         chromaStyle(rules.Chroma.GenericEmph),
					chroma.GenericInserted:     chromaStyle(rules.Chroma.GenericInserted),
					chroma.GenericStrong:       chromaStyle(rules.Chroma.GenericStrong),
					chroma.GenericSubheading:   chromaStyle(rules.Chroma.GenericSubheading),
					chroma.Background:          chromaStyle(rules.Chroma.Background),
				}))
		}
		mutex.Unlock()
	}

	// Use rules.StylePrimitive for indentation (not bs.Current().Style.StylePrimitive)
	// This ensures the indentation uses CodeBlock's background color
	iw := NewIndentWriterBatch(w, int(indentation+margin), func(_ io.Writer, count int) { //nolint:gosec
		_, _ = renderText(w, rules.StylePrimitive, strings.Repeat(" ", count))
	})
	defer iw.Close() //nolint:errcheck

	if len(theme) > 0 {
		// Render BlockPrefix with CodeBlock's style
		_, _ = renderText(iw, rules.StylePrimitive, rules.BlockPrefix)

		// Chroma output to buffer first
		var chromaBuf bytes.Buffer
		err := quick.Highlight(&chromaBuf, e.Code, e.Language, formatter, theme)
		if err != nil {
			return fmt.Errorf("glamour: error highlighting code: %w", err)
		}

		// Fix Chroma's reset codes to preserve background color
		// Chroma outputs \x1b[0m which resets ALL styles including background
		// We need to:
		// 1. Reset foreground color (\x1b[39m)
		// 2. Re-apply background color if one is set
		chromaOutput := chromaBuf.String()
		
		// Build replacement string: reset foreground + re-apply background
		resetReplacement := "\x1b[39m"
		bgColorCode := ""
		if rules.BackgroundColor != nil && *rules.BackgroundColor != "" {
			// Parse background color and build ANSI code
			bgColor := *rules.BackgroundColor
			if len(bgColor) > 0 && bgColor[0] == '#' {
				// Convert hex color to RGB
				if len(bgColor) == 7 {
					var r, g, b int
					fmt.Sscanf(bgColor, "#%02x%02x%02x", &r, &g, &b)
					bgColorCode = fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
					resetReplacement = fmt.Sprintf("\x1b[39m%s", bgColorCode)
				}
			}
		}
		
		chromaOutput = strings.ReplaceAll(chromaOutput, "\x1b[0m", resetReplacement)

		// Process line by line, wrapping each line with background color
		if len(chromaOutput) > 0 {
			lines := strings.Split(chromaOutput, "\n")
			for i, line := range lines {
				// Add background color at the start of each line
				if bgColorCode != "" {
					_, _ = io.WriteString(iw, bgColorCode)
				}
				
				// Write the line content (with Chroma's ANSI codes preserved)
				_, _ = io.WriteString(iw, line)
				
				// Pad the line to fill width with background color
				lineWidth := lipgloss.Width(line)
				if ctx.options.WordWrap > 0 && lineWidth < int(ctx.options.WordWrap) {
					padding := int(ctx.options.WordWrap) - lineWidth
					if padding > 0 {
						// Reset foreground color before padding to avoid inheriting Chroma's colors
						// But keep background color active for padding
						_, _ = io.WriteString(iw, "\x1b[39m")
						// Write padding spaces with background color
						_, _ = io.WriteString(iw, strings.Repeat(" ", padding))
					}
				}
				
				// Reset background at the end of each line (before newline)
				if bgColorCode != "" {
					_, _ = io.WriteString(iw, "\x1b[49m")
				}
				
				// Add newline except for the last line
				if i < len(lines)-1 {
					_, _ = io.WriteString(iw, "\n")
				}
			}
		}

		// Render BlockSuffix with CodeBlock's style
		_, _ = renderText(iw, rules.StylePrimitive, rules.BlockSuffix)
		return nil
	}

	// fallback rendering
	el := &BaseElement{
		Token: e.Code,
		Style: rules.StylePrimitive,
	}

	return el.Render(iw, ctx)
}
