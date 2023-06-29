package eventdispatcher

import (
	"github.com/bloxapp/ssv/eth/eventbatcher"
)

type eventDataHandler interface {
	HandleBlockEventsStream(blockEvents <-chan eventbatcher.BlockEvents, executeTasks bool) (uint64, error)
}
