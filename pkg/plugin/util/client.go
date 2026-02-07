// Package util implements utility and convenience methods for plugins. It is
// not intended for the main stash code to access.
package util

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"

	"github.com/stashapp/stash/pkg/plugin/common"
)

// NewHTTPClient creates an HTTP client configured with authentication
// connecting to the stash server using the provided server connection details.
func NewHTTPClient(provider common.StashServerConnection) *http.Client {
	portStr := strconv.Itoa(provider.Port)

	u, _ := url.Parse("http://" + provider.Host + ":" + portStr + "/api/v1")
	u.Scheme = provider.Scheme

	cookieJar, _ := cookiejar.New(nil)

	cookie := provider.SessionCookie
	if cookie != nil {
		cookieJar.SetCookies(u, []*http.Cookie{
			cookie,
		})
	}

	return &http.Client{
		Jar: cookieJar,
	}
}
