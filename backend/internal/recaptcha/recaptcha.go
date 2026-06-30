package recaptcha

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type siteVerifyResponse struct {
	Success bool     `json:"success"`
	Errors  []string `json:"error-codes"`
}

func VerifyToken(secret, token string) (bool, error) {
	if secret == "" {
		return true, nil
	}

	data := url.Values{
		"secret":   {secret},
		"response": {token},
	}

	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", data)
	if err != nil {
		return false, fmt.Errorf("recaptcha verify request failed: %w", err)
	}
	defer resp.Body.Close()

	var result siteVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("recaptcha verify decode failed: %w", err)
	}

	if !result.Success {
		return false, fmt.Errorf("recaptcha verification failed: %v", result.Errors)
	}

	return true, nil
}
