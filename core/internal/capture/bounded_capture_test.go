package capture

import (
	"context"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

func TestColumnWindowsEndOnColumnEdges(t *testing.T) {
	edges := make([]float64, 0, 30)
	for column := 1; column <= 30; column++ {
		edges = append(edges, 100+float64(column)*300)
	}
	windows := columnWindows(100, 9000, edges, 2560)
	if len(windows) != 4 {
		t.Fatalf("expected 4 windows of whole columns, got %#v", windows)
	}
	previous := 100.0
	for index, window := range windows {
		if window.start != previous || window.end-window.start > 2560 {
			t.Fatalf("window %d is not contiguous or exceeds the limit: %#v", index, window)
		}
		if int(window.end-100)%300 != 0 {
			t.Fatalf("window %d cuts a column: %#v", index, window)
		}
		previous = window.end
	}
	if previous != 9100 {
		t.Fatalf("windows do not cover the whole element: end=%v", previous)
	}
}

func TestColumnWindowsEdgeCases(t *testing.T) {
	if got := columnWindows(0, 1200, nil, 2560); len(got) != 1 || got[0].end != 1200 {
		t.Fatalf("a narrow element is one window: %#v", got)
	}
	if got := columnWindows(0, 0, nil, 2560); got != nil {
		t.Fatalf("an empty element has no windows: %#v", got)
	}
	// A single column wider than the limit is cut at the limit.
	got := columnWindows(0, 6000, []float64{6000}, 2560)
	if len(got) != 3 || got[0].end != 2560 || got[2].end != 6000 {
		t.Fatalf("an oversized column must be sliced: %#v", got)
	}
}

func TestForumAccessDeniedFromRedirectOrNotice(t *testing.T) {
	forum, _ := url.Parse("https://zajuna.sena.edu.co/zajuna/mod/forum/view.php?id=77&forceview=1")
	if !forumAccessDenied(forum, "https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080", "") {
		t.Fatal("a forum redirected to the course page must be denied")
	}
	if !forumAccessDenied(forum, forum.String(), "No dispone de permiso para ver los debates de este foro") {
		t.Fatal("the permission notice must be detected")
	}
	if forumAccessDenied(forum, forum.String(), "Anuncios del instructor") {
		t.Fatal("an accessible forum was flagged")
	}
	course, _ := url.Parse("https://zajuna.sena.edu.co/zajuna/course/view.php?id=41080")
	if forumAccessDenied(course, course.String(), "No dispone de permiso para ver los debates de este foro") {
		t.Fatal("only forum targets can be denied")
	}
}

func TestEmbeddedSheetFrameGuards(t *testing.T) {
	for raw, want := range map[string]bool{
		"https://docs.google.com/spreadsheets/d/e/abc/pubhtml?widget=true": true,
		"http://docs.google.com/spreadsheets/d/e/abc/pubhtml":              false,
		"https://evil.example/spreadsheets/d/e/abc":                        false,
		"https://docs.google.com/document/d/abc":                           false,
	} {
		if got := embeddedSheetFrameURL(raw); got != want {
			t.Fatalf("embeddedSheetFrameURL(%q) = %v, want %v", raw, got, want)
		}
	}
	if w, h := boundedSheetSize(90000, 90000); w != maxEmbeddedSheetWidth || h != maxEmbeddedSheetHeight {
		t.Fatalf("sheet size is not bounded: %d x %d", w, h)
	}
	if w, h := boundedSheetSize(0, 500); w != 0 || h != 0 {
		t.Fatalf("an unmeasured sheet must not be resized: %d x %d", w, h)
	}
}

func browserSmokePage(t *testing.T) (playwright.BrowserContext, playwright.Page) {
	t.Helper()
	if os.Getenv("ZAJUNA_RUN_BROWSER_SMOKE") != "1" {
		t.Skip("browser smoke disabled")
	}
	pw, err := Resolve("").Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pw.Stop() })
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = browser.Close() })
	browserContext, err := browser.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	page, err := browserContext.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	return browserContext, page
}

func pngSize(t *testing.T, path string) (int, int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	return config.Width, config.Height
}

func hasInk(t *testing.T, path string) bool {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			if r, g, b, _ := img.At(x, y).RGBA(); r < 0x8000 && g < 0x8000 && b < 0x8000 {
				return true
			}
		}
	}
	return false
}

// graderFixture mimics Moodle's grader: a scrolling wrapper around a table
// with a user column and 40 grade columns of 300 px (~12.000 px wide).
func graderFixture() string {
	var header, row strings.Builder
	for column := 1; column <= 40; column++ {
		header.WriteString(fmt.Sprintf(`<th class="header item" style="min-width:300px;max-width:300px;width:300px">GA1-AA%d-EV01</th>`, column))
		row.WriteString(`<td class="grade" style="width:300px">4,50</td>`)
	}
	rows := ""
	for student := 1; student <= 6; student++ {
		rows += fmt.Sprintf(`<tr class="userrow"><th class="header user">Aprendiz %d</th>%s</tr>`, student, row.String())
	}
	return `<html><body style="margin:0"><div id="region-main"><div class="gradeparent" style="overflow:auto;width:1000px">` +
		`<table class="gradereport-grader-table" style="border-collapse:collapse;table-layout:fixed"><thead><tr class="heading"><th class="header user">Nombre</th>` +
		header.String() + `</tr></thead><tbody>` + rows + `</tbody></table></div></div></body></html>`
}

func TestGraderColumnWindowsSmoke(t *testing.T) {
	_, page := browserSmokePage(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(graderFixture()))
	}))
	defer server.Close()
	options := CaptureOptions{
		Selector: "#region-main .gradereport-grader-table", RequireSelector: true,
		RowSelector: "tbody tr.userrow", RowsPerShot: 2, MaxWidth: 2560,
	}
	windows := 0
	for column := 0; ; column++ {
		options.ColumnBatch = column
		output := filepath.Join(t.TempDir(), fmt.Sprintf("slot-%d.png", column+1))
		result, err := capturePage(context.Background(), page, server.URL, output, options)
		if errors.Is(err, ErrNoRowsInBatch) {
			break
		}
		if err != nil {
			t.Fatalf("column window %d: %v", column+1, err)
		}
		width, height := pngSize(t, output)
		if width > 2560 || width < 300 || height <= 0 {
			t.Fatalf("column window %d is %dx%d, want at most 2560 px wide", column+1, width, height)
		}
		if !hasInk(t, output) {
			t.Fatalf("column window %d is blank: the scrolling wrapper clipped it", column+1)
		}
		if result.RowsTotal != 6 || result.ColumnWindows < 2 {
			t.Fatalf("unexpected batch metadata: %#v", result)
		}
		windows = result.ColumnWindows
		if column > 10 {
			t.Fatal("column windows never ran out")
		}
	}
	if windows < 5 {
		t.Fatalf("a ~12.000 px grader needs at least 5 windows of 2560 px, got %d", windows)
	}
}

func TestExpandEmbeddedSheetsSmoke(t *testing.T) {
	browserContext, page := browserSmokePage(t)
	var sheetRows strings.Builder
	for row := 1; row <= 120; row++ {
		sheetRows.WriteString(fmt.Sprintf(`<tr style="height:30px"><td>Fase %d</td><td>Actividad %d</td></tr>`, row, row))
	}
	sheet := `<html><body style="margin:0"><div id="sheets-viewport" style="height:100vh;overflow:auto"><div class="grid-container" style="height:100%;overflow:auto"><table class="waffle">` + sheetRows.String() + `</table></div></div></body></html>`
	// Google Sheets is served from memory: the smoke never leaves the machine.
	if err := browserContext.Route("https://docs.google.com/**", func(route playwright.Route) {
		_ = route.Fulfill(playwright.RouteFulfillOptions{Status: playwright.Int(200), ContentType: playwright.String("text/html"), Body: sheet})
	}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body style="margin:0"><div id="region-main"><div class="course-content"><h2>Cronograma</h2>` +
			`<iframe src="https://docs.google.com/spreadsheets/d/e/fixture/pubhtml?widget=true" style="width:800px;height:300px;border:0"></iframe></div></div></body></html>`))
	}))
	defer server.Close()
	capture := func(expand bool) int {
		output := filepath.Join(t.TempDir(), "cronograma.png")
		options := CaptureOptions{Selector: "#region-main .course-content", RequireSelector: true, ExpandEmbeddedSheets: expand}
		if _, err := capturePage(context.Background(), page, server.URL, output, options); err != nil {
			t.Fatal(err)
		}
		_, height := pngSize(t, output)
		return height
	}
	if height := capture(false); height >= 120*30 {
		t.Fatalf("fixture does not reproduce the viewport-only iframe: %d px", height)
	}
	if height := capture(true); height < 120*30 {
		t.Fatalf("the sheet was not expanded: capture is %d px tall, the grid needs %d", height, 120*30)
	}
}
