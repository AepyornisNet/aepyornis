package templatehelpers

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var replyBasePattern = `(h-card|h-entry|h-event|h-adr|h-feed|p-name|p-summary|p-content|u-url|` +
	`u-photo|dt-published|dt-updated|e-content|mention|hashtag|ellipsis|invisible|` +
	`h-[a-zA-Z0-9_-]+|p-[a-zA-Z0-9_-]+|u-[a-zA-Z0-9_-]+|dt-[a-zA-Z0-9_-]+|e-[a-zA-Z0-9_-]+)`

var replyClassPattern = regexp.MustCompile(
	"^" + replyBasePattern + `(\s+` + replyBasePattern + `)*$`,
)

var replyPolicy = buildReplySanitizerPolicy()

func SanitizeReplyHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	return strings.TrimSpace(replyPolicy.Sanitize(raw))
}

func buildReplySanitizerPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.AllowElements("p", "span", "br", "a", "del", "pre", "code", "em", "strong", "b", "i", "u", "ul", "ol", "li", "blockquote")
	p.AllowAttrs("class").Matching(replyClassPattern).OnElements("span", "a")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("rel").OnElements("a")
	p.AllowAttrs("start", "reversed").OnElements("ol")
	p.AllowAttrs("value").OnElements("li")

	p.AllowURLSchemes("http", "https", "dat", "dweb", "ipfs", "ipns", "ssb", "gopher", "xmpp", "magnet", "gemini")
	p.RequireParseableURLs(true)
	p.AllowRelativeURLs(true)

	return p
}
