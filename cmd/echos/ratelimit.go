package main

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

func limit() func(next http.Handler) http.Handler {
	limits := cache.New(cache.NoExpiration, cache.NoExpiration)

	return func(next http.Handler) http.Handler {
		h := func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				render.Status(r, http.StatusInternalServerError)
				render.DefaultResponder(w, r, render.M{
					"error": err.Error(),
				})
				return
			}

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
