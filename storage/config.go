package storage

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	ErrNoSuchPile = errors.New("no such pile")
)

type Config struct {
	Piles map[string]PileConfig `json:"piles"`
}

type PileConfig struct {
	Lifetime time.Duration `json:"lifetime"`
	Origin   string        `json:"origin"`
}

func (c Config) BucketNames() [][]byte {
	names := make([][]byte, 0)
	for key := range c.Piles {
		names = append(names, []byte(key))
	}
	return names
}

func (c Config) Pile(pile string) (PileConfig, error) {
	if pileConfig, ok := c.Piles[pile]; ok {
		return pileConfig, nil
	}
	return PileConfig{}, ErrNoSuchPile
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
