package eurekaapi

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeBaseURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("base URL must include scheme and host: %s", baseURL)
	}
	p := strings.TrimRight(u.Path, "/")
	switch {
	case strings.EqualFold(p, "/eureka/v2"):
		u.Path = p
	case strings.EqualFold(p, "/eureka"):
		u.Path = p + "/v2"
	case p == "":
		// If no path, default to /eureka/v2
		u.Path = defaultBasePath
	default:
		// If custom path, leave as-is; caller takes responsibility.
		u.Path = p
	}
	return u.String(), nil
}
