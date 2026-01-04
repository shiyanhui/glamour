package ansi

import (
	"github.com/yuin/goldmark/ast"
)

// NewElementWithBackgroundFix returns an Element for the given node with background color fix.
// This is a wrapper around NewElement that replaces elements with enhanced versions
// that properly support background colors:
// - CodeBlockElement -> CodeBlockElementWithBackground (always, for proper indentation)
// - HeadingElement -> HeadingElementWithBackground (always, but optimized to skip if no BG)
// - ListItem -> fixes the Exiting newline issue when nested list is the last child
func (r *ANSIRenderer) NewElementWithBackgroundFix(node ast.Node, source []byte) Element {
	elem := r.NewElement(node, source)

	// Replace CodeBlockElement with CodeBlockElementWithBackground
	if codeBlock, ok := elem.Renderer.(*CodeBlockElement); ok {
		if r.context.options.Styles.CodeBlock.StylePrimitive.BackgroundColor != nil {
			elem.Renderer = &CodeBlockElementWithBackground{
				Code:     codeBlock.Code,
				Language: codeBlock.Language,
			}
		}
	}

	// Replace HeadingElement with HeadingElementWithBackground
	if heading, ok := elem.Renderer.(*HeadingElement); ok {
		if r.context.options.Styles.Document.StylePrimitive.BackgroundColor != nil {
			elem.Renderer = &HeadingElementWithBackground{
				Level: heading.Level,
				First: heading.First,
			}
		}
	}

	if heading, ok := elem.Finisher.(*HeadingElement); ok {
		if r.context.options.Styles.Document.StylePrimitive.BackgroundColor != nil {
			elem.Finisher = &HeadingElementWithBackground{
				Level: heading.Level,
				First: heading.First,
			}
		}
	}

	// Fix ListItem Exiting: when a ListItem's last child is a nested List,
	// the original code sets post="" which causes the next ListItem to render
	// on the same line. We fix this by ensuring post="\n" when there's a next sibling.
	if node.Kind() == ast.KindListItem {
		if node.NextSibling() != nil && elem.Exiting == "" {
			elem.Exiting = "\n"
		}
	}

	return elem
}
