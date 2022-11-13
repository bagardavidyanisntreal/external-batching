package main

import (
	"context"
	"time"
)

type Client struct {
	api Service
}

func NewClient(api Service) *Client {
	return &Client{
		api: api,
	}
}

func (c Client) BatchRequest(ctx context.Context, batch Batch) error {
	batchSize, dur := c.api.GetLimits()
	if batchSize == 0 {
		return nil
	}
	var offset uint64
	errs := make(chan error)
	t := time.NewTicker(dur)
	defer t.Stop()
	go func() {
		defer close(errs)
		for {
			if offset+batchSize > uint64(len(batch)) {
				break
			}
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
				select {
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				case <-t.C:
					err := c.api.Process(ctx, batch[offset:offset+batchSize])
					if err != nil {
						errs <- err
						return
					}
					offset += batchSize
				}
			}
		}
		if len(batch[offset:]) > 0 {
			<-t.C
			err := c.api.Process(ctx, batch[offset:])
			if err != nil {
				errs <- err
				return
			}
		}
	}()
	for err := range errs {
		return err
	}
	return nil
}
