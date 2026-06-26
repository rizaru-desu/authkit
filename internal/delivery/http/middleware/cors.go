package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS implements Better Auth-style cross-origin handling:
//   - reflects the request Origin only when it matches a trusted pattern,
//     with credentials enabled (cookies cannot use a wildcard origin);
//   - rejects state-changing requests carrying an untrusted Origin header
//     (CSRF origin validation).
//
// Trusted patterns support wildcards: "*" (one label), "**" (any), "?" (one char).
func CORS(trustedOrigins []string) gin.HandlerFunc {
	matchers := compileOrigins(trustedOrigins)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin != "" && originMatches(matchers, origin)

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Max-Age", "600")
		}
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// CSRF: a browser request that mutates state must come from a trusted
		// origin. Requests without an Origin header (mobile/API/server) pass —
		// they authenticate with Bearer, not cookies.
		if isUnsafeMethod(c.Request.Method) && origin != "" && !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "origin not allowed",
			})
			return
		}

		c.Next()
	}
}

func isUnsafeMethod(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func compileOrigins(patterns []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(originPatternToRegex(p)); err == nil {
			out = append(out, re)
		}
	}
	return out
}

func originMatches(matchers []*regexp.Regexp, origin string) bool {
	for _, re := range matchers {
		if re.MatchString(origin) {
			return true
		}
	}
	return false
}

// originPatternToRegex converts a Better Auth-style glob to an anchored regex.
func originPatternToRegex(pattern string) string {
	var b strings.Builder
	b.WriteByte('^')
	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '*':
			if i+1 < len(runes) && runes[i+1] == '*' {
				b.WriteString(".*") // ** = any, across labels
				i++
			} else {
				b.WriteString("[^.]*") // * = a single label
			}
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(runes[i])))
		}
	}
	b.WriteByte('$')
	return b.String()
}
