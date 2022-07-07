package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/eventloop"
	"go.k6.io/k6/js/modulestest"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/testutils/httpmultibin"
	"go.k6.io/k6/metrics"
	"gopkg.in/guregu/null.v3"
)

func TestClientIsSupportedType(t *testing.T) {
	t.Parallel()

	t.Run("table tests", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			offset  int
			args    []interface{}
			wantErr bool
		}{
			{
				name:    "string is a supported type",
				offset:  1,
				args:    []interface{}{"foo"},
				wantErr: false,
			},
			{
				name:    "int is a supported type",
				offset:  1,
				args:    []interface{}{int(123)},
				wantErr: false,
			},
			{
				name:    "int64 is a supported type",
				offset:  1,
				args:    []interface{}{int64(123)},
				wantErr: false,
			},
			{
				name:    "float64 is a supported type",
				offset:  1,
				args:    []interface{}{float64(123)},
				wantErr: false,
			},
			{
				name:    "bool is a supported type",
				offset:  1,
				args:    []interface{}{bool(true)},
				wantErr: false,
			},
			{
				name:    "multiple identical types args are supported",
				offset:  1,
				args:    []interface{}{int(123), int(456)},
				wantErr: false,
			},
			{
				name:    "multiple mixed types args are supported",
				offset:  1,
				args:    []interface{}{int(123), "foo", bool(true)},
				wantErr: false,
			},
			{
				name:    "slice[T] is not a supported type",
				offset:  1,
				args:    []interface{}{[]string{"1", "2", "3"}},
				wantErr: true,
			},
			{
				name:    "multiple mixed valid and invalid types args are not supported",
				offset:  1,
				args:    []interface{}{int(123), []string{"1", "2", "3"}},
				wantErr: true,
			},
		}
		for _, tt := range tests {
			tt := tt

			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				c := &Client{}
				gotErr := c.isSupportedType(tt.offset, tt.args...)
				assert.Equal(t,
					tt.wantErr,
					gotErr != nil,
					"Client.isSupportedType() error = %v, wantErr %v", gotErr, tt.wantErr,
				)
			})
		}
	})

	t.Run("offset is respected in the error message", func(t *testing.T) {
		t.Parallel()

		c := &Client{}

		gotErr := c.isSupportedType(3, int(123), []string{"1", "2", "3"})

		assert.Error(t, gotErr)
		assert.Contains(t, gotErr.Error(), "argument at index: 4")
	})
}

type testSetup struct {
	tb      *httpmultibin.HTTPMultiBin
	rt      *goja.Runtime
	state   *lib.State
	samples chan metrics.SampleContainer
	ev      *eventloop.EventLoop
	redis   *miniredis.Miniredis
}

func newTestSetup(t testing.TB) testSetup {
	tb := httpmultibin.NewHTTPMultiBin(t)

	rt := goja.New()
	rt.SetFieldNameMapper(common.FieldNameMapper{})

	root, err := lib.NewGroup("", nil)
	require.NoError(t, err)

	samples := make(chan metrics.SampleContainer, 1000)

	state := &lib.State{
		Group:  root,
		Dialer: tb.Dialer,
		Options: lib.Options{
			SystemTags: metrics.NewSystemTagSet(
				metrics.TagURL,
				metrics.TagProto,
				metrics.TagStatus,
				metrics.TagSubproto,
			),
			UserAgent: null.StringFrom("TestUserAgent"),
		},
		Samples:        samples,
		TLSConfig:      tb.TLSClientConfig,
		BuiltinMetrics: metrics.RegisterBuiltinMetrics(metrics.NewRegistry()),
		Tags:           lib.NewTagMap(nil),
	}

	vu := &modulestest.VU{
		CtxField:     tb.Context,
		InitEnvField: &common.InitEnvironment{},
		RuntimeField: rt,
		StateField:   state,
	}

	m := new(RootModule).NewModuleInstance(vu)
	require.NoError(t, rt.Set("Client", m.Exports().Named["Client"]))

	ev := eventloop.New(vu)
	vu.RegisterCallbackField = ev.RegisterCallback

	return testSetup{
		rt:      rt,
		state:   state,
		samples: samples,
		ev:      ev,
		redis:   miniredis.RunT(t),
	}
}

func newInitContextTestSetup(t testing.TB) testSetup {
	rt := goja.New()
	rt.SetFieldNameMapper(common.FieldNameMapper{})

	samples := make(chan metrics.SampleContainer, 1000)

	var state *lib.State = nil

	vu := &modulestest.VU{
		CtxField:     context.Background(),
		InitEnvField: &common.InitEnvironment{},
		RuntimeField: rt,
		StateField:   state,
	}

	m := new(RootModule).NewModuleInstance(vu)
	require.NoError(t, rt.Set("Client", m.Exports().Named["Client"]))

	ev := eventloop.New(vu)
	vu.RegisterCallbackField = ev.RegisterCallback

	return testSetup{
		rt:      rt,
		state:   state,
		samples: samples,
		ev:      ev,
		redis:   miniredis.RunT(t),
	}
}
