package user

// Actor identifies a user performing a domain action.
type Actor struct {
	ID string
}

func NewActor(id string) Actor {
	return Actor{ID: id}
}
