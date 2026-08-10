package secrets

import "github.com/zalando/go-keyring"

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
