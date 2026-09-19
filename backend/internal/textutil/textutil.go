// Package textutil converts raw job-board HTML into readable plain text.
package textutil

import (
	"html"
	"regexp"
	"strings"
)

// blockTags are tags that should become a line break when flattening HTML;
// everything else (span, strong, a…) is unwrapped inline.
var blockTags = map[string]bool{
	"address": true, "article": true, "blockquote": true, "dd": true,
	"div": true, "dl": true, "dt": true, "fieldset": true, "figcaption": true,
	"figure": true, "footer": true, "form": true, "h1": true, "h2": true,
	"h3": true, "h4": true, "h5": true, "h6": true, "header": true,
	"hr": true, "li": true, "main": true, "nav": true, "ol": true, "p": true,
	"pre": true, "section": true, "table": true, "tbody": true, "td": true,
	"tfoot": true, "th": true, "thead": true, "tr": true, "ul": true,
}

var (
	scriptRe  = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script[^>]*>`)
	styleRe   = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style[^>]*>`)
	brRe      = regexp.MustCompile(`(?i)<br\s*/?>`)
	tagRe     = regexp.MustCompile(`(?s)<[^>]*>`)
	tagNameRe = regexp.MustCompile(`(?i)^<\s*/?\s*([a-z][a-z0-9]*)`)

	spaceRunRe = regexp.MustCompile(`[ \t\x{00a0}]+`)
	trailRe    = regexp.MustCompile(`(?m)[ \t]+\n`)
	leadRe     = regexp.MustCompile(`(?m)\n[ \t]+`)
	newlineRun = regexp.MustCompile(`\n{3,}`)
)

// HTMLToText flattens an HTML fragment (job descriptions from board APIs) into
// clean plain text: script/style blocks are dropped, block tags become line
// breaks so paragraphs stop mashing together, inline tags are unwrapped,
// entities like &nbsp; are decoded, and whitespace collapses so paragraphs
// stay separated by at most one blank line. Because the output is plain text,
// running it again changes nothing.
func HTMLToText(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = scriptRe.ReplaceAllString(s, "\n\n")
	s = styleRe.ReplaceAllString(s, "\n\n")
	s = brRe.ReplaceAllString(s, "\n")
	s = tagRe.ReplaceAllStringFunc(s, func(tag string) string {
		if m := tagNameRe.FindStringSubmatch(tag); m != nil && blockTags[strings.ToLower(m[1])] {
			return "\n\n"
		}
		return " "
	})

	// Decode &nbsp; &amp; &#39; … only after tags are gone, so a decoded "<"
	// can never turn back into markup. UnescapeString turns &nbsp; into a
	// non-breaking space; normalize it to a regular one.
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")

	s = spaceRunRe.ReplaceAllString(s, " ")
	s = trailRe.ReplaceAllString(s, "\n")
	s = leadRe.ReplaceAllString(s, "\n")
	s = newlineRun.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
