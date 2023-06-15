package testing

import (
	"context"
	"fmt"
	"time"

	spectypes "github.com/bloxapp/ssv-spec/types"
	"go.uber.org/zap"

	qbftstorage "github.com/bloxapp/ssv/ibft/storage"
	"github.com/bloxapp/ssv/storage"
	"github.com/bloxapp/ssv/storage/basedb"
)

func getDB(logger *zap.Logger) basedb.IDb {
	dbInstance, err := storage.GetStorageFactory(logger, basedb.Options{
		Type:      "badger-memory",
		Path:      fmt.Sprintf("/tmp/badger-%d", time.Now().UnixNano()),
		Reporting: false,
		Ctx:       context.TODO(),
	})
	if err != nil {
		panic(err)
	}
	return dbInstance
}

var allRoles = []spectypes.BeaconRole{
	spectypes.BNRoleAttester,
	spectypes.BNRoleProposer,
	spectypes.BNRoleAggregator,
	spectypes.BNRoleSyncCommittee,
	spectypes.BNRoleSyncCommitteeContribution,
	spectypes.BNRoleValidatorRegistration,
}

func TestingStores(logger *zap.Logger) *qbftstorage.QBFTStores {
	return qbftstorage.NewStoresFromRoles(getDB(logger), allRoles...)
}
