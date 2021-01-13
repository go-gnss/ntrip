package main

import (
	"sync"

	"github.com/go-gnss/ntrip/internal/inmemory"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	sync.RWMutex
	Users map[string]string
}

func (a *Auth) Authorise(action inmemory.Action, mount, username, password string) (bool, error) {
	a.RLock()
	storedPassword, userFound := a.Users[username]
	a.RUnlock()
	if !userFound {
		return false, nil
	}

	err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}
