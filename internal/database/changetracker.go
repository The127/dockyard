package database

type Type int

const (
	Added Type = iota
	Updated
	Deleted
)

type ChangeEntry struct {
	changeType Type
	itemType   int
	item       any
}

func NewChangeEntry(changeType Type, itemType int, item any) *ChangeEntry {
	return &ChangeEntry{
		changeType: changeType,
		itemType:   itemType,
		item:       item,
	}
}

func (e *ChangeEntry) GetChangeType() Type {
	return e.changeType
}

func (e *ChangeEntry) GetItemType() int {
	return e.itemType
}

func (e *ChangeEntry) GetItem() any {
	return e.item
}

type ChangeTracker struct {
	entries []*ChangeEntry
}

func NewTracker() *ChangeTracker {
	return &ChangeTracker{
		entries: []*ChangeEntry{},
	}
}

func (t *ChangeTracker) Add(entry *ChangeEntry) {
	t.entries = append(t.entries, entry)
}

func (t *ChangeTracker) GetChanges() []*ChangeEntry {
	return t.entries
}
