package capture

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// The live learner login is served from the site root, so URL matching alone
// cannot recognise it. These cases lock in the contextual rule that keeps
// ordinary course content out of the challenge path.
func TestChallengeContextRules(t *testing.T) {
	cases := []struct {
		name    string
		rawURL  string
		title   string
		body    string
		wantHit bool
	}{
		{
			name:    "captcha announced on the login page",
			rawURL:  "https://zajuna.sena.edu.co/zajuna/login/index.php",
			title:   "Verificacion",
			body:    "Complete el reCAPTCHA para continuar",
			wantHit: true,
		},
		{
			name:    "captcha announced on the root login form",
			rawURL:  "https://zajuna.sena.edu.co/",
			title:   "Zajuna",
			body:    "Tipo de documento Numero de Documento Contrasena Iniciar sesion Complete el captcha",
			wantHit: true,
		},
		{
			name:    "course activity that explains captchas",
			rawURL:  "https://zajuna.sena.edu.co/zajuna/mod/assign/view.php?id=31",
			title:   "FASE 2 EJECUTAR - Seguridad web",
			body:    "Explique como un CAPTCHA y la autenticacion de dos factores protegen un formulario",
			wantHit: false,
		},
		{
			name:    "ordinary course page",
			rawURL:  "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080",
			title:   "Curso",
			body:    "Contenido del curso",
			wantHit: false,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isChallengePage(testCase.rawURL, testCase.title, testCase.body); got != testCase.wantHit {
				t.Fatalf("isChallengePage = %v, want %v", got, testCase.wantHit)
			}
		})
	}
}

// TestDetectPageChallengeReadsTheDOM proves the authoritative check works on
// markup that never reaches innerText, which is what the string heuristic could
// not see.
func TestDetectPageChallengeReadsTheDOM(t *testing.T) {
	if os.Getenv("ZAJUNA_RUN_BROWSER_SMOKE") != "1" {
		t.Skip("browser smoke disabled")
	}
	cases := []struct {
		name    string
		markup  string
		wantHit bool
	}{
		{
			name:    "recaptcha widget without visible text",
			markup:  `<div class="g-recaptcha" data-sitekey="fixture"></div>`,
			wantHit: true,
		},
		{
			name:    "recaptcha iframe",
			markup:  `<iframe src="https://www.google.com/recaptcha/api2/anchor"></iframe>`,
			wantHit: true,
		},
		{
			name:    "one-time code prompt",
			markup:  `<form><input autocomplete="one-time-code" name="code"></form>`,
			wantHit: true,
		},
		{
			name:    "course content that only mentions captchas",
			markup:  `<div id="region-main"><p>Compare reCAPTCHA v2 con hCaptcha.</p></div>`,
			wantHit: false,
		},
		{
			name:    "moodle login fields",
			markup:  `<form id="login__form-cursos"><input name="document"><input name="password"><input type="hidden" name="logintoken"></form>`,
			wantHit: false,
		},
	}

	runtime := Resolve(os.Getenv("ZAJUNA_PLAYWRIGHT_DIR"))
	pw, err := runtime.Start()
	if err != nil {
		t.Skipf("playwright unavailable: %v", err)
	}
	defer func() { _ = pw.Stop() }()
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if err != nil {
		t.Fatalf("launch chromium: %v", err)
	}
	defer func() { _ = browser.Close() }()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(`<html><body>` + testCase.markup + `</body></html>`))
			}))
			defer server.Close()
			page, err := browser.NewPage()
			if err != nil {
				t.Fatalf("new page: %v", err)
			}
			defer func() { _ = page.Close() }()
			if _, err := page.Goto(server.URL); err != nil {
				t.Fatalf("goto fixture: %v", err)
			}
			selector := detectPageChallenge(page)
			if testCase.wantHit && selector == "" {
				t.Fatal("the challenge widget was not detected in the DOM")
			}
			if !testCase.wantHit && selector != "" {
				t.Fatalf("ordinary markup was flagged as a challenge by %s", selector)
			}
		})
	}
}
