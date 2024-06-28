package shim

import (
	"context"
	"demo/pkg/namespaces"
	"demo/pkg/protobuf"
	"demo/pkg/ttrpcutil"
	"os"
	"sync"
	"time"

	v1 "demo/pkg/api/services/ttrpc/events/v1"
	"demo/pkg/events"
	"demo/pkg/ttrpc"
	"github.com/sirupsen/logrus"
)

const (
	queueSize  = 2048
	maxRequeue = 5
)

type item struct {
	ev    *v1.Envelope
	ctx   context.Context
	count int
}

// NewPublisher creates a new remote events publisher
func NewPublisher(address string) (*RemoteEventsPublisher, error) {
	client, err := ttrpcutil.NewClient(address)
	if err != nil {
		return nil, err
	}

	l := &RemoteEventsPublisher{
		client:  client,
		closed:  make(chan struct{}),
		requeue: make(chan *item, queueSize),
	}

	go l.processQueue()
	return l, nil
}

// RemoteEventsPublisher forwards events to a ttrpc server
type RemoteEventsPublisher struct {
	client  *ttrpcutil.Client
	closed  chan struct{}
	closer  sync.Once
	requeue chan *item
}

// Done returns a channel which closes when done
func (l *RemoteEventsPublisher) Done() <-chan struct{} {
	return l.closed
}

// Close closes the remote connection and closes the done channel
func (l *RemoteEventsPublisher) Close() (err error) {
	err = l.client.Close()
	l.closer.Do(func() {
		close(l.closed)
	})
	return err
}

func (l *RemoteEventsPublisher) processQueue() {
	for i := range l.requeue {
		if i.count > maxRequeue {
			logrus.Errorf("evicting %s from queue because of retry count", i.ev.Topic)
			// drop the event
			continue
		}
		executable, _ := os.Executable()
		if err := l.forwardRequest(i.ctx, &v1.ForwardRequest{Envelope: i.ev}); err != nil {
			logrus.WithError(err).Error("forward event " + executable)
			l.queue(i)
		}
	}
}

func (l *RemoteEventsPublisher) queue(i *item) {
	go func() {
		i.count++
		// re-queue after a short delay
		time.Sleep(time.Duration(1*i.count) * time.Second)
		l.requeue <- i
	}()
}

// Publish publishes the event by forwarding it to the configured ttrpc server
func (l *RemoteEventsPublisher) Publish(ctx context.Context, topic string, event events.Event) error {
	ns, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return err
	}
	any, err := protobuf.MarshalAnyToProto(event)
	if err != nil {
		return err
	}
	i := &item{
		ev: &v1.Envelope{
			Timestamp: protobuf.ToTimestamp(time.Now()),
			Namespace: ns,
			Topic:     topic,
			Event:     any,
		},
		ctx: ctx,
	}

	if err := l.forwardRequest(i.ctx, &v1.ForwardRequest{Envelope: i.ev}); err != nil {
		l.queue(i)
		return err
	}

	return nil
}

func (l *RemoteEventsPublisher) forwardRequest(ctx context.Context, req *v1.ForwardRequest) error {
	service, err := l.client.EventsService()
	if err == nil {
		fCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err = service.Forward(fCtx, req)
		cancel()
		if err == nil {
			return nil
		}
	}

	if err != ttrpc.ErrClosed {
		return err
	}

	// Reconnect and retry request
	if err = l.client.Reconnect(); err != nil {
		return err
	}

	service, err = l.client.EventsService()
	if err != nil {
		return err
	}

	// try again with a fresh context, otherwise we may get a context timeout unexpectedly.
	fCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	_, err = service.Forward(fCtx, req)
	cancel()
	if err != nil {
		return err
	}

	return nil
}
