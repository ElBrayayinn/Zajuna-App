package zajuna

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// hashedFixturePassword mirrors the shape Zajuna returns in its login bridge:
// the raw password is replaced by a bcrypt hash that Moodle expects verbatim.
// The value here is synthetic on purpose: the hash the live site returns is a
// password equivalent and must never be committed, and only its shape matters
// for this test. The document number is fabricated for the same reason.
const hashedFixturePassword = "$2y$10$fixtureOnlyNotARealCredential000000000000000000000000"

// liveBridgeBody reproduces the auto-submitting form that lg.php returned on
// 2026-08-26, single-quoted attributes included.
const liveBridgeBody = `<script type='text/javascript'>
	function enviarForm(){ document.getElementById('loginForm').submit(); }
	enviarForm()
</script>
<body onLoad='enviarForm();'>
<form style='display:none;' id='loginForm' action='/zajuna/login/index.php' class='form' method='POST' autocomplete='nop'>
<input type='hidden' name='josso' value='ZmFrZUpvc3NvRml4dHVyZQ=='>
<input type='hidden' name='logintoken' value='fixtureLoginTokenNotFromLiveSite'>
<input type='hidden' id='typeDocument' name='typeDocument' value='CC'>
<input type='text' id='username' name='username' value='1234567'>
<input type='password' id='password' name='password' autocomplete='nop' value='` + hashedFixturePassword + `'>
<input type='submit' value='Submit'>
</form></body>`

// TestLoginCompletesLiveTwoStepBridgeFlow pins the contract observed against
// the live site on 2026-08-26: lg.php answers HTTP 200 with a hidden
// auto-submitting form, and Moodle only accepts the hashed password it carries.
func TestLoginCompletesLiveTwoStepBridgeFlow(t *testing.T) {
	var forwardedPassword, forwardedUsername, forwardedToken, forwardedJosso string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/controllers/login_user/lg.php":
			_ = r.ParseForm()
			if r.PostForm.Get("document") == "" || r.PostForm.Get("password") == "" {
				http.Error(w, "missing credentials", http.StatusBadRequest)
				return
			}
			w.Header().Set("Set-Cookie", "MoodleSessionsenazajunav=fixture-session; Path=/")
			_, _ = w.Write([]byte(liveBridgeBody))
		case r.URL.Path == "/zajuna/login/index.php" && r.Method == http.MethodPost:
			_ = r.ParseForm()
			forwardedPassword = r.PostForm.Get("password")
			forwardedUsername = r.PostForm.Get("username")
			forwardedToken = r.PostForm.Get("logintoken")
			forwardedJosso = r.PostForm.Get("josso")
			http.Redirect(w, r, "/zajuna/login/index.php?testsession=1487116", http.StatusSeeOther)
		case r.URL.Path == "/zajuna/login/index.php":
			_, _ = w.Write([]byte(`<html><body>redirigiendo</body></html>`))
		case r.URL.Path == "/zajuna/my/courses.php":
			_, _ = w.Write([]byte(`<html><body><div class="usermenu"><span class="usertext">NOMBRE APELLIDO FIXTURE</span></div>` +
				`<h1>Mis cursos</h1><a href="/zajuna/course/view.php?id=41080">Programa fixture (9999999)</a></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := newClient(server.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	session, err := client.Login(context.Background(), Credentials{DocumentType: "CC", Document: "1234567", Password: "raw-fixture-secret"})
	if err != nil {
		t.Fatalf("login through the live bridge shape failed: %v", err)
	}
	if forwardedPassword != hashedFixturePassword {
		t.Fatalf("the bridge password must win over the raw credential, got %q", forwardedPassword)
	}
	if forwardedUsername != "1234567" || forwardedToken == "" || forwardedJosso == "" {
		t.Fatalf("bridge fields were not forwarded: username=%q logintoken=%q josso=%q", forwardedUsername, forwardedToken, forwardedJosso)
	}
	if session.ProfileName != "NOMBRE APELLIDO FIXTURE" {
		t.Fatalf("profile name was not parsed from the authenticated page: %q", session.ProfileName)
	}
}

// TestChallengeDetectionIgnoresCourseProse is the regression that motivated the
// structural rewrite: one real course exposes 577 routes, and its content may
// discuss CAPTCHAs. Prose alone must never abort the flow.
func TestChallengeDetectionIgnoresCourseProse(t *testing.T) {
	body := `<html><head><title>Actividad de seguridad web</title></head><body>` +
		`<div id="region-main"><h2>FASE 2 EJECUTAR</h2>` +
		`<p>Explique como un CAPTCHA y la autenticacion de dos factores protegen un formulario.</p>` +
		`<p>Compare reCAPTCHA v2 con hCaptcha y con un codigo de verificacion por SMS.</p>` +
		`</div></body></html>`
	if looksLikeChallengePage(body) {
		t.Fatalf("course prose was misread as a challenge: %s", challengeReason(body))
	}
}

func TestChallengeDetectionFlagsProseOnlyOnTheLoginPage(t *testing.T) {
	body := `<html><body><form id="login__form-cursos" action="controllers/login_user/lg.php">` +
		`<input name="document"><input name="password"></form>` +
		`<p>Complete el captcha para continuar</p></body></html>`
	if !looksLikeChallengePage(body) {
		t.Fatal("a challenge announced on the login page must abort the flow")
	}
}

func TestChallengeDetectionFlagsWidgets(t *testing.T) {
	cases := map[string]string{
		"recaptcha v2":     `<div class="g-recaptcha" data-sitekey="fixture"></div>`,
		"recaptcha script": `<script src="https://www.google.com/recaptcha/api.js"></script>`,
		"hcaptcha":         `<div class="h-captcha" data-sitekey="fixture"></div>`,
		"turnstile":        `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script>`,
	}
	for name, markup := range cases {
		if !looksLikeChallengePage(`<html><body>` + markup + `</body></html>`) {
			t.Errorf("%s widget was not detected", name)
		}
	}
}

func TestChallengeDetectionFlagsOneTimeCodeFields(t *testing.T) {
	cases := map[string]string{
		"autocomplete": `<input autocomplete="one-time-code" name="code">`,
		"otp name":     `<input type="text" name="user_otp">`,
		"mfa id":       `<input type="text" id="mfa_token" name="token">`,
		"verification": `<input type="text" name="verification_code">`,
	}
	for name, markup := range cases {
		if !looksLikeChallengePage(`<html><body><form>` + markup + `</form></body></html>`) {
			t.Errorf("%s second-factor field was not detected", name)
		}
	}
}

// TestChallengeDetectionIgnoresOrdinaryMoodleInputs guards the second-factor
// scan against the fields the real login and Moodle pages already carry.
func TestChallengeDetectionIgnoresOrdinaryMoodleInputs(t *testing.T) {
	body := `<html><body><form id="login__form-cursos">` +
		`<input name="document"><input name="password"><input name="logintoken" type="hidden">` +
		`<input name="sesskey" type="hidden"><input name="typeDocument"></form></body></html>`
	if looksLikeChallengePage(body) {
		t.Fatalf("ordinary Moodle login fields were misread as a challenge: %s", challengeReason(body))
	}
}

// TestLoginReportsChallengeRatherThanHTTPStatus keeps the challenge diagnosis
// ahead of the generic status check, so the operator learns a human is needed.
func TestLoginReportsChallengeRatherThanHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/controllers/login_user/lg.php" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`<div class="g-recaptcha" data-sitekey="fixture"></div>`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client, err := newClient(server.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Login(context.Background(), Credentials{DocumentType: "CC", Document: "1", Password: "x"})
	if !errors.Is(err, ErrChallengeRequired) {
		t.Fatalf("a challenge served with HTTP 401 must report the challenge, got %v", err)
	}
	if strings.Contains(err.Error(), "401") {
		t.Fatalf("the challenge must not be reported as a plain HTTP failure: %v", err)
	}
}
