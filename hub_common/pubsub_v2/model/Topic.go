package model

type ITopic interface {
	Id() string
	Publishers() []string
	Subscribers() []string
}

type Topic struct {
	id          string
	publishers  []string
	subscribers []string
}

func (t *Topic) Id() string {
	return t.id
}

func (t *Topic) Publishers() []string {
	return t.publishers
}

func (t *Topic) Subscribers() []string {
	return t.subscribers
}
