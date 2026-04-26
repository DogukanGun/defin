package chain

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

type BlockHandler func(ctx context.Context, header *types.Header) error

type Subscriber struct {
	client    *Client
	log       *slog.Logger
	handler   BlockHandler
	blockTime time.Duration
}

func NewSubscriber(client *Client, handler BlockHandler, blockTime time.Duration, log *slog.Logger) *Subscriber {
	if blockTime <= 0 {
		blockTime = 2 * time.Second
	}
	return &Subscriber{
		client:    client,
		handler:   handler,
		blockTime: blockTime,
		log:       log,
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	ws := s.client.WS()
	if ws != nil {
		headers := make(chan *types.Header, 16)
		sub, err := ws.SubscribeNewHead(ctx, headers)
		if err == nil {
			s.log.Info("subscribed to new blocks", "chain", s.client.Name())
			go s.runWS(ctx, sub, headers)
			return nil
		}
		s.log.Warn("eth_subscribe unavailable, falling back to polling",
			"chain", s.client.Name(), "error", err)
	}

	// Polling fallback: fetch block header every blockTime interval
	s.log.Info("polling for new blocks", "chain", s.client.Name(), "interval", s.blockTime)
	go s.runPoll(ctx)
	return nil
}

func (s *Subscriber) runWS(ctx context.Context, sub interface{ Err() <-chan error; Unsubscribe() }, headers <-chan *types.Header) {
	defer sub.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			s.log.Info("block subscriber stopped", "chain", s.client.Name())
			return
		case err := <-sub.Err():
			s.log.Error("subscription error", "chain", s.client.Name(), "error", err)
			s.reconnect(ctx)
			return
		case header := <-headers:
			start := time.Now()
			if err := s.handler(ctx, header); err != nil {
				_ = err
			}
			s.log.Debug("block processed",
				"chain", s.client.Name(),
				"block", header.Number.Uint64(),
				"duration", time.Since(start),
			)
		}
	}
}

func (s *Subscriber) runPoll(ctx context.Context) {
	ticker := time.NewTicker(s.blockTime)
	defer ticker.Stop()

	var lastBlock uint64
	for {
		select {
		case <-ctx.Done():
			s.log.Info("block poller stopped", "chain", s.client.Name())
			return
		case <-ticker.C:
			num, err := s.client.BlockNumber(ctx)
			if err != nil {
				s.log.Warn("poll block number failed", "chain", s.client.Name(), "error", err)
				continue
			}
			if num <= lastBlock {
				continue
			}
			lastBlock = num

			header, err := s.client.HTTP().HeaderByNumber(ctx, nil)
			if err != nil {
				s.log.Warn("poll header failed", "chain", s.client.Name(), "error", err)
				continue
			}
			start := time.Now()
			if err := s.handler(ctx, header); err != nil {
				_ = err
			}
			s.log.Debug("block processed (poll)",
				"chain", s.client.Name(),
				"block", header.Number.Uint64(),
				"duration", time.Since(start),
			)
		}
	}
}

func (s *Subscriber) reconnect(ctx context.Context) {
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		delay := time.Duration(attempt) * time.Second
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		s.log.Info("reconnecting", "chain", s.client.Name(), "attempt", attempt, "delay", delay)
		time.Sleep(delay)

		if err := s.Start(ctx); err != nil {
			s.log.Error("reconnect failed", "chain", s.client.Name(), "error", fmt.Sprintf("%v", err))
			continue
		}
		return
	}
}
