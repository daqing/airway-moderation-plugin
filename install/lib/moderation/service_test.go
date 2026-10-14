package moderation

import (
	"encoding/json"
	"testing"
)

func TestScreenReportsWordsAndCategories(t *testing.T) {
	service, err := New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result := service.Screen("有人在群里发广告，还提到网络赌博，大家小心")
	if !result.Flagged || len(result.Words) != 1 || result.Words[0] != "网络赌博" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Categories) != 1 || result.Categories[0] != "illegal" {
		t.Fatalf("categories = %#v", result.Categories)
	}
}

// An unflagged result still marshals to arrays, so a client can read the
// response without nil checks.
func TestScreenAllowsOrdinaryContent(t *testing.T) {
	service, _ := New("")
	result := service.Screen("新版本什么时候发？")
	if result.Flagged || len(result.Words) != 0 || len(result.Categories) != 0 {
		t.Fatalf("result = %#v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) != `{"flagged":false,"categories":[],"words":[]}` {
		t.Fatalf("encoded = %s", encoded)
	}
}

func TestReloadPicksUpAnEditedList(t *testing.T) {
	path := writeWordList(t, `{"categories":{"custom":["旧词"]}}`)
	service, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !service.Screen("旧词").Flagged {
		t.Fatal("the loaded word was not flagged")
	}

	if err := writeWordListAt(t, path, `{"categories":{"custom":["新词"]}}`); err != nil {
		t.Fatalf("rewrite word list: %v", err)
	}
	if err := service.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if service.Screen("旧词").Flagged {
		t.Fatal("the removed word is still flagged")
	}
	if !service.Screen("新词").Flagged {
		t.Fatal("the added word is not flagged")
	}
	if service.Info().Words != 1 {
		t.Fatalf("source = %#v", service.Info())
	}
}

// A word list an operator broke must not disarm screening: the reload fails
// and the list already in use keeps answering.
func TestReloadKeepsTheListInUseOnFailure(t *testing.T) {
	path := writeWordList(t, `{"categories":{"custom":["旧词"]}}`)
	service, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := writeWordListAt(t, path, `{"categories":`); err != nil {
		t.Fatalf("rewrite word list: %v", err)
	}
	if err := service.Reload(); err == nil {
		t.Fatal("a broken word list was accepted")
	}
	if !service.Screen("旧词").Flagged {
		t.Fatal("the list in use was replaced by a broken reload")
	}
	if service.Info().Words != 1 {
		t.Fatalf("source = %#v", service.Info())
	}
}

func TestNewRejectsABrokenList(t *testing.T) {
	if _, err := New(writeWordList(t, `{"categories":}`)); err == nil {
		t.Fatal("New accepted a broken word list")
	}
}

func TestSetupInstallsTheServiceHandlersRead(t *testing.T) {
	path := writeWordList(t, `{"categories":{"custom":["服务词表"]}}`)
	if err := Setup(path); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if !Current().Screen("服务词表").Flagged {
		t.Fatal("Current still screens against another list")
	}
	if Current().Path() != path {
		t.Fatalf("path = %q", Current().Path())
	}
}
