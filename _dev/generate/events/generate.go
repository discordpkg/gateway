package events

type EventInfo struct {
	Name        string
	Description string
	Event       string
}

func (e EventInfo) String() string {
	return e.Name
}
