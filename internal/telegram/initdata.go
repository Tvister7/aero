package telegram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

type User struct {
	ID       int64
	Username string
}

type Sender interface {
	Send(ctx context.Context, chatID int64, text string) error
}

func Verify(botToken, initData string, now time.Time) (User, *domain.Error) {
	if strings.TrimSpace(botToken) == "" || strings.TrimSpace(initData) == "" {
		return User{}, domain.Fail(domain.Unauthorized)
	}
	values, err := url.ParseQuery(initData)
	if err != nil || values.Get("hash") == "" {
		return User{}, domain.Fail(domain.Unauthorized)
	}
	auth, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || now.Sub(time.Unix(auth, 0)) > 24*time.Hour {
		return User{}, domain.Fail(domain.Unauthorized)
	}
	got, err := hex.DecodeString(values.Get("hash"))
	if err != nil {
		return User{}, domain.Fail(domain.Unauthorized)
	}
	want := mac(botToken, dataCheck(values))
	if !hmac.Equal(got, want) {
		return User{}, domain.Fail(domain.Unauthorized)
	}
	var user struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}
	if json.Unmarshal([]byte(values.Get("user")), &user) != nil || user.ID == 0 {
		return User{}, domain.Fail(domain.Unauthorized)
	}
	return User{ID: user.ID, Username: user.Username}, nil
}

func Sign(botToken string, values url.Values) string {
	copied := url.Values{}
	for key, list := range values {
		if key == "hash" {
			continue
		}
		copied[key] = append([]string(nil), list...)
	}
	copied.Set("hash", hex.EncodeToString(mac(botToken, dataCheck(copied))))
	return copied.Encode()
}

func dataCheck(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+values.Get(key))
	}
	return strings.Join(lines, "\n")
}

func mac(botToken, message string) []byte {
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	sum := hmac.New(sha256.New, secret.Sum(nil))
	sum.Write([]byte(message))
	return sum.Sum(nil)
}
