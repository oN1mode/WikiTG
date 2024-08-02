package utils

import (
	"net/url"
	"strings"
)

func TrimString(str string) string {
	return strings.Trim(str, "https://ru.wikipedia.org/wiki/")
}

func RecoveryString(str string) string {
	return ("https://ru.wikipedia.org/wiki/" + str)
}

func UrlEncoded(str string) (string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
