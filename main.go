package main

import (
	"context"
	"os"
	"os/signal"
	samplepb "sdil-analytics/gen/protos"
	"syscall"
	"time"

	goakt "github.com/tochemey/goakt/v2/actors"
	"github.com/tochemey/goakt/v2/log"
)

var (
	lastCount = 0
)

func main() {
	ctx := context.Background()

	logger := log.DefaultLogger

	actorSystem, err := goakt.NewActorSystem("SampleActorSystem",
		goakt.WithExpireActorAfter(1*time.Second),
		goakt.WithLogger(logger),
		goakt.WithMailboxSize(1_000_000),
		goakt.WithActorInitMaxRetries(3),
	)

	if err != nil {
		logger.Error("Error creating actor system", err)
		return
	}

	err = actorSystem.Start(ctx)
	if err != nil {
		logger.Error("Error starting actor system", err)
		return
	}

	pinger := NewPinger()
	actor, err := actorSystem.Spawn(ctx, "Ping", pinger)
	for i := 0; i < 1_000; i++ {
		_ = goakt.Tell(ctx, actor, new(samplepb.Ping))
	}

	logger.Info("Sleeping for 5 seconds")
	time.Sleep(5*time.Second)

	logger.Info("Spawning again")
	actor, err = actorSystem.Spawn(ctx, "Ping", pinger)
	for i := 0; i < 1_000; i++ {
		_ = goakt.Tell(ctx, actor, new(samplepb.Ping))
	}

	interruptSignal := make(chan os.Signal, 1)
	signal.Notify(interruptSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interruptSignal

	_ = actorSystem.Stop(ctx)
}

type Pinger struct {
	count int
	logger log.Logger
}

var _ goakt.Actor = (*Pinger)(nil)

func NewPinger() *Pinger {
	return &Pinger{logger: log.DefaultLogger}
}

func (p *Pinger) PreStart(ctx context.Context) error {
	if lastCount != 0 {
		p.count = lastCount
	} else {
		p.count = 0
	}
	p.logger = log.DefaultLogger
	p.logger.Info("Starting pinger")
	return nil
}

func (p *Pinger) Receive(ctx goakt.ReceiveContext) {
	switch ctx.Message().(type) {
	case *samplepb.Ping:
		p.count++
		if p.count%100 == 0 {
			p.logger.Infof("count: %d", p.count)
		}
	default:
		ctx.Unhandled()
	}
}

func (p *Pinger) PostStop(ctx context.Context) error {
	p.logger.Info("Stopping pinger")
	lastCount = p.count
	return nil
}
