package stager

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

//go:generate counterfeiter . URIEncoder
type URIEncoder interface {
	Encode(string) string
}

type HostnameEncoder struct {
	Replacement string
}

func (e *HostnameEncoder) Encode(url string) string {
	re := regexp.MustCompile(`@(.*)?:`)

	hostname := stripScheme(e.Replacement)
	replaceWith := fmt.Sprintf("@%s:", hostname)
	return re.ReplaceAllString(url, replaceWith)
}

func stripScheme(providedURI string) string {
	if strings.HasPrefix(providedURI, "http://") || strings.HasPrefix(providedURI, "https://") {
		u, _ := url.Parse(providedURI)
		return u.Host
	}
	return providedURI
}
