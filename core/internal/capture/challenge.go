package capture

import (
	"strings"

	"github.com/mxschmitt/playwright-go"
)

// Challenge detection is deliberately structural.
//
// The previous implementation matched the bare substrings "captcha" and
// "recaptcha" against the page body. Zajuna course content legitimately talks
// about CAPTCHAs (a single real course exposes 577 routes, several of them
// technical activities), so that match aborted valid captures with
// zajuna_challenge_required. An embedded widget is now authoritative anywhere
// on the page, while a prose mention only counts inside an authentication
// context, where a human really is being challenged.

// captchaWidgetSelectors match the DOM a CAPTCHA provider injects. Any hit is
// conclusive: the page cannot be completed without a human.
var captchaWidgetSelectors = []string{
	".g-recaptcha",
	".h-captcha",
	".cf-turnstile",
	"[data-sitekey]",
	`textarea[name="g-recaptcha-response"]`,
	`textarea[name="h-captcha-response"]`,
	`iframe[src*="recaptcha" i]`,
	`iframe[src*="hcaptcha" i]`,
	`iframe[src*="turnstile" i]`,
	`script[src*="recaptcha" i]`,
	`script[src*="hcaptcha" i]`,
	`script[src*="challenges.cloudflare.com" i]`,
}

// oneTimeCodeSelectors match a second-factor prompt. Zajuna's learner login
// never renders one; if it appears, the flow needs a human.
var oneTimeCodeSelectors = []string{
	`input[autocomplete="one-time-code"]`,
	`input[name*="otp" i]`,
	`input[name*="totp" i]`,
	`input[name*="mfa" i]`,
	`input[name*="twofactor" i]`,
	`input[name*="two_factor" i]`,
	`input[name*="authcode" i]`,
	`input[name*="auth_code" i]`,
	`input[name*="verificationcode" i]`,
	`input[name*="verification_code" i]`,
	`input[id*="otp" i]`,
}

// detectPageChallenge reports the selector that proves a CAPTCHA or MFA widget
// is present, or "" when the live DOM carries no challenge.
func detectPageChallenge(page playwright.Page) string {
	if page == nil {
		return ""
	}
	for _, selector := range captchaWidgetSelectors {
		if locatorPresent(page, selector) {
			return selector
		}
	}
	for _, selector := range oneTimeCodeSelectors {
		if locatorPresent(page, selector) {
			return selector
		}
	}
	return ""
}

func locatorPresent(page playwright.Page, selector string) bool {
	count, err := page.Locator(selector).Count()
	return err == nil && count > 0
}

// challengeProseMarkers are only consulted inside an authentication context.
var challengeProseMarkers = []string{
	"g-recaptcha",
	"h-captcha",
	"hcaptcha",
	"recaptcha",
	"captcha",
	"turnstile",
	"autenticación de dos factores",
	"autenticacion de dos factores",
	"two-factor",
	"two factor",
	"código de verificación",
	"codigo de verificacion",
	"one-time-code",
}

// isChallengePage keeps the string-only signature used where a live page is not
// available. A prose marker counts only when the URL, title and body show an
// authentication context, so ordinary course content is never mistaken for a
// challenge.
func isChallengePage(rawURL, title, body string) bool {
	if !isAuthenticationContext(rawURL, title, body) {
		return false
	}
	content := strings.ToLower(rawURL + "\n" + title + "\n" + body)
	for _, marker := range challengeProseMarkers {
		if strings.Contains(content, marker) {
			return true
		}
	}
	return false
}

// isAuthenticationContext is true while Zajuna is still asking who the user is.
// The live learner login is served from the site root, so the URL alone is not
// enough and the visible login form text is also accepted.
func isAuthenticationContext(rawURL, title, body string) bool {
	lowerURL := strings.ToLower(rawURL)
	if strings.Contains(lowerURL, "/login") || strings.Contains(lowerURL, "login_user") || strings.Contains(lowerURL, "/auth/") {
		return true
	}
	return isZajunaLoginPage(rawURL, title, body)
}

// pageChallengeReason returns a non-empty diagnostic when the live page needs a
// human. The DOM check is authoritative; the prose check only adds coverage for
// a challenge rendered without a recognisable widget.
func pageChallengeReason(page playwright.Page, rawURL, title, body string) string {
	if selector := detectPageChallenge(page); selector != "" {
		return "widget " + selector
	}
	if isChallengePage(rawURL, title, body) {
		return "texto de desafío en la pantalla de autenticación"
	}
	return ""
}

// normalizeSpanish lowercases text and folds the accents Zajuna uses in its
// login labels. The live learner form is served from the site root, so its
// visible label text is the only reliable signal and it has to survive a page
// rendered without accents.
func normalizeSpanish(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, symbol := range strings.ToLower(value) {
		switch symbol {
		case 'á', 'à', 'ä', 'â':
			symbol = 'a'
		case 'é', 'è', 'ë', 'ê':
			symbol = 'e'
		case 'í', 'ì', 'ï', 'î':
			symbol = 'i'
		case 'ó', 'ò', 'ö', 'ô':
			symbol = 'o'
		case 'ú', 'ù', 'ü', 'û':
			symbol = 'u'
		case 'ñ':
			symbol = 'n'
		}
		builder.WriteRune(symbol)
	}
	return builder.String()
}
