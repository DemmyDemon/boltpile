package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	ErrNoSuchPile = errors.New("no such pile")
)

type Lifetime struct {
	time.Duration
}

func (lt Lifetime) MarshalJSON() ([]byte, error) {
	return json.Marshal(lt.String())
}

func (lt *Lifetime) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		lt.Duration = time.Duration(value)
	case string:
		dur, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("unmarshal lifetime: %w", err)
		}
		lt.Duration = dur
	default:
		return errors.New("invalid lifetime")
	}
	return nil
}

type Config struct {
	Piles         map[string]PileConfig `json:"piles"`
	ForwardHeader string                `json:"forward_header"`
}

type PileConfig struct {
	Lifetime Lifetime `json:"lifetime"`
	Origin   string   `json:"origin"`
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
	log.Debug().Str("filename", filename).Msg("Loading config")
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
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration!")
	}
	return config
}
