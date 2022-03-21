package wasm

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osmosis-labs/osmosis/v7/app/wasm"
)

func TestFullDenom(t *testing.T) {
	actor := RandomAccountAddress()

	specs := map[string]struct {
		addr         string
		subDenom     string
		expFullDenom string
		expErr       bool
	}{
		"valid address": {
			addr:         actor.String(),
			subDenom:     "sub_denom_1",
			expFullDenom: fmt.Sprintf("cw/%s/sub_denom_1", actor.String()),
		},
		"empty address": {
			addr:     "",
			subDenom: "sub_denom_1",
			expErr:   true,
		},
		"invalid address": {
			addr:     "invalid",
			subDenom: "sub_denom_1",
			expErr:   true,
		},
		"empty sub-denom": {
			addr:     actor.String(),
			subDenom: "",
			expErr:   true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// when
			gotFullDenom, gotErr := wasm.GetFullDenom(spec.addr, spec.subDenom)
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expFullDenom, gotFullDenom, "exp %s but got %s", spec.expFullDenom, gotFullDenom)
		})
	}

}
