package activitypub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

// RemoteReply holds the data extracted from a remote reply note.
type RemoteReply struct {
	ObjectIRI string
	ActorIRI  string
	ActorName string
	Content   string
}

// FetchRemoteReplies fetches a replies collection from the given IRI and
// returns the parsed replies.  It follows the "first" page link if needed.
func FetchRemoteReplies(ctx context.Context, repliesIRI string) ([]RemoteReply, error) {
	body, err := fetchAPDocument(ctx, repliesIRI)
	if err != nil {
		return nil, err
	}

	// Try to extract items directly (some implementations inline them)
	replies := extractReplies(body)
	if len(replies) > 0 {
		return replies, nil
	}

	// Otherwise follow the "first" link for the first page
	firstIRI := extractRepliesFirstPageIRI(body)
	if firstIRI == "" {
		return nil, nil
	}

	pageBody, err := fetchAPDocument(ctx, firstIRI)
	if err != nil {
		return nil, err
	}

	return extractReplies(pageBody), nil
}

func fetchAPDocument(ctx context.Context, iri string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", ContentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch rejected: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func extractReplies(body []byte) []RemoteReply {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}

	itemsRaw, ok := raw["orderedItems"]
	if !ok {
		itemsRaw, ok = raw["items"]
		if !ok {
			return nil
		}
	}

	var items []json.RawMessage
	if err := json.Unmarshal(itemsRaw, &items); err != nil {
		return nil
	}

	var replies []RemoteReply
	for _, itemJSON := range items {
		reply := parseReplyItem(itemJSON)
		if reply != nil {
			replies = append(replies, *reply)
		}
	}

	return replies
}

func parseReplyItem(raw json.RawMessage) *RemoteReply {
	// Items can be IRIs (strings) pointing to Note objects, or inline objects.
	// Try inline objects first.
	var note vocab.Object
	if err := jsonld.Unmarshal(raw, &note); err != nil {
		return nil
	}

	if note.ID == "" {
		return nil
	}

	content := ""
	if note.Content != nil {
		content = note.Content.String()
	}

	if content == "" {
		return nil
	}

	actorIRI := itemIRIString(note.AttributedTo)
	actorName := ""

	if actorIRI != "" {
		name, _, ok := GetCachedActorProfile(actorIRI)
		if ok {
			actorName = name
		}
	}

	return &RemoteReply{
		ObjectIRI: note.ID.String(),
		ActorIRI:  actorIRI,
		ActorName: actorName,
		Content:   content,
	}
}

func extractRepliesFirstPageIRI(body []byte) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}

	firstRaw, ok := raw["first"]
	if !ok {
		return ""
	}

	var iri string
	if err := json.Unmarshal(firstRaw, &iri); err == nil && iri != "" {
		return iri
	}

	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(firstRaw, &obj); err == nil {
		return obj.ID
	}

	return ""
}
