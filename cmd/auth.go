package main

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"
)

const AUTH_SCHEMA string = "Basic "

// [auth] Save a token to the token file
func readTokens() (TokenList, error) {
	var tokens TokenList
	
	err := readJSONFile(&tokens, TOKENS_FILE_PATH)
	if err != nil {
		return tokens, err
	}

	return tokens, nil
}

// [auth] Save a token to the token file
func saveToken(t Token) error {
	tokens, err := readTokens()
	if err != nil {
		return err
	}

	// Append a token to the list
	tokens.Tokens = append(tokens.Tokens, t)

	err = writeJSONFile(tokens, TOKENS_FILE_PATH)
	if err != nil {
		return err
	}

	return nil
}

// [auth] Validate a given token if it is matched with any token in the list and not expired
func verifyToken(r *http.Request) (ok bool, err error) {
	// Extract the authorization header
	id, secret, err := extractAuthHeader(r)
	if err != nil {
		return false, err
	}

	// Get current tokens in the server
	tokens, err := readTokens()
	if err != nil {
		return false, err
	}

	// Create a new token list for saving updated tokens back
	newTokens := []Token{}

	// Validate the token
	ok = false
	now := time.Now()
	for _, token := range tokens.Tokens {
		// Compare client's secret and saved secret
		// If the secret is the same, then they pass the authentication
		// and the sharing is accepted
		if(id == token.DeviceId && secret == token.Secret && now.Before(token.ExpiredAt)) {
			ok = true
		}
		// Update the token list
		if(now.Before(token.ExpiredAt) && token.Secret != secret) {
			newTokens = append(newTokens, token)
		}

	}
	if !ok {
		return false, nil
	}

	// Save updated token list to the file
	err = writeJSONFile(TokenList{Tokens: newTokens}, TOKENS_FILE_PATH)
	if err != nil {
		return false, err
	}

	return true, nil
}


/* --- MISCELLANEOUS --- */

// [auth] Extract id and password from a given authorization header
func extractAuthHeader(r *http.Request) (id, password string, err error) {
	// Extract the authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, AUTH_SCHEMA) {
		return "", "", errors.New("authorization requires Basic scheme")
	}
	
	// Decode the base-64 token to string
	bAuth, err := base64.StdEncoding.DecodeString(authHeader[len(AUTH_SCHEMA):])
	if err != nil {
		return "", "", errors.New("base64 encoding issue")
	}

	// Extract the token
	auth := string(bAuth)
	authArr := strings.Split(auth, ":")
	if len(authArr) != 2 {
		return "", "", errors.New("authorization header is not completed")
	}

	return authArr[0], authArr[1], nil
}