package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClient_BatchRequest(t *testing.T) {
	t.Parallel()
	tt := map[string]struct {
		mockF func(ctx context.Context, api *MockService) context.Context
		batch Batch
		err   error
	}{
		"zero batch size": {
			mockF: func(ctx context.Context, api *MockService) context.Context {
				api.On("GetLimits").
					Return(uint64(0), time.Minute*2).
					Once()
				return context.Background()
			},
			batch: make(Batch, 66),
		},
		"cancel ctx": {
			mockF: func(ctx context.Context, api *MockService) context.Context {
				api.On("GetLimits").
					Return(uint64(12), time.Minute*2).
					Once()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				return ctx
			},
			batch: make(Batch, 100),
			err:   errors.New("context canceled"),
		},
		"process err on take num 5": {
			mockF: func(ctx context.Context, api *MockService) context.Context {
				api.On("GetLimits").
					Return(uint64(13), time.Second).
					Once()
				api.On("Process", ctx, mock.Anything).
					Run(func(args mock.Arguments) {
						fmt.Println(args) // to illustrate request params
					}).
					Return(nil).
					Times(4)
				api.On("Process", ctx, mock.Anything).
					Run(func(args mock.Arguments) {
						fmt.Println("now something went wrong!")
					}).
					Return(errors.New("something went wrong! break")).
					Once()
				return ctx
			},
			batch: make(Batch, 100),
			err:   errors.New("something went wrong! break"),
		},
		"all ok, batched by 13 item in batch each second": {
			mockF: func(ctx context.Context, api *MockService) context.Context {
				api.On("GetLimits").
					Return(uint64(13), time.Second).
					Once()
				api.On("Process", ctx, mock.Anything).
					Run(func(args mock.Arguments) {
						fmt.Println(args)
					}).
					Return(nil)
				return ctx
			},
			batch: make(Batch, 100),
		},
	}

	for name, tc := range tt {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			api := NewMockService(t)
			client := NewClient(api)
			ctx := context.Background()
			if tc.mockF != nil {
				ctx = tc.mockF(ctx, api)
			}
			err := client.BatchRequest(ctx, tc.batch)
			require.Equal(t, tc.err, err)
		})
	}
}
