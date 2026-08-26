package zajuna

import "strings"

// Challenge detection over raw HTML.
//
// The previous implementation matched the bare substrings "captcha" and
// "recaptcha" anywhere in the response. Zajuna serves Moodle pages whose
// markup and course content can mention a CAPTCHA without asking for one, so
// that match risked aborting a valid login with zajuna_challenge_required.
// Markup that only a provider emits is now authoritative; prose only counts on
// a page that is still asking who the user is.

// captchaWidgetMarkers appear exclusively when a provider widget is embedded.
var captchaWidgetMarkers = []string{
	`class="g-recaptcha`,
	`class='g-recaptcha`,
	`g-recaptcha-response`,
	`class="h-captcha`,
	`class='h-captcha`,
	`h-captcha-response`,
	`class="cf-turnstile`,
	`class='cf-turnstile`,
	`data-sitekey`,
	`google.com/recaptcha/api`,
	`recaptcha/api.js`,
	`recaptcha/enterprise.js`,
	`hcaptcha.com/1/api.js`,
	`challenges.cloudflare.com/turnstile`,
}

// oneTimeCodeFieldMarkers name a second-factor field. Zajuna's learner login
// never renders one, so any of these means a human has to finish the flow.
var oneTimeCodeFieldMarkers = []string{
	"otp",
	"totp",
	"mfa",
	"twofactor",
	"two_factor",
	"two-factor",
	"authcode",
	"auth_code",
	"verificationcode",
	"verification_code",
	"codigoverificacion",
	"codigo_verificacion",
}

// challengeProseMarkers are only consulted on an authentication page.
var challengeProseMarkers = []string{
	"recaptcha",
	"hcaptcha",
	"captcha",
	"turnstile",
	"autenticación de dos factores",
	"autenticacion de dos factores",
	"two-factor",
	"two factor",
	"código de verificación",
	"codigo de verificacion",
}

// looksLikeChallengePage reports whether Zajuna is asking for a CAPTCHA or a
// second factor. Callers turn this into zajuna_challenge_required and never try
// to solve it.
func looksLikeChallengePage(body string) bool {
	return challengeReason(body) != ""
}

// challengeReason returns the marker that proved the challenge, for diagnostics.
func challengeReason(body string) string {
	lower := strings.ToLower(body)
	for _, marker := range captchaWidgetMarkers {
		if strings.Contains(lower, strings.ToLower(marker)) {
			return marker
		}
	}
	if field := oneTimeCodeField(body); field != "" {
		return "campo de segundo factor " + field
	}
	if !looksLikeLoginPage(body) {
		return ""
	}
	for _, marker := range challengeProseMarkers {
		if strings.Contains(lower, marker) {
			return marker
		}
	}
	return ""
}

// oneTimeCodeField finds an input that collects a one-time code. The name, id
// and autocomplete attributes are all checked because Moodle plugins differ.
func oneTimeCodeField(body string) string {
	for _, tag := range inputTagPattern.FindAllString(body, -1) {
		attributes := inputAttributes(tag)
		if strings.EqualFold(attributes["autocomplete"], "one-time-code") {
			return "autocomplete=one-time-code"
		}
		for _, key := range []string{"name", "id"} {
			value := strings.ToLower(attributes[key])
			if value == "" {
				continue
			}
			for _, marker := range oneTimeCodeFieldMarkers {
				if strings.Contains(value, marker) {
					return key + "=" + value
				}
			}
		}
	}
	return ""
}
