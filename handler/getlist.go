package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/DemmyDemon/boltpile/storage"
	"github.com/rs/zerolog/log"
)

func GetList(eg storage.PileGetter, config storage.Config, limiter *RateLimiter) http.HandlerFunc {
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
		if !HasBearerToken(pileConfig.ListKey, r) {
			log.Warn().Str("operation", "list").Str("pile", pile).Str("peer", peer).Msg("Invalid or missing bearer token")
			SendFailure(w, http.StatusForbidden, "access denied")
			return
		}

		entries, err := eg.GetPileEntries(pile)
		if err != nil {
			log.Error().Err(err).Str("operation", "list").Str("pile", pile).Str("peer", peer).Msg("Failed obtaining pile entries")
			SendFailure(w, http.StatusInternalServerError, "error looking up list")
		}
		idents := []string{}
		now := time.Now()
		for entryID, entryMeta := range entries {
			if now.Before(entryMeta.Time().Add(pileConfig.Lifetime.Duration)) {
				idents = append(idents, entryID)
			} else {
				log.Debug().Str("peer", peer).Str("operation", "list").Str("pile", pile).Str("entry", entryID).Str("created", entryMeta.Time().Format(storage.TIME_FORMAT)).Msg("expired, but not culled yet")
			}
		}
		sort.SliceStable(idents, func(a int, b int) bool {
			return entries[idents[a]].Time().Before(entries[idents[b]].Time())
		})

		logEntry := log.Info().Str("operation", "list").Str("pile", pile).Str("peer", peer)
		sb := strings.Builder{}
		sb.WriteRune('{')
		sb.WriteString(`"format":1,`)
		sb.WriteString(fmt.Sprintf(`"lifetime":%q,`, pileConfig.Lifetime.String()))
		sb.WriteString(fmt.Sprintf(`"origin":%q,`, pileConfig.Origin))
		sb.WriteString(`"entries":[`)
		if len(entries) > 0 {
			sb.WriteRune('\n')
		}
		for i, entryID := range idents {
			entryMeta := entries[entryID]
			sb.WriteRune('\t')
			sb.WriteString(fmt.Sprintf(`{"filename":%q,"uploaded":%q,"entry":%q}`, entryMeta.Filename(), entryMeta.Time().UTC().Format(storage.TIME_FORMAT), entryID))
			if i < len(idents)-1 {
				sb.WriteRune(',')
			}
			sb.WriteRune('\n')
		}
		sb.WriteString(`]}`)
		SendMessage(w, http.StatusOK, sb.String())
		logEntry.Msg("Served!")
	}
}
