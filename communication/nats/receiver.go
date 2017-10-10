package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"
)

type receiverNats struct {
	connection   *nats.Conn
	messageTopic string
}

func (receiver *receiverNats) Receive(consumer communication.MessageConsumer) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(consumer.MessageType()),
		func(message *nats.Msg) {
			consumer.Consume(message.Data)
		},
	)
	return err
}

func (receiver *receiverNats) Respond(
	requestType communication.RequestType,
	callback communication.RequestHandler,
) error {

	_, err := receiver.connection.Subscribe(
		receiver.messageTopic+string(requestType),
		func(message *nats.Msg) {
			response := callback(message.Data)
			receiver.connection.Publish(message.Reply, []byte(response))
		},
	)
	return err
}
