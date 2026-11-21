package store_test

import (
	"context"
	"testing"

	"github.com/Tvister7/aero/internal/domain"
)

func TestMessageStartsFromCatalogAndKeepsEdits(t *testing.T) {
	db := openTest(t)
	list, err := db.Messages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Key != domain.MessageStart || list[0].Body != domain.StartText(true) {
		t.Fatalf("catalog %+v", list)
	}
	saved, err := db.PutMessage(context.Background(), domain.MessageStart, "  Ждём вас в студии.  ")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Body != "Ждём вас в студии." || saved.Title != "Приветствие /start" {
		t.Fatalf("saved %+v", saved)
	}
	body, err := db.MessageBody(context.Background(), domain.MessageStart)
	if err != nil || body != "Ждём вас в студии." {
		t.Fatalf("body %q %v", body, err)
	}
	if _, err := db.PutMessage(context.Background(), "reminder", "позже"); err == nil {
		t.Fatal("unknown key")
	}
	if _, err := db.PutMessage(context.Background(), domain.MessageStart, "   "); err == nil {
		t.Fatal("empty body")
	}
}
