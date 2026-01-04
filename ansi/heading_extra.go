package ansi

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
)

// HeadingElementWithBackground is an enhanced version of HeadingElement
// that properly supports background colors.
//
// The standard HeadingElement has background color issues because:
// 1. Suffix "\n" creates an empty line with only ANSI codes, no visible chars
// 2. Underline extends to the entire line width (including padding)
//
// This version fixes both issues by:
// 1. Filling the rest of the line with spaces (with background color)
// 2. Using a separate style for padding (background only, no underline)
type HeadingElementWithBackground struct {
	Level int
	First bool
}

// Render renders a HeadingElementWithBackground.
func (e *HeadingElementWithBackground) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := ctx.options.Styles.Heading

	switch e.Level {
	case h1:
		rules = cascadeStyles(rules, ctx.options.Styles.H1)
	case h2:
		rules = cascadeStyles(rules, ctx.options.Styles.H2)
	case h3:
		rules = cascadeStyles(rules, ctx.options.Styles.H3)
	case h4:
		rules = cascadeStyles(rules, ctx.options.Styles.H4)
	case h5:
		rules = cascadeStyles(rules, ctx.options.Styles.H5)
	case h6:
		rules = cascadeStyles(rules, ctx.options.Styles.H6)
	}

	if !e.First {
		_, _ = renderText(w, bs.Current().Style.StylePrimitive, "\n")
	}

	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: cascadeStyle(bs.Current().Style, rules, false),
	}
	bs.Push(be)

	_, _ = renderText(w, bs.Parent().Style.StylePrimitive, rules.BlockPrefix)
	_, _ = renderText(bs.Current().Block, bs.Current().Style.StylePrimitive, rules.Prefix)
	return nil
}

// Finish finishes rendering a HeadingElementWithBackground.
func (e *HeadingElementWithBackground) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := bs.Current().Style
	// Use NewMarginWriterBatch instead of NewMarginWriter to get FinalPadFunc support
	// which uses background-only style for final line padding (no underline artifacts)
	mw := NewMarginWriterBatch(ctx, w, rules)
	defer mw.Close() //nolint:errcheck

	flow := lipgloss.Wrap(bs.Current().Block.String(), int(bs.Width(ctx)), "") //nolint: gosec
	_, err := io.WriteString(mw, flow)
	if err != nil {
		return fmt.Errorf("glamour: error writing to writer: %w", err)
	}

	// Render Suffix with special handling for newlines and background colors
	// Fast path: if no background color or Suffix is not "\n", use standard rendering
	if rules.Suffix != "\n" || rules.StylePrimitive.BackgroundColor == nil {
		_, _ = renderText(w, bs.Current().Style.StylePrimitive, rules.Suffix)
	} else {
		// Slow path: fill the rest of the line with spaces (with background color)
		width := ctx.options.WordWrap
		if width > 0 {
			usedWidth := lipgloss.Width(flow)
			
			// Optimization: only check for newlines if the width suggests wrapping occurred
			if usedWidth >= width {
				lastNewlineIdx := strings.LastIndex(flow, "\n")
				if lastNewlineIdx >= 0 {
					lastLine := flow[lastNewlineIdx+1:]
					usedWidth = lipgloss.Width(lastLine)
				}
			}
			
			if usedWidth < width {
				// Create a new style with only background color, removing underline, color, etc.
				paddingStyle := StylePrimitive{
					BackgroundColor: rules.StylePrimitive.BackgroundColor,
				}
				padding := width - usedWidth
				_, _ = renderText(w, paddingStyle, strings.Repeat(" ", padding))
			}
		}
		// Output the newline
		_, _ = io.WriteString(w, "\n")
	}
	
	_, _ = renderText(w, bs.Parent().Style.StylePrimitive, rules.BlockSuffix)

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}
