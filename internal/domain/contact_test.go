package domain

import "testing"

func TestMergeProfileClearsAndRejects(t *testing.T) {
	client := Client{Name: "Анна", Phone: "+995555112233"}
	cleared, err := MergeProfile(client, "  ", "")
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Name != "" || cleared.Phone != "" {
		t.Fatalf("not cleared %+v", cleared)
	}
	if _, err := MergeProfile(client, "12345", "+995555112233"); err == nil || err.Message == "" {
		t.Fatalf("digits name %v", err)
	}
	if _, err := MergeProfile(client, "Анна", "телефон"); err == nil || err.Message == "" {
		t.Fatalf("letter phone %v", err)
	}
	if _, err := MergeProfile(client, "Анна-Мария", "+995 555 11 22 33"); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizePricing(t *testing.T) {
	got, err := NormalizePricing(`{"personal":{"split":130,"single":80},"base":{"1":35}}`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"base":{"1":35},"personal":{"single":80,"split":130}}` {
		t.Fatalf("pricing %s", got)
	}
	if _, err := NormalizePricing(`{"vip":{"1":10}}`); err == nil {
		t.Fatal("unknown section accepted")
	}
	if _, err := NormalizePricing(`{"base":{"1":0}}`); err == nil {
		t.Fatal("zero price accepted")
	}
	if _, err := NormalizePricing(`[]`); err == nil {
		t.Fatal("array accepted")
	}
}
