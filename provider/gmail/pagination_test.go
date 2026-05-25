package gmail

import (
	"testing"

	"google.golang.org/api/gmail/v1"
)

func TestCollectMessageIDs(t *testing.T) {
	pager := func(nextToken string) (*gmail.ListMessagesResponse, error) {
		ids := []*gmail.Message{
			{Id: "1"}, {Id: "2"}, {Id: "3"},
		}
		return &gmail.ListMessagesResponse{
			Messages:           ids,
			ResultSizeEstimate: 3,
		}, nil
	}
	ids, total, err := collectMessageIDs(pager, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 ids, got %d", len(ids))
	}
}

func TestSliceWindow(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	result := sliceWindow(items, 1, 2)
	if len(result) != 2 || result[0] != "b" || result[1] != "c" {
		t.Errorf("unexpected window: %v", result)
	}
}
