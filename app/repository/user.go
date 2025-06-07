// Package repository provides data access functionality.
package repository

import (
	"errors"

	"github.com/geoffjay/plantd/app/models"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// TODO: move these into a database.
var hashedPasswords map[string]string
var users map[string]*models.User

// Initialize sets up the repository.
//
// TODO: remove this once there's a database.
func Initialize() {
	log.Debug("initializing user repository")

	hashedPasswords = make(map[string]string)
	for username, password := range map[string]string{
		"admin@plantd.com": "password",
		"user@plantd.com":  "password",
	} {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			panic(err)
		}
		hashedPasswords[username] = string(hashedPassword)
	}

	users = make(map[string]*models.User)
	for email, hashedPassword := range hashedPasswords {
		users[email] = &models.User{Email: email, Password: hashedPassword}
	}
}

// EmptyHashString is used to help prevent timing attacks.
func emptyHashString() string {
	emptyHash, err := bcrypt.GenerateFromPassword([]byte(""), 10)
	if err != nil {
		panic(err)
	}
	return string(emptyHash)
}

// FindUserByCredentials finds a user by their email and password.
func FindUserByCredentials(email, password string) (*models.User, error) {
	var checkPassword string

	log.WithFields(log.Fields{"email": email}).Info("looking up user by credentials")

	user, exists := users[email]
	if exists {
		log.Info("found user")
		checkPassword = user.Password
	} else {
		log.Info("user not found")
		checkPassword = emptyHashString()
	}

	if bcrypt.CompareHashAndPassword([]byte(checkPassword), []byte(password)) != nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}
