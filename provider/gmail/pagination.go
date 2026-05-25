package gmail

import "google.golang.org/api/gmail/v1"

type messageIDPager func(nextToken string) (*gmail.ListMessagesResponse, error)

func collectMessageIDs(pager messageIDPager, targetCount int) ([]string, int, error) {
	var allIDs []string
	var total int
	pageToken := ""
	firstPage := true

	for len(allIDs) < targetCount {
		resp, err := pager(pageToken)
		if err != nil {
			return nil, 0, err
		}
		if firstPage {
			total = int(resp.ResultSizeEstimate)
			firstPage = false
		}
		for _, m := range resp.Messages {
			allIDs = append(allIDs, m.Id)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return allIDs, total, nil
}

func sliceWindow[T any](items []T, offset, limit int) []T {
	start := offset
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
