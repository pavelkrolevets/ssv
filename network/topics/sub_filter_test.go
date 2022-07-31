package topics

import (
	forksfactory "github.com/bloxapp/ssv/network/forks/factory"
	forksprotocol "github.com/bloxapp/ssv/protocol/forks"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
)

func TestSubFilter(t *testing.T) {
	f := forksfactory.NewFork(forksprotocol.GenesisForkVersion)
	l := zap.L()
	sf := newSubFilter(l, f, 2)

	require.False(t, sf.CanSubscribe("xxx"))
	require.False(t, sf.CanSubscribe(f.GetTopicFullName("xxx")))
	sf.(Whitelist).Register(f.GetTopicFullName("1"))
	require.True(t, sf.CanSubscribe(f.GetTopicFullName("1")))
	require.False(t, sf.CanSubscribe(f.GetTopicFullName("2")))
}
