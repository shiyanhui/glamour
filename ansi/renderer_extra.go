package ansi

import (
	"fmt"
	"io"

	east "github.com/yuin/goldmark-emoji/ast"
	"github.com/yuin/goldmark/ast"
	astext "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// RegisterFuncsWithBackgroundFix registers rendering functions with background color fix.
// This replaces the default RegisterFuncs and uses renderNodeWithBackgroundFix
// instead of renderNode to fix code block background color issues.
func (r *ANSIRenderer) RegisterFuncsWithBackgroundFix(reg renderer.NodeRendererFuncRegisterer) {
	// blocks
	reg.Register(ast.KindDocument, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindHeading, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindBlockquote, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindCodeBlock, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindFencedCodeBlock, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindHTMLBlock, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindList, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindListItem, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindParagraph, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindTextBlock, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindThematicBreak, r.renderNodeWithBackgroundFix)

	// inlines
	reg.Register(ast.KindAutoLink, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindCodeSpan, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindEmphasis, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindImage, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindLink, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindRawHTML, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindText, r.renderNodeWithBackgroundFix)
	reg.Register(ast.KindString, r.renderNodeWithBackgroundFix)

	// tables
	reg.Register(astext.KindTable, r.renderNodeWithBackgroundFix)
	reg.Register(astext.KindTableHeader, r.renderNodeWithBackgroundFix)
	reg.Register(astext.KindTableRow, r.renderNodeWithBackgroundFix)
	reg.Register(astext.KindTableCell, r.renderNodeWithBackgroundFix)

	// definitions
	reg.Register(astext.KindDefinitionList, r.renderNodeWithBackgroundFix)
	reg.Register(astext.KindDefinitionTerm, r.renderNodeWithBackgroundFix)
	reg.Register(astext.KindDefinitionDescription, r.renderNodeWithBackgroundFix)

	// footnotes
	reg.Register(astext.KindFootnote, r.renderNodeWithBackgroundFix)
	reg.Register(astext.KindFootnoteList, r.renderNodeWithBackgroundFix)
	reg.Register(astext.KindFootnoteLink, r.renderNodeWithBackgroundFix)
	reg.Register(astext.KindFootnoteBacklink, r.renderNodeWithBackgroundFix)

	// checkboxes
	reg.Register(astext.KindTaskCheckBox, r.renderNodeWithBackgroundFix)

	// strikethrough
	reg.Register(astext.KindStrikethrough, r.renderNodeWithBackgroundFix)

	// emoji
	reg.Register(east.KindEmoji, r.renderNodeWithBackgroundFix)
}

// renderNodeWithBackgroundFix is an optimized version of renderNode that fixes
// code block background color issues.
//
// The only difference from the original renderNode is that it uses
// NewElementWithBackgroundFix instead of NewElement, which replaces
// CodeBlockElement with CodeBlockElementWithBackground.
func (r *ANSIRenderer) renderNodeWithBackgroundFix(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	writeTo := io.Writer(w)
	bs := r.context.blockStack

	// children get rendered by their parent
	if isChild(node) {
		return ast.WalkContinue, nil
	}

	// Use NewElementWithBackgroundFix instead of NewElement
	e := r.NewElementWithBackgroundFix(node, source)
	if entering { //nolint: nestif
		// everything below the Document element gets rendered into a block buffer
		if bs.Len() > 0 {
			writeTo = io.Writer(bs.Current().Block)
		}

		_, _ = io.WriteString(writeTo, e.Entering)
		if e.Renderer != nil {
			err := e.Renderer.Render(writeTo, r.context)
			if err != nil {
				return ast.WalkStop, fmt.Errorf("glamour: error rendering: %w", err)
			}
		}
	} else {
		// everything below the Document element gets rendered into a block buffer
		if bs.Len() > 0 {
			writeTo = io.Writer(bs.Parent().Block)
		}

		// if we're finished rendering the entire document,
		// flush to the real writer
		if node.Type() == ast.TypeDocument {
			writeTo = w
		}

		if e.Finisher != nil {
			err := e.Finisher.Finish(writeTo, r.context)
			if err != nil {
				return ast.WalkStop, fmt.Errorf("glamour: error finishing render: %w", err)
			}
		}

		_, _ = io.WriteString(bs.Current().Block, e.Exiting)
	}

	return ast.WalkContinue, nil
}
