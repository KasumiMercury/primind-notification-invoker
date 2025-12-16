package domain

import "fmt"

type FCMToken string

func NewFCMToken(token string) (FCMToken, error) {
	if token == "" {
		return "", fmt.Errorf("%w: empty token", ErrInvalidToken)
	}
	return FCMToken(token), nil
}

func NewFCMTokens(tokens []string) ([]FCMToken, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("%w: empty tokens list", ErrInvalidToken)
	}

	result := make([]FCMToken, len(tokens))
	for i, t := range tokens {
		token, err := NewFCMToken(t)
		if err != nil {
			return nil, fmt.Errorf("token at index %d: %w", i, err)
		}
		result[i] = token
	}
	return result, nil
}

func (t FCMToken) String() string {
	return string(t)
}

func ToStrings(tokens []FCMToken) []string {
	result := make([]string, len(tokens))
	for i, t := range tokens {
		result[i] = t.String()
	}
	return result
}
