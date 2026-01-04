package glamour

import (
	"charm.land/glamour/v2/ansi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

const (
	highPriorityExtra = 200
)

// NewTermRendererWithBackgroundFix returns a new TermRenderer with code block background color fix.
// This is an optimized version of NewTermRenderer that uses RegisterFuncsWithBackgroundFix
// to properly render code block background colors.
func NewTermRendererWithBackgroundFix(options ...TermRendererOption) (*TermRenderer, error) {
	tr := &TermRenderer{
		md: goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				extension.DefinitionList,
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
		),
		ansiOptions: ansi.Options{
			WordWrap: defaultWidth,
		},
	}
	for _, o := range options {
		if err := o(tr); err != nil {
			return nil, err
		}
	}

	// Create ANSIRenderer with background fix
	ar := ansi.NewRenderer(tr.ansiOptions)

	// Create a custom renderer that uses RegisterFuncsWithBackgroundFix
	tr.md.SetRenderer(
		renderer.NewRenderer(
			renderer.WithNodeRenderers(
				util.Prioritized(&ansiRendererWithBackgroundFix{ar}, highPriorityExtra),
			),
		),
	)
	return tr, nil
}

// ansiRendererWithBackgroundFix wraps ANSIRenderer to use RegisterFuncsWithBackgroundFix
type ansiRendererWithBackgroundFix struct {
	*ansi.ANSIRenderer
}

// RegisterFuncs registers rendering functions with background color fix
func (r *ansiRendererWithBackgroundFix) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	r.ANSIRenderer.RegisterFuncsWithBackgroundFix(reg)
}
