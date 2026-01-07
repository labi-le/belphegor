package domain

type EventRequest = Event[Request]

type Request struct {
	ID MessageID
}

func NewRequest(id MessageID) Event[Request] {
	return NewEvent(Request{ID: id})
}
