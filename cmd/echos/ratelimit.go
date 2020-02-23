package main

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

func limit() func(next http.Handler) http.Handler {
	limits := cache.New(cache.NoExpiration, cache.NoExpiration)

	return func(next http.Handler) http.Handler {
		h := func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			var limiter *rate.Limiter

			if v, ok := limits.Get(ip); ok {
				limiter = v.(*rate.Limiter)
			} else {
				limit := rate.Every(time.Millisecond * 10)
				limiter = rate.NewLimiter(limit, 150)
				limits.SetDefault(ip, limiter)
			}

			if !limiter.Allow() {
				render.Status(r, http.StatusTooManyRequests)
				render.DefaultResponder(w, r, render.M{})
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(h)
	}
}

func getClientIP(r *http.Request) string {
	clientIP := r.Header.Get("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	}

	if clientIP == "" {
		if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
			clientIP = ip
		}
	}

	return clientIP
}

func realIP(next http.Handler) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = getClientIP(r)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(h)
}
