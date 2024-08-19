package storage

import (
	"encoding/json"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

type Config struct {
	Piles map[string]PileConfig `json:"piles"`
}

type PileConfig struct {
	Lifetime time.Duration `json:"lifetime"`
}

func (c Config) BucketNames() [][]byte {
	names := make([][]byte, 0)
	for key := range c.Piles {
		names = append(names, []byte(key))
	}
	return names
}

func LoadConfig(filename string) Config {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal().Msgf("%s was not found.", filename)
		}
		log.Fatal().Err(err).Msgf("Error opening %s", filename)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := Config{}
	decoder.Decode(&config)
	return config
}
