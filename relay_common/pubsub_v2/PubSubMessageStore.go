package pubsub_v2

// relations:
// tbl messages
// index uint32 | topic_id string | payload blob | priority uint8

// tbl subscriber_groups
//

type IPubSubMessageStore interface {
	Put(IPubSubMessage) error
	BulkPut([]IPubSubMessage) error
	Get(uint32) (IPubSubMessage, error)
	GetInRange(uint32, uint32) ([]IPubSubMessage, error)
	GetBySubscriberGroupFrom(string, uint32) ([]IPubSubMessage, error)
}

type MySqlPubSubMessageStore struct {
}
