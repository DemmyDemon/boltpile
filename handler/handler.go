package handler

import (
	"net"
	"net/http"
	"strings"

	"github.com/DemmyDemon/boltpile/storage"
	"github.com/rs/zerolog/log"
)

const (
	MAX_SIZE_DEFAULT = 5242880 // in bytes, 5MB
	ACCESS_DENIED    = `{"error":"access denied", "success":false}`
	NOT_IMPLEMENTED  = `{"error":"not implemented", "success":false}`
	ENTRY_NOT_FOUND  = `{"error":"entry not found", "success":false}`
	REQUEST_WEIRD    = `{"error":"request too weird", "success":false}`
	CHILL_OUT        = `{"error":"you need to chill out", "success":false}`
	OOOPS            = `{"error":"we messed up on our end", "success":false}`
	SUCCESS          = `{"success":true, "size":%d, "entry":%q}`
)

func SendMessage(w http.ResponseWriter, statusCode int, messge string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(messge))
}

func DeterminePeer(config storage.Config, r *http.Request) string {
	remote := r.RemoteAddr
	peer, _, err := net.SplitHostPort(remote)
	if err != nil {
		log.Warn().Err(err).Msg("Splitting host and port from remote address is weird.")
		return remote
	}
	if config.ForwardHeader != "" {
		if forwardHeader := r.Header.Get(config.ForwardHeader); forwardHeader != "" {
			return forwardHeader
		}
	}
	return peer
}

func HasBearerToken(token string, r *http.Request) bool {
	if token == "" {
		return true
	}
	if token == "!" {
		return false
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}
	authType, candidateToken, found := strings.Cut(authHeader, " ")
	if !found {
		return false
	}
	if strings.ToLower(authType) != "bearer" {
		return false
	}
	return candidateToken == token
}
