package managementhealth

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const truncationMarker = "...[TRUNCATED]"

var (
	ansiEscape      = regexp.MustCompile("\\x1b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\\x07]*(?:\\x07|\\x1b\\\\))")
	credentialField = regexp.MustCompile(`(?i)\b(authorization|api[-_]?key|access[-_]?token|refresh[-_]?token|oauth[-_]?token|client[-_]?secret|bot[-_]?secret|channel[-_]?token|password|passwd|secret|token)\s*[:=]\s*(?:bearer\s+)?["']?[^\s,"';]+`)
	bearerToken     = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/-]{8,}={0,2}`)
	providerSecret  = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{16,}|xox[baprs]-[A-Za-z0-9-]{16,}|AIza[A-Za-z0-9_-]{20,})\b`)
	jwtToken        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`)
	emailAddress    = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	accountField    = regexp.MustCompile(`(?i)\b(account|account_id|user|user_id|username|chat_id|channel_id|phone|email)\s*[:=]\s*["']?[^\s,"';]+`)
	windowsPath     = regexp.MustCompile(`(?i)(?:[a-z]:[\\/]|\\\\)[^\s"'<>]+`)
	fileURI         = regexp.MustCompile(`(?i)file://[^\s"'<>]+`)
	posixPath       = regexp.MustCompile(`(?:~[/\\]|/(?:Users|home|root|var|tmp|etc|opt|mnt|private)/)[^\s"'<>]+`)
)

func redactAndSanitize(value string) string {
	value = strings.ToValidUTF8(value, "�")
	value = ansiEscape.ReplaceAllString(value, "")
	value = credentialField.ReplaceAllString(value, `${1}=[REDACTED]`)
	value = bearerToken.ReplaceAllString(value, "Bearer [REDACTED]")
	value = providerSecret.ReplaceAllString(value, "[REDACTED]")
	value = jwtToken.ReplaceAllString(value, "[REDACTED]")
	value = emailAddress.ReplaceAllString(value, "[ACCOUNT]")
	value = accountField.ReplaceAllString(value, `${1}=[ACCOUNT]`)
	value = fileURI.ReplaceAllString(value, "[PATH]")
	value = windowsPath.ReplaceAllString(value, "[PATH]")
	value = posixPath.ReplaceAllString(value, "[PATH]")

	var out strings.Builder
	out.Grow(len(value))
	for _, r := range value {
		switch {
		case r == '\t':
			out.WriteByte(' ')
		case r == '<':
			out.WriteRune('‹')
		case r == '>':
			out.WriteRune('›')
		case unicode.IsControl(r) || unicode.In(r, unicode.Cf):
			// Drop terminal controls, bidi overrides and zero-width format text.
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func truncateUTF8(value string, maxBytes int) (string, bool) {
	if len(value) <= maxBytes {
		return value, false
	}
	if maxBytes <= len(truncationMarker) {
		return truncationMarker[:maxBytes], true
	}
	limit := maxBytes - len(truncationMarker)
	for limit > 0 && !utf8.RuneStart(value[limit]) {
		limit--
	}
	return value[:limit] + truncationMarker, true
}
