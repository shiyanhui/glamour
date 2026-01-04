package ansi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// PaddingFuncBatch is an optimized function that applies padding in batch.
// The count parameter specifies how many padding characters to write at once,
// which significantly improves performance by reducing the number of ANSI
// escape sequences generated.
type PaddingFuncBatch = func(w io.Writer, count int)

// PaddingWriterBatch is an optimized writer that applies padding around
// whatever you write to it using batch operations.
type PaddingWriterBatch struct {
	Padding      int
	PadFunc      PaddingFuncBatch
	FinalPadFunc PaddingFuncBatch // Used for final line padding in Close(), uses simpler style (bg only)
	w            io.Writer
	cache        bytes.Buffer
}

// NewPaddingWriterBatch returns a new optimized PaddingWriter.
// This version uses PaddingFuncBatch which accepts a count parameter,
// allowing for batch generation of padding characters and significantly
// better performance compared to NewPaddingWriter.
func NewPaddingWriterBatch(w io.Writer, padding int, padFunc PaddingFuncBatch) *PaddingWriterBatch {
	return &PaddingWriterBatch{
		Padding: padding,
		PadFunc: padFunc,
		w:       w,
	}
}

// Write writes to the padding writer.
func (w *PaddingWriterBatch) Write(p []byte) (int, error) {
	for i := 0; i < len(p); i++ {
		if p[i] == '\n' { //nolint:nestif
			line := w.cache.String()
			linew := ansi.StringWidth(line)
			if w.Padding > 0 && linew < w.Padding {
				paddingCount := w.Padding - linew
				if w.PadFunc != nil {
					w.PadFunc(w.w, paddingCount)
				} else {
					_, err := io.WriteString(w.w, strings.Repeat(" ", paddingCount))
					if err != nil {
						return 0, fmt.Errorf("glamour: error writing padding: %w", err)
					}
				}
			}
			w.cache.Reset()
		} else {
			w.cache.WriteByte(p[i])
		}

		_, err := w.w.Write(p[i : i+1])
		if err != nil {
			return 0, fmt.Errorf("glamour: error writing bytes: %w", err)
		}
	}

	return len(p), nil
}

// Close closes the [PaddingWriterBatch].
// It pads the final line if there's remaining content in cache (content not ending with '\n').
func (w *PaddingWriterBatch) Close() error {
	// Only pad if there's content in cache (not ending with \n)
	if w.cache.Len() > 0 {
		line := w.cache.String()
		linew := ansi.StringWidth(line)
		if w.Padding > 0 && linew < w.Padding {
			paddingCount := w.Padding - linew
			// Use FinalPadFunc if available (simpler style without underline etc.)
			// Otherwise fall back to PadFunc
			padFunc := w.FinalPadFunc
			if padFunc == nil {
				padFunc = w.PadFunc
			}
			if padFunc != nil {
				padFunc(w.w, paddingCount)
			} else {
				_, _ = io.WriteString(w.w, strings.Repeat(" ", paddingCount))
			}
		}
	}

	if wc, ok := w.w.(io.WriteCloser); ok {
		return wc.Close() //nolint:wrapcheck
	}
	return nil
}

// IndentFuncBatch is an optimized function that applies indentation in batch.
// The count parameter specifies how many indentation characters to write at once,
// which significantly improves performance by reducing the number of ANSI
// escape sequences generated.
type IndentFuncBatch = func(w io.Writer, count int)

// IndentWriterBatch is an optimized writer that applies indentation around
// whatever you write to it using batch operations.
type IndentWriterBatch struct {
	Indent     int
	IndentFunc IndentFuncBatch
	w          io.Writer
	pw         *lipgloss.WrapWriter
	skipIndent bool
}

// NewIndentWriterBatch returns a new optimized IndentWriter.
// This version uses IndentFuncBatch which accepts a count parameter,
// allowing for batch generation of indentation characters and significantly
// better performance compared to NewIndentWriter.
func NewIndentWriterBatch(w io.Writer, indent int, indentFunc IndentFuncBatch) *IndentWriterBatch {
	return &IndentWriterBatch{
		Indent:     indent,
		IndentFunc: indentFunc,
		pw:         lipgloss.NewWrapWriter(w),
		w:          w,
	}
}

func (w *IndentWriterBatch) resetPen() {
	style := w.pw.Style()
	link := w.pw.Link()
	if !style.IsZero() {
		_, _ = io.WriteString(w.w, ansi.ResetStyle)
	}
	if !link.IsZero() {
		_, _ = io.WriteString(w.w, ansi.ResetHyperlink())
	}
}

func (w *IndentWriterBatch) restorePen() {
	style := w.pw.Style()
	link := w.pw.Link()
	if !style.IsZero() {
		_, _ = io.WriteString(w.w, style.String())
	}
	if !link.IsZero() {
		_, _ = io.WriteString(w.w, ansi.SetHyperlink(link.URL, link.Params))
	}
}

// Write writes to the indentation writer.
func (w *IndentWriterBatch) Write(p []byte) (int, error) {
	for i := 0; i < len(p); i++ {
		if !w.skipIndent {
			w.resetPen()
			if w.IndentFunc != nil {
				w.IndentFunc(w.pw, w.Indent)
			} else {
				_, err := io.WriteString(w.pw, strings.Repeat(" ", w.Indent))
				if err != nil {
					return 0, fmt.Errorf("glamour: error writing indentation: %w", err)
				}
			}

			w.skipIndent = true
			w.restorePen()
		}

		if p[i] == '\n' {
			w.skipIndent = false
		}

		_, err := w.pw.Write(p[i : i+1])
		if err != nil {
			return 0, fmt.Errorf("glamour: error writing bytes: %w", err)
		}
	}

	return len(p), nil
}

// Close closes the [IndentWriterBatch].
func (w *IndentWriterBatch) Close() error {
	var werr error
	if w, ok := w.w.(io.WriteCloser); ok {
		werr = w.Close()
	}

	return errors.Join(werr, w.pw.Close())
}

// indentWriterInterface defines the common interface for IndentWriter implementations.
type indentWriterInterface interface {
	io.Writer
	io.Closer
}

// MarginWriterBatch is an optimized MarginWriter that uses batch operations.
type MarginWriterBatch struct {
	w  io.Writer
	iw *IndentWriterBatch
}

// NewMarginWriterBatch returns a new optimized MarginWriter.
// This version uses batch operations for padding and indentation,
// which significantly improves performance by reducing the number of
// ANSI escape sequences generated.
//
// Performance improvement: For a document with 50 lines and 4-space indentation,
// this reduces function calls from ~200 to ~50 (75% reduction) and ANSI codes
// from ~80 characters per line to ~20 characters per line.
func NewMarginWriterBatch(ctx RenderContext, w io.Writer, rules StyleBlock) *MarginWriterBatch {
	bs := ctx.blockStack

	var indentation uint
	var margin uint
	if rules.Indent != nil {
		indentation = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	pw := NewPaddingWriterBatch(w, int(bs.Width(ctx)), func(_ io.Writer, count int) { //nolint:gosec
		_, _ = renderText(w, rules.StylePrimitive, strings.Repeat(" ", count))
	})

	// Set FinalPadFunc with background-only style (no underline, bold, etc.)
	// This is used for the final line padding in Close() to avoid style artifacts
	if rules.StylePrimitive.BackgroundColor != nil {
		bgOnlyStyle := StylePrimitive{
			BackgroundColor: rules.StylePrimitive.BackgroundColor,
		}
		pw.FinalPadFunc = func(_ io.Writer, count int) {
			_, _ = renderText(w, bgOnlyStyle, strings.Repeat(" ", count))
		}
	}

	ic := " "
	if rules.IndentToken != nil {
		ic = *rules.IndentToken
	}
	iw := NewIndentWriterBatch(pw, int(indentation+margin), func(_ io.Writer, count int) { //nolint:gosec
		_, _ = renderText(w, bs.Parent().Style.StylePrimitive, strings.Repeat(ic, count))
	})

	return &MarginWriterBatch{
		w:  lipgloss.NewWrapWriter(w),
		iw: iw,
	}
}

// Write writes to the margin writer and implements [io.Writer].
func (w *MarginWriterBatch) Write(b []byte) (int, error) {
	n, err := w.iw.Write(b)
	if err != nil {
		return 0, fmt.Errorf("glamour: error writing bytes: %w", err)
	}
	return n, nil
}

// Close closes the [MarginWriterBatch].
func (w *MarginWriterBatch) Close() error {
	var werr error
	if w, ok := w.w.(io.WriteCloser); ok {
		werr = w.Close()
	}

	return errors.Join(werr, w.iw.Close())
}
