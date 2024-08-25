package handler

import (
	"net/http"
	"strings"

	"github.com/DemmyDemon/boltpile/storage"
	"github.com/rs/zerolog/log"
)

func GetList(eg storage.EntryGetter, config storage.Config, limiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pile := r.PathValue("pile")
		peer := DeterminePeer(config, r)

		if !limiter.Allow(peer) {
			log.Warn().Str("operation", "list").Str("pile", pile).Str("peer", peer).Msg("Hit the rate limit!")
			SendFailure(w, http.StatusTooManyRequests, "you need to chill out")
			return
		}

		pileConfig, err := config.Pile(pile)
		if err != nil {
			log.Error().Err(err).Str("operation", "list").Str("pile", pile).Str("peer", peer).Msg("Couldn't obtain pile config")
			SendFailure(w, http.StatusNotFound, "pile not found")
			return
		}
		if !HasBearerToken(pileConfig.POSTKey, r) {
			log.Warn().Str("operation", "list").Str("pile", pile).Str("peer", peer).Msg("Invalid or missing bearer token")
			SendFailure(w, http.StatusForbidden, "access denied")
			return
		}

		logEntry := log.Info().Str("operation", "list").Str("pile", pile).Str("peer", peer)
		sb := strings.Builder{}
		sb.WriteString("{")

	}
}
