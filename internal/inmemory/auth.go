package inmemory

type Action int

const (
	PublishAction Action = iota
	SubscribeAction
)

type Authoriser interface {
	Authorise(action Action, mount, username, password string) (authorised bool, err error)
}
