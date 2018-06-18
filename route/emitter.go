package route

import (
	"encoding/json"
	"fmt"

	nats "github.com/nats-io/go-nats"
)

const publisherSubject = "router.register"

type Publisher interface {
	Publish(subj string, data []byte) error
}

type NATSPublisher struct {
	NatsClient *nats.Conn
}

func (p *NATSPublisher) Publish(subj string, data []byte) error {
	return p.NatsClient.Publish(subj, data)
}

type RouteEmitter struct {
	Publisher Publisher
	Scheduler TaskScheduler
	Work      <-chan []RegistryMessage
}

func NewRouteEmitter(nats *nats.Conn, workChannel chan []RegistryMessage) *RouteEmitter {
	return &RouteEmitter{
		Publisher: &NATSPublisher{NatsClient: nats},
		Scheduler: &SimpleLoopScheduler{},
		Work:      workChannel,
	}
}

func NewRouteEmitter(nats *nats.Conn, workChannel chan []RegistryMessage, scheduler TaskScheduler) *RouteEmitter {
	return &RouteEmitter{
		Publisher: &NATSPublisher{NatsClient: nats},
		Scheduler: scheduler,
		Work:      workChannel,
	}
}

func (r *RouteEmitter) Start() {
	r.Scheduler.Schedule(func() error {
		select {
		case batch := <-r.Work:
			r.emit(batch)
		}
		return nil
	})
}

func (r *RouteEmitter) emit(batch []RegistryMessage) {
	for _, route := range batch {
		routeJson, err := json.Marshal(route)
		if err != nil {
			fmt.Println("Faild to marshal route message:", err.Error())
			continue
		}

		if err = r.Publisher.Publish(publisherSubject, routeJson); err != nil {
			fmt.Println("failed to publish route:", err.Error())
		}
	}
}
