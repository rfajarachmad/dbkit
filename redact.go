package dbkit

import (
	"net/url"
	"strings"
)

// Redact returns dsn with its password replaced, for logging.
// Unparseable input is returned as "<invalid dsn>".
func Redact(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "<invalid dsn>"
		}
		return u.Redacted()
	}
	pairs, err := splitKeywords(dsn)
	if err != nil {
		return "<invalid dsn>"
	}
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		v := p.value
		if p.key == "password" {
			v = "xxxxx"
		}
		out = append(out, p.key+"="+quoteKeywordValue(v))
	}
	return strings.Join(out, " ")
}
