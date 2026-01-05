package conversation

import (
	"regexp"
	"strings"
)

// UserAgentInfo contains parsed browser and OS information
type UserAgentInfo struct {
	Browser         string
	BrowserVersion  string
	OS              string
	OSVersion       string
}

// ParseUserAgent parses a user agent string to extract browser and OS info
func ParseUserAgent(ua string) UserAgentInfo {
	info := UserAgentInfo{}

	if ua == "" {
		return info
	}

	// Parse browser
	info.Browser, info.BrowserVersion = parseBrowser(ua)

	// Parse OS
	info.OS, info.OSVersion = parseOS(ua)

	return info
}

// GetBrowserString returns a formatted browser string (e.g., "Chrome 120")
func (u UserAgentInfo) GetBrowserString() string {
	if u.Browser == "" {
		return ""
	}
	if u.BrowserVersion != "" {
		return u.Browser + " " + u.BrowserVersion
	}
	return u.Browser
}

// GetOSString returns a formatted OS string (e.g., "Windows 10")
func (u UserAgentInfo) GetOSString() string {
	if u.OS == "" {
		return ""
	}
	if u.OSVersion != "" {
		return u.OS + " " + u.OSVersion
	}
	return u.OS
}

// parseBrowser extracts browser name and version from user agent
func parseBrowser(ua string) (name, version string) {
	// Order matters - check more specific browsers first
	browsers := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"Edge", regexp.MustCompile(`Edg(?:e|A|iOS)?/(\d+(?:\.\d+)?)`)},
		{"Opera", regexp.MustCompile(`(?:OPR|Opera)[/ ](\d+(?:\.\d+)?)`)},
		{"Samsung Browser", regexp.MustCompile(`SamsungBrowser/(\d+(?:\.\d+)?)`)},
		{"UC Browser", regexp.MustCompile(`UCBrowser/(\d+(?:\.\d+)?)`)},
		{"Firefox", regexp.MustCompile(`Firefox/(\d+(?:\.\d+)?)`)},
		{"Chrome", regexp.MustCompile(`Chrome/(\d+(?:\.\d+)?)`)},
		{"Safari", regexp.MustCompile(`Version/(\d+(?:\.\d+)?).*Safari`)},
		{"IE", regexp.MustCompile(`(?:MSIE |rv:)(\d+(?:\.\d+)?)`)},
	}

	for _, b := range browsers {
		matches := b.pattern.FindStringSubmatch(ua)
		if len(matches) > 1 {
			return b.name, matches[1]
		}
	}

	// Fallback checks
	if strings.Contains(ua, "Safari") && !strings.Contains(ua, "Chrome") {
		return "Safari", ""
	}

	return "", ""
}

// parseOS extracts operating system name and version from user agent
func parseOS(ua string) (name, version string) {
	// Check for mobile first
	if strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") {
		// iOS
		pattern := regexp.MustCompile(`OS (\d+[_\.]\d+(?:[_\.]\d+)?)`)
		matches := pattern.FindStringSubmatch(ua)
		if len(matches) > 1 {
			ver := strings.ReplaceAll(matches[1], "_", ".")
			return "iOS", ver
		}
		return "iOS", ""
	}

	if strings.Contains(ua, "Android") {
		pattern := regexp.MustCompile(`Android (\d+(?:\.\d+)?)`)
		matches := pattern.FindStringSubmatch(ua)
		if len(matches) > 1 {
			return "Android", matches[1]
		}
		return "Android", ""
	}

	// Desktop OS
	if strings.Contains(ua, "Windows") {
		// Map Windows NT versions to marketing names
		if strings.Contains(ua, "Windows NT 10.0") {
			return "Windows", "10"
		}
		if strings.Contains(ua, "Windows NT 6.3") {
			return "Windows", "8.1"
		}
		if strings.Contains(ua, "Windows NT 6.2") {
			return "Windows", "8"
		}
		if strings.Contains(ua, "Windows NT 6.1") {
			return "Windows", "7"
		}
		if strings.Contains(ua, "Windows NT 6.0") {
			return "Windows", "Vista"
		}
		if strings.Contains(ua, "Windows NT 5.1") {
			return "Windows", "XP"
		}
		return "Windows", ""
	}

	if strings.Contains(ua, "Mac OS X") {
		pattern := regexp.MustCompile(`Mac OS X (\d+[_\.]\d+(?:[_\.]\d+)?)`)
		matches := pattern.FindStringSubmatch(ua)
		if len(matches) > 1 {
			ver := strings.ReplaceAll(matches[1], "_", ".")
			return "macOS", ver
		}
		return "macOS", ""
	}

	if strings.Contains(ua, "Linux") {
		if strings.Contains(ua, "Ubuntu") {
			return "Ubuntu", ""
		}
		if strings.Contains(ua, "Fedora") {
			return "Fedora", ""
		}
		if strings.Contains(ua, "Debian") {
			return "Debian", ""
		}
		return "Linux", ""
	}

	if strings.Contains(ua, "CrOS") {
		return "Chrome OS", ""
	}

	return "", ""
}
