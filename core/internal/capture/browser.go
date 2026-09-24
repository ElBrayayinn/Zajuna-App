package capture

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mxschmitt/playwright-go"
)

type Runtime struct {
	Root        string
	DriverDir   string
	BrowsersDir string
}

type CaptureResult struct {
	Title           string
	FinalURL        string
	Selector        string
	SelectorMatched bool
	// RowsTotal/RowStart describe a row-batched capture: RowsTotal is the
	// number of counted rows in the container and RowStart the 1-based index
	// of the first row in the shot. Both are 0 when the capture was not batched.
	RowsTotal int
	RowStart  int
}

var ErrBlockedPage = errors.New("la página destino fue bloqueada por el sitio remoto")
var ErrLoginPage = errors.New("la página destino es la pantalla de autenticación de Zajuna")
var ErrChallengePage = errors.New("la página destino pide CAPTCHA o MFA")
var ErrSelectorNotFound = errors.New("el selector requerido no apareció en la página destino")

// ErrNoRowsInBatch means the requested row batch starts after the last row:
// the list is already fully covered by earlier slots, so there is nothing to
// capture. Callers must treat it as "slot not needed", not as a failure.
var ErrNoRowsInBatch = errors.New("no quedan filas para este lote")

const browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type CaptureOptions struct {
	Selector        string
	Selectors       []string
	RevealSelectors []string
	HideSelectors   []string
	ViewportWidth   int
	ViewportHeight  int
	FullPage        bool
	LabelHint       string
	OwnerName       string
	Timeout         time.Duration
	RequireSelector bool
	OwnerOnly       bool
	// RowSelector/RowsPerShot/RowBatch crop list and table evidence to one
	// batch of rows (see checklist.CaptureTarget). RowBatch is zero-based.
	RowSelector string
	RowsPerShot int
	RowBatch    int
	// OptionalSlot marks a slot that may legitimately not exist (e.g. the
	// 5th "Grabaciones" section of a course with 4 phases): when no selector
	// matches, the capture returns ErrNoRowsInBatch ("not needed") instead of
	// a failure.
	OptionalSlot bool
}

// BrowserCookie is the cookie shape needed to bridge an authenticated HTTP
// session into an isolated Chromium context. It is never persisted.
type BrowserCookie struct {
	Name     string
	Value    string
	URL      string
	Domain   string
	Path     string
	Secure   bool
	HttpOnly bool
	SameSite string
	Expires  *time.Time
}

func Resolve(executablePath string) Runtime {
	if configured := os.Getenv("ZAJUNA_PLAYWRIGHT_DIR"); configured != "" {
		return newRuntime(configured)
	}
	if executablePath == "" {
		executablePath, _ = os.Executable()
	}
	return newRuntime(filepath.Join(filepath.Dir(executablePath), "playwright"))
}

func newRuntime(root string) Runtime {
	return Runtime{
		Root:        root,
		DriverDir:   filepath.Join(root, "driver"),
		BrowsersDir: filepath.Join(root, "browsers"),
	}
}

func (r Runtime) Installed() bool {
	if r.Root == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(r.DriverDir, "package", "cli.js")); err != nil {
		return false
	}
	entries, err := os.ReadDir(r.BrowsersDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}
	return false
}

func (r Runtime) Start() (*playwright.Playwright, error) {
	if !r.Installed() {
		return nil, fmt.Errorf("runtime Chromium no instalado en %s; ejecuta npm run browser:install", r.Root)
	}
	var instance *playwright.Playwright
	err := withPlaywrightEnv(r, func() error {
		var runErr error
		instance, runErr = playwright.Run(&playwright.RunOptions{
			DriverDirectory:     r.DriverDir,
			Browsers:            []string{"chromium"},
			SkipInstallBrowsers: true,
			Verbose:             false,
		})
		if runErr != nil {
			return fmt.Errorf("iniciar Playwright local: %w", runErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// playwrightEnvMu serializes the process-wide PLAYWRIGHT_* variables with
// the driver launch that inherits them. Parallel checklist sessions start
// drivers concurrently; without it one launch could inherit another
// runtime's paths between Setenv and exec.
var playwrightEnvMu sync.Mutex

func withPlaywrightEnv(r Runtime, run func() error) error {
	playwrightEnvMu.Lock()
	defer playwrightEnvMu.Unlock()
	for name, value := range map[string]string{"PLAYWRIGHT_DRIVER_PATH": r.DriverDir, "PLAYWRIGHT_BROWSERS_PATH": r.BrowsersDir} {
		if os.Getenv(name) == value {
			continue
		}
		if err := os.Setenv(name, value); err != nil {
			return fmt.Errorf("configurar %s de Playwright: %w", name, err)
		}
	}
	return run()
}

func (r Runtime) CaptureURL(ctx context.Context, targetURL, outputPath string) error {
	_, err := r.CaptureURLWithMetadata(ctx, targetURL, outputPath)
	return err
}

func (r Runtime) CaptureURLWithMetadata(ctx context.Context, targetURL, outputPath string) (CaptureResult, error) {
	return r.CaptureURLWithMetadataAndCookies(ctx, targetURL, outputPath, nil)
}

func (r Runtime) CaptureURLWithMetadataAndCookies(ctx context.Context, targetURL, outputPath string, cookies []BrowserCookie) (CaptureResult, error) {
	return r.CaptureURLWithMetadataAndCookiesAndOptions(ctx, targetURL, outputPath, cookies, CaptureOptions{})
}

func (r Runtime) CaptureURLWithMetadataAndCookiesAndOptions(ctx context.Context, targetURL, outputPath string, cookies []BrowserCookie, options CaptureOptions) (CaptureResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Host == "" {
		return CaptureResult{}, errors.New("la URL de captura debe ser http o https")
	}
	if err := ValidateCaptureNavigationURL(targetURL, parsed); err != nil {
		return CaptureResult{}, err
	}
	if outputPath == "" {
		return CaptureResult{}, errors.New("la ruta de salida de captura es obligatoria")
	}
	if err := ctx.Err(); err != nil {
		return CaptureResult{}, err
	}
	absoluteOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return CaptureResult{}, fmt.Errorf("resolver salida de captura: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absoluteOutput), 0o700); err != nil {
		return CaptureResult{}, fmt.Errorf("crear carpeta de captura: %w", err)
	}
	pw, err := r.Start()
	if err != nil {
		return CaptureResult{}, err
	}
	defer pw.Stop()
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if err != nil {
		return CaptureResult{}, fmt.Errorf("lanzar Chromium local: %w", err)
	}
	defer browser.Close()
	browserContext, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(browserUserAgent),
		Locale:    playwright.String("es-CO"),
	})
	if err != nil {
		return CaptureResult{}, fmt.Errorf("crear contexto Chromium: %w", err)
	}
	defer browserContext.Close()
	if err := installNetworkPolicy(browserContext, parsed); err != nil {
		return CaptureResult{}, fmt.Errorf("configurar política de red de Chromium: %w", err)
	}
	if len(cookies) > 0 {
		browserCookies := make([]playwright.OptionalCookie, 0, len(cookies))
		for _, cookie := range cookies {
			item, ok := cookie.playwrightCookie()
			if !ok {
				continue
			}
			browserCookies = append(browserCookies, item)
		}
		if len(browserCookies) > 0 {
			if err := browserContext.AddCookies(browserCookies); err != nil {
				return CaptureResult{}, fmt.Errorf("cargar sesión autenticada en Chromium: %w", err)
			}
		}
	}
	page, err := browserContext.NewPage()
	if err != nil {
		return CaptureResult{}, fmt.Errorf("crear página Chromium: %w", err)
	}
	return capturePage(ctx, page, targetURL, absoluteOutput, options)
}

func capturePage(ctx context.Context, page playwright.Page, targetURL, absoluteOutput string, options CaptureOptions) (CaptureResult, error) {
	parsedTarget, err := url.Parse(targetURL)
	if err != nil || parsedTarget.Host == "" {
		return CaptureResult{}, errors.New("la URL de captura debe ser http o https")
	}
	if err := ValidateCaptureNavigationURL(targetURL, parsedTarget); err != nil {
		return CaptureResult{}, err
	}
	if options.ViewportWidth > 0 && options.ViewportHeight > 0 {
		if err := page.SetViewportSize(options.ViewportWidth, options.ViewportHeight); err != nil {
			return CaptureResult{}, fmt.Errorf("configurar viewport de captura: %w", err)
		}
	}
	if _, err := page.Goto(targetURL); err != nil {
		return CaptureResult{}, fmt.Errorf("navegar para captura: %w", err)
	}
	_ = page.WaitForLoadState()
	title, _ := page.Title()
	finalURL := page.URL()
	if err := ValidateCaptureNavigationURL(finalURL, parsedTarget); err != nil {
		return CaptureResult{}, fmt.Errorf("captura bloqueada: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return CaptureResult{}, err
	}
	body, _ := page.Locator("body").InnerText()
	if isZajunaLoginPage(finalURL, title, body) {
		return CaptureResult{}, fmt.Errorf("%w: URL final %s", ErrLoginPage, finalURL)
	}
	if reason := pageChallengeReason(page, finalURL, title, body); reason != "" {
		return CaptureResult{}, fmt.Errorf("%w: %s en URL final %s", ErrChallengePage, reason, finalURL)
	}
	if blocked, blockedErr := isBlockedPage(page, title); blocked {
		return CaptureResult{}, blockedErr
	}
	prepareCourseMenu(page, options)
	prepareHiddenEvidenceRegions(page, options.HideSelectors)
	// Moodle flash notifications (e.g. "No dispone de permiso para ver los
	// debates de este foro") must never leak into evidence.
	prepareHiddenEvidenceRegions(page, moodleNotificationSelectors)
	result := CaptureResult{Title: title, FinalURL: finalURL}
	batching := strings.TrimSpace(options.RowSelector) != "" && options.RowsPerShot > 0
	selectors := make([]string, 0, len(options.Selectors)+1)
	addSelector := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range selectors {
			if existing == value {
				return
			}
		}
		selectors = append(selectors, value)
	}
	addSelector(options.Selector)
	for _, selector := range options.Selectors {
		addSelector(selector)
	}
	matchedCandidates := 0
	emptyPrimaryContainer := false
	lastScreenshotError := ""
	selectorDiagnostics := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		locator := page.Locator(selector)
		rawCount, _ := locator.Count()
		if hint := strings.TrimSpace(options.LabelHint); hint != "" {
			filtered := locator.Filter(playwright.LocatorFilterOptions{HasText: hint})
			count, countErr := filtered.Count()
			if countErr != nil || count == 0 {
				selectorDiagnostics = append(selectorDiagnostics, fmt.Sprintf("%s raw=%d hint=0", selector, rawCount))
				// Never use the first generic node when a semantic hint does not
				// match; that would produce unrelated evidence.
				continue
			}
			locator = filtered
		}
		if options.OwnerOnly {
			ownerName := strings.TrimSpace(options.OwnerName)
			if ownerName == "" {
				continue
			}
			// When batching rows, the owner filter applies to rows inside the
			// container (see captureRowBatch), not to the container itself.
			if !batching {
				filtered := locator.Filter(playwright.LocatorFilterOptions{HasText: ownerName})
				count, countErr := filtered.Count()
				if countErr != nil || count == 0 {
					selectorDiagnostics = append(selectorDiagnostics, fmt.Sprintf("%s raw=%d owner=0", selector, rawCount))
					continue
				}
				locator = filtered
			}
		}
		if options.OwnerOnly && !batching {
			selectorDiagnostics = append(selectorDiagnostics, fmt.Sprintf("%s raw=%d owner=%d", selector, rawCount, func() int { value, _ := locator.Count(); return value }()))
		}
		if count, countErr := locator.Count(); countErr == nil && count > 0 {
			matchedCandidates += count
			timeout := 5000.0
			if options.Timeout > 0 {
				timeout = float64(options.Timeout.Milliseconds())
			}
			if batching {
				batch, batchErr := captureRowBatch(page, locator.First(), absoluteOutput, timeout, options)
				if errors.Is(batchErr, ErrNoRowsInBatch) && batch.total == 0 {
					// This candidate has no rows (e.g. a per-row fallback such as
					// `.discussion`). Only a container that actually counted rows
					// may declare the batch empty; keep looking.
					selectorDiagnostics = append(selectorDiagnostics, fmt.Sprintf("%s raw=%d rows=0", selector, rawCount))
					if selector == strings.TrimSpace(options.Selector) {
						emptyPrimaryContainer = true
					}
					continue
				}
				if batch.handled {
					selectorDiagnostics = append(selectorDiagnostics, fmt.Sprintf("%s raw=%d rows=%d", selector, rawCount, batch.total))
					if batchErr == nil {
						result.Selector = selector
						result.SelectorMatched = true
						result.RowsTotal = batch.total
						result.RowStart = batch.start + 1
						return result, nil
					}
					// Rows were counted here, so this container is the list:
					// "no more rows" is definitive, and a screenshot error must
					// surface as a failure instead of becoming an empty batch on
					// a fallback selector (which would prune good evidence).
					return CaptureResult{}, batchErr
				}
				if batchErr != nil {
					lastScreenshotError = batchErr.Error()
					continue
				}
				// No rows at all on the first batch: fall back to the legacy
				// container capture, including the container-level owner filter.
				selectorDiagnostics = append(selectorDiagnostics, fmt.Sprintf("%s raw=%d rows=0", selector, rawCount))
				if selector == strings.TrimSpace(options.Selector) {
					emptyPrimaryContainer = true
				}
				if options.OwnerOnly {
					filtered := locator.Filter(playwright.LocatorFilterOptions{HasText: strings.TrimSpace(options.OwnerName)})
					if ownerCount, ownerErr := filtered.Count(); ownerErr != nil || ownerCount == 0 {
						selectorDiagnostics = append(selectorDiagnostics, fmt.Sprintf("%s raw=%d owner=0", selector, rawCount))
						continue
					}
					locator = filtered
				}
			}
			var captureErr error
			if options.FullPage {
				_, captureErr = page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(absoluteOutput), FullPage: playwright.Bool(true), Timeout: playwright.Float(timeout)})
			} else {
				expandCourseSection(locator.First(), timeout)
				_, captureErr = locator.First().Screenshot(playwright.LocatorScreenshotOptions{Path: playwright.String(absoluteOutput), Timeout: playwright.Float(timeout)})
			}
			if captureErr == nil {
				result.Selector = selector
				result.SelectorMatched = true
				return result, nil
			} else {
				lastScreenshotError = captureErr.Error()
			}
		}
	}
	if batching && options.RowBatch == 0 && options.OwnerOnly && emptyPrimaryContainer {
		// The list exists but none of its rows is by the instructor: a real
		// absence of evidence, reported in plain words.
		return CaptureResult{}, fmt.Errorf("%w: la lista no tiene publicaciones del instructor autenticado", ErrSelectorNotFound)
	}
	if batching && options.RowBatch > 0 && emptyPrimaryContainer {
		// The real list container rendered but holds no (owner) rows: later
		// batches are simply not needed. A missing container stays a failure.
		return CaptureResult{}, fmt.Errorf("%w: la lista no tiene filas", ErrNoRowsInBatch)
	}
	if options.OptionalSlot && matchedCandidates == 0 {
		return CaptureResult{}, fmt.Errorf("%w: el espacio opcional no existe en esta página", ErrNoRowsInBatch)
	}
	if options.RequireSelector {
		diagnostics := fmt.Sprintf("candidatos=%d", matchedCandidates)
		if len(selectorDiagnostics) > 0 {
			diagnostics += ", selectores=" + strings.Join(selectorDiagnostics, " | ")
		}
		if options.OwnerOnly {
			counts := make([]string, 0, 8)
			for _, diagnosticSelector := range []string{"#region-main", "#page-mod-forum-view", "#region-main table", ".forumheaderlist", ".discussion", ".forumpost", ".forum-post", "article"} {
				count, _ := page.Locator(diagnosticSelector).Count()
				counts = append(counts, fmt.Sprintf("%s=%d", diagnosticSelector, count))
			}
			diagnostics += ", DOM=" + strings.Join(counts, ",")
		}
		if lastScreenshotError != "" {
			diagnostics += fmt.Sprintf(", error de captura: %s", lastScreenshotError)
		}
		return CaptureResult{}, fmt.Errorf("%w: %s (%s)", ErrSelectorNotFound, strings.TrimSpace(options.Selector), diagnostics)
	}
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(absoluteOutput), FullPage: playwright.Bool(true)}); err != nil {
		return CaptureResult{}, fmt.Errorf("guardar captura: %w", err)
	}
	if len(selectors) > 0 {
		result.Selector = selectors[0]
	}
	return result, nil
}

func prepareHiddenEvidenceRegions(page playwright.Page, selectors []string) {
	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}
		_, _ = page.Evaluate(`(selector) => {
			for (const node of document.querySelectorAll(selector)) {
				node.setAttribute('data-zajuna-hidden-evidence', 'true');
				node.style.display = 'none';
			}
			return true;
		}`, selector)
	}
}

// expandCourseSectionScript opens a collapsed Moodle 4 course section (and
// its nested subsections) in place. Section content is already in the DOM as
// `.content.collapse` with display:none, so a capture of a collapsed section
// only showed its title bar. Nothing is clicked and nothing navigates.
const expandCourseSectionScript = `(section) => {
	if (!section.matches('li.section, [data-for="section"]')) {
		return false;
	}
	// Only the section's own panel: nested subsections stay collapsed so the
	// evidence shows the organization (their titles) instead of a page tens
	// of thousands of pixels tall. Items that need a subsection target it.
	const panels = [...section.querySelectorAll(':scope > .content.collapse, :scope > .course-content-item-content.collapse')];
	// A subsection stays invisible (and its screenshot times out) while any
	// ancestor section is collapsed: open the whole chain up to the course.
	for (let node = section.parentElement; node; node = node.parentElement) {
		if (node.classList && node.classList.contains('collapse')) {
			panels.push(node);
		}
	}
	for (const panel of panels) {
		panel.classList.add('show');
		panel.style.display = 'block';
		panel.style.height = 'auto';
	}
	for (const toggle of section.querySelectorAll(':scope > .course-section-header [data-toggle="collapse"][aria-expanded="false"]')) {
		toggle.classList.remove('collapsed');
		toggle.setAttribute('aria-expanded', 'true');
	}
	return panels.length > 0;
}`

func expandCourseSection(section playwright.Locator, timeout float64) {
	_, _ = section.Evaluate(expandCourseSectionScript, nil, playwright.LocatorEvaluateOptions{Timeout: playwright.Float(timeout)})
}

// moodleNotificationSelectors are flash/alert regions Moodle injects after a
// redirect (permission errors, session notices). They are never evidence.
var moodleNotificationSelectors = []string{
	"#user-notifications",
	"#region-main > .alert-dismissible",
	"#page-content .alert-block.alert-dismissible",
}

// rowBatchWindow returns the half-open row range [start, end) covered by the
// zero-based batch when total rows are split into shots of perShot rows. ok
// is false when the batch starts at or after the last row.
func rowBatchWindow(total, perShot, batch int) (start, end int, ok bool) {
	if total <= 0 || perShot <= 0 || batch < 0 {
		return 0, 0, false
	}
	start = batch * perShot
	if start >= total {
		return 0, 0, false
	}
	end = start + perShot
	if end > total {
		end = total
	}
	return start, end, true
}

// rowBatchHideScript counts rows inside the container and, when the batch
// window is not empty, hides every row outside it (and every non-owner row
// when ownerOnly). Counting and hiding happen in one evaluation so both use
// the same DOM snapshot. Returns {total}.
const rowBatchHideScript = `(container, args) => {
	const normalize = (value) => (value || '')
		.normalize('NFD')
		.replace(/[̀-ͯ]/g, '')
		.replace(/\s+/g, ' ')
		.trim()
		.toLowerCase();
	const owner = normalize(args.owner);
	const all = Array.from(container.querySelectorAll(args.rowSelector));
	const counted = [];
	const hide = [];
	for (const row of all) {
		// Prefer the author cell: a discussion started by someone else may
		// still show the instructor as its last poster.
		const author = row.querySelector('td.author, .author, [data-region="author-name"]');
		const ownerText = author ? author.textContent : row.textContent;
		if (args.ownerOnly && (owner === '' || !normalize(ownerText).includes(owner))) {
			hide.push(row);
		} else {
			counted.push(row);
		}
	}
	const total = counted.length;
	const start = args.batch * args.perShot;
	if (total === 0 || start >= total) {
		return { total };
	}
	const end = Math.min(start + args.perShot, total);
	counted.forEach((row, index) => {
		if (index < start || index >= end) {
			hide.push(row);
		}
	});
	for (const row of hide) {
		if (row.hasAttribute('data-zajuna-row-batch-hidden')) {
			continue;
		}
		row.setAttribute('data-zajuna-row-batch-hidden', 'true');
		row.setAttribute('data-zajuna-row-batch-display', row.style.display || '');
		row.style.display = 'none';
	}
	return { total };
}`

// rowBatchRestoreScript undoes rowBatchHideScript so later work on the same
// page sees the original rows.
const rowBatchRestoreScript = `() => {
	for (const row of document.querySelectorAll('[data-zajuna-row-batch-hidden]')) {
		row.style.display = row.getAttribute('data-zajuna-row-batch-display') || '';
		row.removeAttribute('data-zajuna-row-batch-hidden');
		row.removeAttribute('data-zajuna-row-batch-display');
	}
	return true;
}`

type rowBatchOutcome struct {
	// handled is true when rows were counted and the batch window was
	// captured (or its screenshot attempted). false with a nil error means
	// there were no rows on batch 0 and the caller must fall back.
	handled bool
	total   int
	start   int
}

// captureRowBatch screenshots only the requested batch of rows inside the
// container. It returns ErrNoRowsInBatch (without writing a file) when the
// batch starts after the last counted row.
func captureRowBatch(page playwright.Page, container playwright.Locator, absoluteOutput string, timeout float64, options CaptureOptions) (rowBatchOutcome, error) {
	raw, err := container.Evaluate(rowBatchHideScript, map[string]any{
		"rowSelector": strings.TrimSpace(options.RowSelector),
		"owner":       strings.TrimSpace(options.OwnerName),
		"ownerOnly":   options.OwnerOnly,
		"perShot":     options.RowsPerShot,
		"batch":       options.RowBatch,
	}, playwright.LocatorEvaluateOptions{Timeout: playwright.Float(timeout)})
	restore := func() { _, _ = page.Evaluate(rowBatchRestoreScript) }
	if err != nil {
		restore()
		return rowBatchOutcome{}, fmt.Errorf("contar filas del lote: %w", err)
	}
	total := evaluatedInt(raw, "total")
	start, _, ok := rowBatchWindow(total, options.RowsPerShot, options.RowBatch)
	if total == 0 && options.RowBatch == 0 {
		restore()
		return rowBatchOutcome{}, nil
	}
	if total == 0 {
		restore()
		return rowBatchOutcome{}, fmt.Errorf("%w: el contenedor no tiene filas", ErrNoRowsInBatch)
	}
	if !ok {
		restore()
		return rowBatchOutcome{handled: true, total: total}, fmt.Errorf("%w: lote %d con %d filas por captura, %d filas en total", ErrNoRowsInBatch, options.RowBatch+1, options.RowsPerShot, total)
	}
	_, captureErr := container.Screenshot(playwright.LocatorScreenshotOptions{Path: playwright.String(absoluteOutput), Timeout: playwright.Float(timeout)})
	restore()
	return rowBatchOutcome{handled: true, total: total, start: start}, captureErr
}

func evaluatedInt(raw any, key string) int {
	values, ok := raw.(map[string]any)
	if !ok {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	}
	return 0
}

// prepareCourseMenu gives Moodle's accordion sections a chance to render
// their activity cards before a semantic selector is evaluated. This is
// intentionally limited to explicit collapse controls inside the course
// content; it never clicks activity links or arbitrary page controls.
func prepareCourseMenu(page playwright.Page, options CaptureOptions) {
	selectors := append([]string{options.Selector}, options.Selectors...)
	needsCourseMenu := false
	for _, selector := range selectors {
		if strings.Contains(selector, ".course-content") || strings.Contains(selector, "activity-item") {
			needsCourseMenu = true
			break
		}
	}
	if !needsCourseMenu {
		return
	}
	_ = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateNetworkidle,
		Timeout: playwright.Float(5000),
	})
	if len(options.RevealSelectors) > 0 {
		for _, selector := range options.RevealSelectors {
			control := page.Locator(strings.TrimSpace(selector))
			if count, err := control.Count(); err == nil && count > 0 {
				expanded, _ := control.First().GetAttribute("aria-expanded")
				if strings.EqualFold(strings.TrimSpace(expanded), "false") {
					_ = control.First().Click(playwright.LocatorClickOptions{
						Force:   playwright.Bool(true),
						Timeout: playwright.Float(1000),
					})
				}
				_, _ = page.Evaluate(`(selector) => {
					const control = document.querySelector(selector);
					if (!control) return false;
					const targetID = control.getAttribute('aria-controls') || (control.getAttribute('href') || '').replace(/^#/, '');
					const target = targetID ? document.getElementById(targetID) : null;
					if (!target) return false;
					control.setAttribute('aria-expanded', 'true');
					control.classList.remove('collapsed');
					let node = target;
					while (node && node.id !== 'region-main') {
						node.classList.add('show');
						node.classList.remove('collapse');
						node.removeAttribute('hidden');
						if (getComputedStyle(node).display === 'none') node.style.display = 'block';
						if (getComputedStyle(node).visibility === 'hidden') node.style.visibility = 'visible';
						node = node.parentElement;
					}
					return true;
				}`, strings.TrimSpace(selector))
			}
		}
		page.WaitForTimeout(300)
		return
	}
	collapseSelectors := []string{
		"#region-main .course-content [data-toggle='collapse'][aria-expanded='false']",
		"#region-main .course-content a[aria-expanded='false']",
		"#region-main .course-content [aria-expanded='false'][role='button']",
		"#region-main .course-content [aria-expanded='false'].collapsed",
	}
	clicked := 0
	for round := 0; round < 16 && clicked < 40; round++ {
		openedOne := false
		for _, selector := range collapseSelectors {
			locator := page.Locator(selector)
			count, err := locator.Count()
			if err != nil || count == 0 {
				continue
			}
			for index := 0; index < count && clicked < 40; index++ {
				candidate := locator.Nth(index)
				visible, visibleErr := candidate.IsVisible()
				if visibleErr != nil || !visible {
					continue
				}
				if err := candidate.Click(playwright.LocatorClickOptions{
					Force:   playwright.Bool(true),
					Timeout: playwright.Float(500),
				}); err != nil {
					continue
				}
				clicked++
				openedOne = true
				break
			}
			if openedOne {
				break
			}
		}
		if !openedOne {
			break
		}
		page.WaitForTimeout(100)
	}
	page.WaitForTimeout(350)
}

func isBlockedPage(page playwright.Page, title string) (bool, error) {
	body, err := page.Locator("body").InnerText()
	if err != nil {
		return false, nil
	}
	content := strings.ToLower(strings.TrimSpace(title + "\n" + body))
	if !containsBlockedPageMarkers(content) {
		return false, nil
	}
	return true, fmt.Errorf("%w: respuesta WAF de Zajuna", ErrBlockedPage)
}

func containsBlockedPageMarkers(content string) bool {
	content = strings.ToLower(content)
	return strings.Contains(content, "web page blocked") ||
		(strings.Contains(content, "attack id:") && strings.Contains(content, "message id:"))
}

func (r Runtime) RenderHTMLToPDF(ctx context.Context, htmlContent, outputPath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if htmlContent == "" || outputPath == "" {
		return errors.New("HTML y ruta de PDF son obligatorios")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	absoluteOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolver salida PDF: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absoluteOutput), 0o700); err != nil {
		return fmt.Errorf("crear carpeta PDF: %w", err)
	}
	pw, err := r.Start()
	if err != nil {
		return err
	}
	defer pw.Stop()
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if err != nil {
		return fmt.Errorf("lanzar Chromium para PDF: %w", err)
	}
	defer browser.Close()
	browserContext, err := browser.NewContext()
	if err != nil {
		return fmt.Errorf("crear contexto PDF: %w", err)
	}
	defer browserContext.Close()
	page, err := browserContext.NewPage()
	if err != nil {
		return fmt.Errorf("crear página PDF: %w", err)
	}
	if err := page.SetContent(htmlContent); err != nil {
		return fmt.Errorf("preparar contenido PDF: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := page.PDF(playwright.PagePdfOptions{Path: playwright.String(absoluteOutput), Format: playwright.String("A4"), PrintBackground: playwright.Bool(true), Tagged: playwright.Bool(true)}); err != nil {
		return fmt.Errorf("guardar PDF: %w", err)
	}
	return nil
}

func (c BrowserCookie) playwrightCookie() (playwright.OptionalCookie, bool) {
	if strings.TrimSpace(c.Name) == "" {
		return playwright.OptionalCookie{}, false
	}
	item := playwright.OptionalCookie{
		Name:     c.Name,
		Value:    c.Value,
		HttpOnly: playwright.Bool(c.HttpOnly),
		Secure:   playwright.Bool(c.Secure),
	}
	if strings.TrimSpace(c.Domain) != "" && strings.TrimSpace(c.Path) != "" {
		item.Domain = playwright.String(c.Domain)
		item.Path = playwright.String(c.Path)
	} else if strings.TrimSpace(c.URL) != "" {
		item.URL = playwright.String(c.URL)
	} else {
		return playwright.OptionalCookie{}, false
	}
	if c.Expires != nil {
		item.Expires = playwright.Float(float64(c.Expires.Unix()))
	}
	switch strings.ToLower(strings.TrimSpace(c.SameSite)) {
	case "strict":
		item.SameSite = playwright.SameSiteAttributeStrict
	case "none":
		item.SameSite = playwright.SameSiteAttributeNone
	case "lax":
		item.SameSite = playwright.SameSiteAttributeLax
	}
	return item, true
}
