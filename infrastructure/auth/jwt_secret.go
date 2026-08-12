package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	"stersh.ru/mediator/domain"
)

func LoadOrCreateJWTSecret(settings domain.SettingRepository) (string, error) {
	secret, err := settings.Get(domain.SettingJWTSecret)
	if err == nil && secret != "" {
		return secret, nil
	}
	if err != nil && !errors.Is(err, domain.ErrSettingNotFound) {
		return "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	secret = base64.RawURLEncoding.EncodeToString(buf)
	if err := settings.Set(domain.SettingJWTSecret, secret); err != nil {
		return "", err
	}
	return secret, nil
}
