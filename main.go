package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DemmyDemon/boltpile/handler"
	"github.com/DemmyDemon/boltpile/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.DurationFieldUnit = time.Second
	zerolog.DurationFieldInteger = true

	loglevel := strings.ToLower(os.Getenv("BOLTPILE_LOGLEVEL"))
	switch loglevel {
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "info", "":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
		log.Warn().Msgf("BOLTPILE_LOGLEVEL of %q makes no sense to me. Using %q instead.", loglevel, "warn")
	}
}

func setupDirectory() string {
	directory := os.Getenv("POLTPILE_DIR")
	if directory == "" {
		var err error
		directory, err = os.Getwd()
		if err != nil {
			log.Fatal().Msgf("Could not determine what directory boltpile is running from: %s", err)
			os.Exit(1)
		}
	}
	if err := os.Chdir(directory); err != nil {
		log.Fatal().Msgf("Could not set Boltpile working directory to %s: %s", directory, err)
		os.Exit(1)
	}
	return directory
}

func setupPort() string {
	port := os.Getenv("BOLTPILE_PORT")
	if port == "" {
		port = "1995"
	}
	if _, err := strconv.Atoi(port); err != nil {
		log.Warn().Msgf("BOLTPILE_PORT %q does not appear to be a valid integer, falling back to port 1995", port)
		port = "1995"
	}
	return port
}

func main() {
	setupLogging()
	dir := setupDirectory()
	bind := os.Getenv("BOLTPILE_BIND")
	port := setupPort()

	log.Info().Msgf("boltpile starting in %s, listening on %s:%s", dir, bind, port)

	entryHandler := storage.MustOpenBoltDatabase("boltpile.db")

	config := storage.LoadConfig("boltpile.json")

	if err := entryHandler.Startup(config); err != nil {
		log.Fatal().Err(err).Msg("Error during startup maintenance")
	}
	log.Debug().Msg("Startup maintenance complete")

	rateLimiter := handler.NewRateLimiter()

	http.Handle("GET /{pile}/{entry}", handler.GetFile(entryHandler, config))
	http.Handle("POST /{pile}/", handler.PostFile(entryHandler, config, rateLimiter))
	http.Handle("GET /{pile}/", handler.GetList(entryHandler, config, rateLimiter))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/" {
			log.Info().Str("peer", handler.DeterminePeer(config, r)).Msg("Requested /, forwarded to boltpile GitHub repo")
			http.Redirect(w, r, "https://github.com/DemmyDemon/boltpile", http.StatusSeeOther)
		} else {
			log.Info().Str("peer", handler.DeterminePeer(config, r)).Str("method", r.Method).Str("url", r.URL.String()).Msg("Not a recognized request")
			handler.SendMessage(w, http.StatusBadRequest, handler.REQUEST_WEIRD)
		}
	})
	if err := http.ListenAndServe(bind+":"+port, nil); err != nil {
		log.Fatal().Err(err).Msg("Error while serving boltpile!")
	}
}
