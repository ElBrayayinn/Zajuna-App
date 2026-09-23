package secrets

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const ServiceName = "zajuna-app"

type Store interface {
	Set(user, password string) error
	Get(user string) (string, error)
}

type SystemStore struct{}

func (SystemStore) Set(user, password string) error {
	return keyring.Set(ServiceName, user, password)
}

func (SystemStore) Get(user string) (string, error) {
	return keyring.Get(ServiceName, user)
}

// Delete removes the stored password. A missing entry is not an error, so a
// data reset can call it unconditionally.
func (SystemStore) Delete(user string) error {
	if err := keyring.Delete(ServiceName, user); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}
