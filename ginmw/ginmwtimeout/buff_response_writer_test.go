package ginmwtimeout

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelevantCaller(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		frame := relevantCaller()
		require.Equal(t, "testing.tRunner", frame.Function)
	})
	t.Run("nested-in-nonrelevant", func(t *testing.T) {
		foo := func() {
			frame := relevantCaller()
			require.Equal(t, "testing.tRunner", frame.Function)
		}
		foo()
	})
}
