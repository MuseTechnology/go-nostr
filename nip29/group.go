package nip29

import (
	"fmt"

	"github.com/MuseTechnology/go-nostr"
)

type Group struct {
	ID      string
	Name    string
	Picture string
	About   string
	Members map[string]*Role
	Private bool
	Closed  bool

	LastMetadataUpdate nostr.Timestamp
	LastAdminsUpdate   nostr.Timestamp
	LastMembersUpdate  nostr.Timestamp
}

func NewGroup(id string) Group {
	return Group{
		ID:      id,
		Name:    id,
		Members: make(map[string]*Role),
	}
}

func (group Group) ToMetadataEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      nostr.KindSimpleGroupMetadata,
		CreatedAt: group.LastMetadataUpdate,
		Content:   group.About,
		Tags: nostr.Tags{
			nostr.Tag{"d", group.ID},
		},
	}
	if group.Name != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"name", group.Name})
	}
	if group.About != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"about", group.Name})
	}
	if group.Picture != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"picture", group.Picture})
	}

	// status
	if group.Private {
		evt.Tags = append(evt.Tags, nostr.Tag{"private"})
	} else {
		evt.Tags = append(evt.Tags, nostr.Tag{"public"})
	}
	if group.Closed {
		evt.Tags = append(evt.Tags, nostr.Tag{"closed"})
	} else {
		evt.Tags = append(evt.Tags, nostr.Tag{"open"})
	}

	return evt
}

func (group Group) ToAdminsEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      nostr.KindSimpleGroupAdmins,
		CreatedAt: group.LastAdminsUpdate,
		Tags:      make(nostr.Tags, 1, 1+len(group.Members)/3),
	}
	evt.Tags[0] = nostr.Tag{"d", group.ID}

	for member, role := range group.Members {
		if role != nil {
			// is an admin
			tag := make([]string, 3, 3+len(role.Permissions))
			tag[0] = "p"
			tag[1] = member
			tag[2] = role.Name
			for perm := range role.Permissions {
				tag = append(tag, string(perm))
			}
			evt.Tags = append(evt.Tags, tag)
		}
	}

	return evt
}

func (group Group) ToMembersEvent() *nostr.Event {
	evt := &nostr.Event{
		Kind:      nostr.KindSimpleGroupMembers,
		CreatedAt: group.LastMembersUpdate,
		Tags:      make(nostr.Tags, 1, 1+len(group.Members)),
	}
	evt.Tags[0] = nostr.Tag{"d", group.ID}

	for member := range group.Members {
		// include both admins and normal members
		evt.Tags = append(evt.Tags, nostr.Tag{"p", member})
	}

	return evt
}

func (group *Group) MergeInMetadataEvent(evt *nostr.Event) error {
	if evt.Kind != nostr.KindSimpleGroupMetadata {
		return fmt.Errorf("expected kind %d, got %d", nostr.KindSimpleGroupMetadata, evt.Kind)
	}
	if evt.CreatedAt < group.LastMetadataUpdate {
		return fmt.Errorf("event is older than our last update (%d vs %d)", evt.CreatedAt, group.LastMetadataUpdate)
	}

	group.LastMetadataUpdate = evt.CreatedAt
	group.Name = group.ID

	if tag := evt.Tags.GetFirst([]string{"name", ""}); tag != nil {
		group.Name = (*tag)[1]
	}
	if tag := evt.Tags.GetFirst([]string{"about", ""}); tag != nil {
		group.About = (*tag)[1]
	}
	if tag := evt.Tags.GetFirst([]string{"picture", ""}); tag != nil {
		group.Picture = (*tag)[1]
	}

	if tag := evt.Tags.GetFirst([]string{"private"}); tag != nil {
		group.Private = true
	}
	if tag := evt.Tags.GetFirst([]string{"closed"}); tag != nil {
		group.Closed = true
	}

	return nil
}

func (group *Group) MergeInAdminsEvent(evt *nostr.Event) error {
	if evt.Kind != nostr.KindSimpleGroupAdmins {
		return fmt.Errorf("expected kind %d, got %d", nostr.KindSimpleGroupAdmins, evt.Kind)
	}
	if evt.CreatedAt < group.LastAdminsUpdate {
		return fmt.Errorf("event is older than our last update (%d vs %d)", evt.CreatedAt, group.LastAdminsUpdate)
	}

	group.LastAdminsUpdate = evt.CreatedAt
	for _, tag := range evt.Tags {
		if len(tag) < 3 {
			continue
		}
		if tag[0] != "p" {
			continue
		}
		if !nostr.IsValid32ByteHex(tag[1]) {
			continue
		}

		role := group.Members[tag[1]]
		if role == nil {
			role = &Role{Name: tag[2]}
			group.Members[tag[1]] = role
		}
		if role.Permissions == nil {
			role.Permissions = make(map[Permission]struct{}, len(tag)-3)
		}
		for _, perm := range tag[2:] {
			role.Permissions[Permission(perm)] = struct{}{}
		}
	}

	return nil
}

func (group *Group) MergeInMembersEvent(evt *nostr.Event) error {
	if evt.Kind != nostr.KindSimpleGroupMembers {
		return fmt.Errorf("expected kind %d, got %d", nostr.KindSimpleGroupMembers, evt.Kind)
	}
	if evt.CreatedAt < group.LastMembersUpdate {
		return fmt.Errorf("event is older than our last update (%d vs %d)", evt.CreatedAt, group.LastMembersUpdate)
	}

	group.LastMembersUpdate = evt.CreatedAt
	for _, tag := range evt.Tags {
		if len(tag) < 2 {
			continue
		}
		if tag[0] != "p" {
			continue
		}
		if !nostr.IsValid32ByteHex(tag[1]) {
			continue
		}

		_, exists := group.Members[tag[1]]
		if !exists {
			group.Members[tag[1]] = EmptyRole
		}
	}

	return nil
}
