package main

import (
	"fmt"
	"net"
	"net/http"
)

// Set secure headers to a response
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' cdnjs.cloudflare.com; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com")

		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}

// Log incoming requests to the info.log
func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.RequestURI)

		next.ServeHTTP(w, r)
	})
}

// Recover any panics occurred on the server 
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func()  {
			if err := recover(); err != nil{
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Remove all temporary files associated with a requested form
func (app *application) clearPostFormData(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		// clear contents
		if r.MultipartForm != nil {
			err := r.MultipartForm.RemoveAll()
			if err != nil {
				app.serverError(w, err)
				return
			}
		}
	})
}

// Allow only the PC that is running the server to serve the incoming request
func (app *application) thisPCOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check if incoming request is from this pc or not
		hostIPs := []string{
			"127.0.0.1",
			"::1",
			app.hostInfo.IPAddr.String(),
		}

		remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverError(w, err)
			return
		}

		found := false
		for _, hostIP := range hostIPs {
			if remoteIP == hostIP {
				found = true
			}
		}

		if !found {
			fmt.Printf("remote IP: %s\n", remoteIP)
			app.clientError(w, http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

