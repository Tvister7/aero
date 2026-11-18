package telegram

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/Tvister7/aero/internal/domain"
)

func TestInitData(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(now.Unix(), 10))
	values.Set("query_id", "aa")
	values.Set("user", `{"id":1001,"username":"anna"}`)
	raw := Sign("token", values)
	user, err := Verify("token", raw, now)
	if err != nil || user.ID != 1001 || user.Username != "anna" {
		t.Fatalf("user %+v err %v", user, err)
	}
	_, err = Verify("token", raw, now.Add(25*time.Hour))
	if err == nil || err.Code != domain.Unauthorized {
		t.Fatalf("old auth: %v", err)
	}
	_, err = Verify("other", raw, now)
	if err == nil || err.Code != domain.Unauthorized {
		t.Fatalf("bad token: %v", err)
	}
	_, err = Verify("token", "", now)
	if err == nil || err.Code != domain.Unauthorized {
		t.Fatalf("empty: %v", err)
	}
}
