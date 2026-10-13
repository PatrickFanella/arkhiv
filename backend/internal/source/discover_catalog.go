package source

import (
	"context"
	"encoding/json"
	"fmt"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"net/url"
)

type CatalogDiscovery struct {
	Items []struct {
		Identifier string `json:"identifier"`
	} `json:"items"`
	Cursor string `json:"cursor"`
	Total  int64  `json:"total"`
}

func (c *Client) DiscoverCatalog(ctx context.Context, collection, cursor string) (CatalogDiscovery, Response, error) {
	var page CatalogDiscovery
	if !domain.ValidID(collection) || len(cursor) > 4096 {
		return page, Response{}, fmt.Errorf("invalid catalog discovery bounds")
	}
	query := "collection:" + collection + " AND NOT mediatype:collection"
	// The source reused distinct q/cursor requests with identical fields during
	// October 13, 2026 checks. An absent metadata field partitions responses.
	fields := "identifier,arkhiv_query_" + domain.Digest([]byte(query+"\x00"+cursor))
	params := url.Values{"q": {query}, "fields": {fields}, "count": {"100"}}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	response, err := c.get(ctx, "/services/search/v1/scrape?"+params.Encode(), "", "")
	if err != nil {
		return page, response, err
	}
	if json.Unmarshal(response.Raw, &page) != nil || page.Items == nil || len(page.Items) > 100 || len(page.Cursor) > 4096 || page.Total < 0 {
		return page, response, fmt.Errorf("invalid catalog discovery response")
	}
	for _, item := range page.Items {
		if !domain.ValidID(item.Identifier) {
			return page, response, fmt.Errorf("invalid discovered identity")
		}
	}
	if page.Cursor != "" && (page.Cursor == cursor || len(page.Items) == 0) {
		return page, response, fmt.Errorf("catalog cursor made no progress")
	}
	return page, response, nil
}
