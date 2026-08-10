package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	capabilityCookieName = "zajuna_capability"
	maxAPIRequestBytes   = 32 << 20
)

func newCapabilityToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// protectLocalAPI adds a per-process capability cookie and rejects mutating
// requests that do not originate from this loopback application. The cookie
// is HttpOnly and SameSite=Strict, so React fetch calls inherit it without
// exposing the capability token to page scripts or URLs.
func protectLocalAPI(next http.Handler, capability string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			writeError(w, http.StatusBadRequest, errors.New("el core solo acepta solicitudes loopback"))
			return
		}
		if capability != "" {
			http.SetCookie(w, &http.Cookie{Name: capabilityCookieName, Value: capability, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 0})
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && isMutatingMethod(r.Method) {
			if capability != "" && !hasCapability(r, capability) {
				writeError(w, http.StatusForbidden, errors.New("la capacidad local no es válida o expiró"))
				return
			}
			if err := validateLocalRequestOrigin(r); err != nil {
				writeError(w, http.StatusForbidden, err)
				return
			}
			if err := validateRequestContentType(r); err != nil {
				writeError(w, http.StatusUnsupportedMediaType, err)
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxAPIRequestBytes)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func hasCapability(r *http.Request, capability string) bool {
	cookie, err := r.Cookie(capabilityCookieName)
	return err == nil && cookie.Value == capability
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isLoopbackHost(rawHost string) bool {
	host := rawHost
	if parsedHost, _, err := net.SplitHostPort(rawHost); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateLocalRequestOrigin(r *http.Request) error {
	if fetchSite := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))); fetchSite == "cross-site" {
		return errors.New("la solicitud cross-site no está permitida")
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return nil
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || !isLoopbackHost(parsed.Host) {
		return errors.New("el origen de la solicitud no es loopback")
	}
	if !sameHost(r.Host, parsed.Host) {
		return errors.New("el origen de la solicitud no coincide con el core local")
	}
	return nil
}

func sameHost(left, right string) bool {
	leftHost, leftPort, leftErr := net.SplitHostPort(left)
	rightHost, rightPort, rightErr := net.SplitHostPort(right)
	if leftErr != nil {
		leftHost, leftPort = left, ""
	}
	if rightErr != nil {
		rightHost, rightPort = right, ""
	}
	return strings.EqualFold(strings.Trim(leftHost, "[]"), strings.Trim(rightHost, "[]")) && leftPort == rightPort
}

func validateRequestContentType(r *http.Request) error {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(r.URL.Path, "/api/evidences/upload") {
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			return errors.New("la carga de evidencias requiere multipart/form-data")
		}
		return nil
	}
	// Empty-body actions (cancel, restore, create-backup, notification read)
	// do not need a media type. Any request carrying data must declare JSON.
	if r.ContentLength == 0 && len(r.TransferEncoding) == 0 {
		return nil
	}
	if !strings.HasPrefix(contentType, "application/json") {
		return errors.New("las mutaciones JSON requieren Content-Type application/json")
	}
	return nil
}
