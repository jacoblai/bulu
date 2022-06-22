package engine

import (
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"strings"
)

func (d *Engine) JwtAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.Config.JwtSecret == "none" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			auth = strings.Replace(auth, "Bearer ", "", -1)
		}
		if auth != "" {
			token, err := jwt.ParseWithClaims(auth, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(d.Config.JwtSecret), nil
			})
			if err == nil {
				if _, ok := token.Claims.(*jwt.StandardClaims); ok && token.Valid {
					next.ServeHTTP(w, r)
					return
				}
			}
		}
		w.Header().Set("WWW-Authenticate", "Bearer realm=Restricted")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})
}
