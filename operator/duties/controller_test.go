package duties

import (
	"context"
	"sync"
	"testing"
	"time"

	eth2apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	spectypes "github.com/bloxapp/ssv-spec/types"
	"github.com/cornelk/hashmap"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/bloxapp/ssv/logging"
	"github.com/bloxapp/ssv/networkconfig"
	"github.com/bloxapp/ssv/operator/duties/mocks"
)

func TestDutyController_ListenToTicker(t *testing.T) {
	logger := logging.TestLogger(t)
	var wg sync.WaitGroup

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockExecutor := mocks.NewMockDutyExecutor(mockCtrl)
	mockExecutor.EXPECT().ExecuteDuty(gomock.Any(), gomock.Any()).DoAndReturn(func(logger *zap.Logger, duty *spectypes.Duty) error {
		require.NotNil(t, duty)
		require.True(t, duty.Slot > 0)
		wg.Done()
		return nil
	}).AnyTimes()

	mockFetcher := mocks.NewMockDutyFetcher(mockCtrl)
	mockFetcher.EXPECT().GetDuties(gomock.Any(), gomock.Any()).DoAndReturn(func(logger *zap.Logger, slot phase0.Slot) ([]spectypes.Duty, error) {
		return []spectypes.Duty{{Slot: slot, PubKey: phase0.BLSPubKey{}}}, nil
	}).AnyTimes()

	dutyCtrl := &dutyController{
		ctx:                    context.Background(),
		network:                networkconfig.TestNetwork,
		executor:               mockExecutor,
		fetcher:                mockFetcher,
		syncCommitteeDutiesMap: hashmap.New[uint64, *hashmap.Map[phase0.ValidatorIndex, *eth2apiv1.SyncCommitteeDuty]](),
	}

	cn := make(chan phase0.Slot)

	secPerSlot = 2
	defer func() {
		secPerSlot = 12
	}()

	currentSlot := dutyCtrl.network.Beacon.EstimatedCurrentSlot()

	go dutyCtrl.listenToTicker(logger, cn)
	wg.Add(2)
	go func() {
		cn <- currentSlot
		time.Sleep(time.Second * time.Duration(secPerSlot))
		cn <- currentSlot + 1
	}()

	wg.Wait()
}

func TestDutyController_ShouldExecute(t *testing.T) {
	logger := logging.TestLogger(t)
	ctrl := dutyController{network: networkconfig.TestNetwork}
	currentSlot := uint64(ctrl.network.Beacon.EstimatedCurrentSlot())

	require.True(t, ctrl.shouldExecute(logger, &spectypes.Duty{Slot: phase0.Slot(currentSlot), PubKey: phase0.BLSPubKey{}}))
	require.False(t, ctrl.shouldExecute(logger, &spectypes.Duty{Slot: phase0.Slot(currentSlot - 1000), PubKey: phase0.BLSPubKey{}}))
	require.False(t, ctrl.shouldExecute(logger, &spectypes.Duty{Slot: phase0.Slot(currentSlot + 1000), PubKey: phase0.BLSPubKey{}}))
}

func TestDutyController_GetSlotStartTime(t *testing.T) {
	d := dutyController{network: networkconfig.TestNetwork}

	ts := d.network.Beacon.GetSlotStartTime(646523)
	require.Equal(t, int64(1624266276), ts.Unix())
}

func TestDutyController_GetCurrentSlot(t *testing.T) {
	d := dutyController{network: networkconfig.TestNetwork}

	slot := d.network.Beacon.EstimatedCurrentSlot()
	require.Greater(t, slot, phase0.Slot(646855))
}

func TestDutyController_GetEpochFirstSlot(t *testing.T) {
	d := dutyController{network: networkconfig.TestNetwork}

	slot := d.network.Beacon.GetEpochFirstSlot(20203)
	require.EqualValues(t, 646496, slot)
}
