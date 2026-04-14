package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

// ── Argon2id ──────────────────────────────────────────────────────────────────
// Parâmetros OWASP 2024: time=3, memory=64MB, threads=4

const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

// HashPassword gera um hash Argon2id no formato PHC.
// Exemplo: $argon2id$v=19$m=65536,t=3,p=4$<base64-salt>$<base64-hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword compara senha com hash PHC em tempo constante.
func VerifyPassword(password, encodedHash string) error {
	parts := strings.Split(encodedHash, "$")
	// formato: ["", "argon2id", "v=19", "m=...,t=...,p=...", "<salt>", "<hash>"]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return errors.New("invalid hash format")
	}
	var m, t uint32
	var p uint8
	if n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || n != 3 || m == 0 || t == 0 || p == 0 {
		return errors.New("invalid hash parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return err
	}
	inputHash := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(storedHash)))
	if subtle.ConstantTimeCompare(inputHash, storedHash) != 1 {
		return errors.New("password mismatch")
	}
	return nil
}

// ── JWT ───────────────────────────────────────────────────────────────────────

type contextKey string

const ctxUserEmail contextKey = "user_email"

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Server) GenerateToken(email string) (string, error) {
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *Server) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return token.Claims.(*Claims), nil
}

// ── Handlers ──────────────────────────────────────────────────────────────────

const cookieName = "telm_session"

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := s.store.FindUserByEmail(r.Context(), body.Email)
	if err != nil {
		jsonErr(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := VerifyPassword(body.Password, user.Password); err != nil {
		jsonErr(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := s.GenerateToken(user.Email)
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24h
		HttpOnly: true,
		Secure:   os.Getenv("HTTPS") == "true",
		SameSite: http.SameSiteStrictMode,
	})

	jsonOK(w, map[string]string{"email": user.Email})
}

func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("HTTPS") == "true",
		SameSite: http.SameSiteStrictMode,
	})
	jsonOK(w, map[string]string{"ok": "true"})
}

func (s *Server) HandleMe(w http.ResponseWriter, r *http.Request) {
	email, _ := r.Context().Value(ctxUserEmail).(string)
	jsonOK(w, map[string]string{"email": email})
}

func (s *Server) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			jsonErr(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := s.ValidateToken(cookie.Value)
		if err != nil {
			jsonErr(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserEmail, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
