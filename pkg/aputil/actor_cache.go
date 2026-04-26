package aputil

import vocab "github.com/go-ap/activitypub"

func ActorIconIRI(actor *vocab.Actor) string {
	if actor == nil || vocab.IsNil(actor.Icon) {
		return ""
	}

	if vocab.IsIRI(actor.Icon) {
		return actor.Icon.GetLink().String()
	}

	iconIRI := ""
	_ = vocab.OnLink(actor.Icon, func(link *vocab.Link) error {
		iconIRI = link.Href.String()
		return nil
	})
	if iconIRI != "" {
		return iconIRI
	}

	_ = vocab.OnObject(actor.Icon, func(obj *vocab.Object) error {
		if !vocab.IsNil(obj.URL) {
			iconIRI = itemIRIString(obj.URL)
			if iconIRI != "" {
				return nil
			}
		}

		if obj.ID != "" {
			iconIRI = obj.ID.String()
		}

		return nil
	})

	return iconIRI
}
