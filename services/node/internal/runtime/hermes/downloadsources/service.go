package downloadsources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"
)

const settingKey = "hermes.download-sources.v1"

var ErrCorrupt = errors.New("stored Hermes download source settings are invalid")

type Store interface {
	GetAppSetting(context.Context, string) ([]byte, bool, error)
	PutAppSetting(context.Context, string, []byte, time.Time) error
	DeleteAppSetting(context.Context, string) error
}

type Provider interface {
	Get(context.Context) (Config, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Get(ctx context.Context) (Config, error) {
	if s == nil || s.store == nil {
		return Default(), nil
	}
	payload, found, err := s.store.GetAppSetting(ctx, settingKey)
	if err != nil {
		return Config{}, err
	}
	if !found {
		return Default(), nil
	}
	var config Config
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, ErrCorrupt
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Config{}, ErrCorrupt
	}
	normalized, err := Normalize(config)
	if err != nil {
		return Config{}, ErrCorrupt
	}
	return normalized, nil
}

func (s *Service) Save(ctx context.Context, config Config) (Config, error) {
	normalized, err := Normalize(config)
	if err != nil {
		return Config{}, err
	}
	if s == nil || s.store == nil {
		return Config{}, errors.New("download source settings store is unavailable")
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return Config{}, err
	}
	if err := s.store.PutAppSetting(ctx, settingKey, payload, s.now()); err != nil {
		return Config{}, err
	}
	return normalized, nil
}

func (s *Service) Reset(ctx context.Context) (Config, error) {
	if s == nil || s.store == nil {
		return Config{}, errors.New("download source settings store is unavailable")
	}
	if err := s.store.DeleteAppSetting(ctx, settingKey); err != nil {
		return Config{}, err
	}
	return Default(), nil
}
