package topics

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/bloxapp/ssv-spec/qbft"
	spectypes "github.com/bloxapp/ssv-spec/types"
	"github.com/bloxapp/ssv-spec/types/testingutils"
	"github.com/bloxapp/ssv/operator/validator/mocks"
	"github.com/golang/mock/gomock"
	"github.com/herumi/bls-eth-go-binary/bls"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ps_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
	"time"

	"github.com/bloxapp/ssv/network/forks/genesis"
	"github.com/bloxapp/ssv/protocol/v2/types"
	"github.com/bloxapp/ssv/utils/threshold"
)

const peerID = "16Uiu2HAkyWQyCb6reWXGQeBUt9EXArk6h3aq3PsFMwLNq3pPGH1r"

func TestMsgValidator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	pks := createSharePublicKeys(4)
	logger := zap.L()
	f := genesis.ForkGenesis{}
	controller := mocks.NewMockController(ctrl)
	mv := NewSSVMsgValidator(context.Background(), &f, controller, *logger)
	controller.EXPECT().GetShare(gomock.Any()).Return(&types.SSVShare{}, nil)
	require.NotNil(t, mv)

	t.Run("valid consensus msg", func(t *testing.T) {
		pkHex := pks[0]
		msg, err := dummySSVConsensusMsg(pkHex, 15160)
		require.NoError(t, err)
		raw, err := msg.Encode()
		require.NoError(t, err)
		pk, err := hex.DecodeString(pkHex)
		require.NoError(t, err)
		topics := f.ValidatorTopicID(pk)
		pmsg := newPBMsg(raw, f.GetTopicFullName(topics[0]), []byte(peerID))
		res := mv(context.Background(), "peerID", pmsg)
		// TODO: make it accept or delete test
		require.Equal(t, res, pubsub.ValidationAccept)
	})

	// TODO: enable once topic validation is in place
	// t.Run("wrong topic", func(t *testing.T) {
	//	pkHex := "b5de683dbcb3febe8320cc741948b9282d59b75a6970ed55d6f389da59f26325331b7ea0e71a2552373d0debb6048b8a"
	//	msg, err := dummySSVConsensusMsg(pkHex, 15160)
	//	require.NoError(t, err)
	//	raw, err := msg.Encode()
	//	require.NoError(t, err)
	//	pk, err := hex.DecodeString("a297599ccf617c3b6118bbd248494d7072bb8c6c1cc342ea442a289415987d306bad34415f89469221450a2501a832ec")
	//	require.NoError(t, err)
	//	topics := f.ValidatorTopicID(pk)
	//	pmsg := newPBMsg(raw, topics[0], []byte("peerID"))
	//	res := mv("peerID", pmsg)
	//	require.Equal(t, res, pubsub.ValidationReject)
	// })

	t.Run("empty message", func(t *testing.T) {
		pmsg := newPBMsg([]byte{}, "xxx", []byte{})
		res := mv(context.Background(), "xxxx", pmsg)
		require.Equal(t, res, pubsub.ValidationReject)
	})

	// TODO: enable once topic validation is in place
	// t.Run("invalid validator public key", func(t *testing.T) {
	//	msg, err := dummySSVConsensusMsg("10101011", 1)
	//	require.NoError(t, err)
	//	raw, err := msg.Encode()
	//	require.NoError(t, err)
	//	pmsg := newPBMsg(raw, "xxx", []byte{})
	//	res := mv(context.Background(), "xxxx", pmsg)
	//	require.Equal(t, res, pubsub.ValidationReject)
	// })
}

func TestSSVMsgValidator_QBFT(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	logger := zap.L()
	f := genesis.ForkGenesis{}
	controller := mocks.NewMockController(ctrl)
	ks := testingutils.Testing4SharesSet()
	valPK := ks.ValidatorPK.Serialize()
	controller.EXPECT().GetShare(gomock.Eq(valPK)).Return(&types.SSVShare{
		Share: spectypes.Share{
			OperatorID:      0,
			ValidatorPubKey: valPK,
			SharePubKey:     nil,
			Committee: []*spectypes.Operator{
				{spectypes.OperatorID(1), ks.Shares[1].GetPublicKey().Serialize()},
				{spectypes.OperatorID(2), ks.Shares[2].GetPublicKey().Serialize()},
				{spectypes.OperatorID(3), ks.Shares[3].GetPublicKey().Serialize()},
				{spectypes.OperatorID(4), ks.Shares[4].GetPublicKey().Serialize()},
			},
			Quorum:              ks.Threshold,
			PartialQuorum:       ks.PartialThreshold,
			DomainType:          testingutils.TestingSSVDomainType,
			FeeRecipientAddress: [20]byte{},
			Graffiti:            nil,
		},
	}, nil).AnyTimes()
	share, _ := controller.GetShare(valPK)

	t.Run("simple happy flow", func(t *testing.T) {
		mv := NewSSVMsgValidator(context.Background(), &f, controller, *logger)
		require.NotNil(t, mv)
		msgs := []*qbft.SignedMessage{
			testingutils.TestingProposalMessage(ks.Shares[1], spectypes.OperatorID(1)),

			testingutils.TestingPrepareMessage(ks.Shares[1], spectypes.OperatorID(1)),
			testingutils.TestingPrepareMessage(ks.Shares[2], spectypes.OperatorID(2)),
			testingutils.TestingPrepareMessage(ks.Shares[3], spectypes.OperatorID(3)),

			testingutils.TestingCommitMessage(ks.Shares[1], spectypes.OperatorID(1)),
			testingutils.TestingCommitMessage(ks.Shares[2], spectypes.OperatorID(2)),
			testingutils.TestingCommitMessage(ks.Shares[3], spectypes.OperatorID(3)),

			//Decided message
			testingutils.TestingCommitMultiSignerMessage(sliceMapValues(ks.Shares, 1, 4), sliceOpIDs(1, 4)),

			testingutils.TestingProposalMessageWithHeight(ks.Shares[1], spectypes.OperatorID(1), 1),

			testingutils.TestingPrepareMessageWithHeight(ks.Shares[1], spectypes.OperatorID(1), 1),
			testingutils.TestingPrepareMessageWithHeight(ks.Shares[2], spectypes.OperatorID(2), 1),
			testingutils.TestingPrepareMessageWithHeight(ks.Shares[3], spectypes.OperatorID(3), 1),

			testingutils.TestingCommitMessageWithHeight(ks.Shares[1], spectypes.OperatorID(1), 1),
			testingutils.TestingCommitMessageWithHeight(ks.Shares[2], spectypes.OperatorID(2), 1),
			testingutils.TestingCommitMessageWithHeight(ks.Shares[3], spectypes.OperatorID(3), 1),

			//Decided message
			testingutils.TestingCommitMultiSignerMessageWithHeight(sliceMapValues(ks.Shares, 1, 4), sliceOpIDs(1, 4), 1),
		}

		sszMessages := wrapSignedMessages(t, msgs, spectypes.BNRoleAttester, spectypes.SSVConsensusMsgType, share)
		for i, msg := range sszMessages {
			raw, err := msg.Encode()
			require.NoError(t, err)
			topics := f.ValidatorTopicID(valPK)
			pmsg := newPBMsg(raw, f.GetTopicFullName(topics[0]), []byte(peerID))
			validationResult := mv(context.Background(), peerID, pmsg)
			require.Equal(t, pubsub.ValidationAccept, validationResult, "failed on message %d", i+1)
		}
	})

	t.Run("roundchange comes too soon", func(t *testing.T) {
		mv := NewSSVMsgValidator(context.Background(), &f, controller, *logger)
		require.NotNil(t, mv)

		msgs := []*qbft.SignedMessage{
			testingutils.TestingProposalMessage(ks.Shares[1], spectypes.OperatorID(1)),

			testingutils.TestingPrepareMessage(ks.Shares[1], spectypes.OperatorID(1)),
			testingutils.TestingPrepareMessage(ks.Shares[2], spectypes.OperatorID(2)),
			testingutils.TestingPrepareMessage(ks.Shares[3], spectypes.OperatorID(3)),

			testingutils.TestingRoundChangeMessageWithRound(ks.Shares[1], spectypes.OperatorID(1), 2),

			testingutils.TestingCommitMessage(ks.Shares[1], spectypes.OperatorID(1)),
			testingutils.TestingCommitMessage(ks.Shares[2], spectypes.OperatorID(2)),
			testingutils.TestingCommitMessage(ks.Shares[3], spectypes.OperatorID(3)),

			//Decided message
			testingutils.TestingCommitMultiSignerMessage(sliceMapValues(ks.Shares, 1, 4), sliceOpIDs(1, 4)),
		}
		sszMessages := wrapSignedMessages(t, msgs, spectypes.BNRoleAttester, spectypes.SSVConsensusMsgType, share)

		for i, msg := range sszMessages {
			raw, err := msg.Encode()
			require.NoError(t, err)
			topics := f.ValidatorTopicID(valPK)
			pmsg := newPBMsg(raw, f.GetTopicFullName(topics[0]), []byte("peerID"))
			validationResult := mv(context.Background(), "peerID", pmsg)
			// if roundchange message, expect reject
			if i == 4 {
				require.Equal(t, pubsub.ValidationReject, validationResult, "failed on message %d", i+1)
				continue
			}
			require.Equal(t, pubsub.ValidationAccept, validationResult, "failed on message %d", i+1)
		}
	})

	t.Run("roundchange comes on time", func(t *testing.T) {
		mv := NewSSVMsgValidator(context.Background(), &f, controller, *logger)
		require.NotNil(t, mv)

		msgs := []*qbft.SignedMessage{
			testingutils.TestingProposalMessage(ks.Shares[1], spectypes.OperatorID(1)),

			testingutils.TestingPrepareMessage(ks.Shares[1], spectypes.OperatorID(1)),
			testingutils.TestingPrepareMessage(ks.Shares[2], spectypes.OperatorID(2)),
			testingutils.TestingPrepareMessage(ks.Shares[3], spectypes.OperatorID(3)),

			testingutils.TestingRoundChangeMessageWithRound(ks.Shares[1], spectypes.OperatorID(1), 2),

			// this comes from an old round by the signer
			testingutils.TestingCommitMessage(ks.Shares[1], spectypes.OperatorID(1)),
			//this comes from the new round.. commit message doesn't have prepared message but scheduler doesn't care
			testingutils.TestingCommitMessageWithRound(ks.Shares[1], spectypes.OperatorID(1), 2),
			testingutils.TestingCommitMessage(ks.Shares[2], spectypes.OperatorID(2)),
			testingutils.TestingCommitMessage(ks.Shares[3], spectypes.OperatorID(3)),

			//Decided message
			testingutils.TestingCommitMultiSignerMessage(sliceMapValues(ks.Shares, 1, 4), sliceOpIDs(1, 4)),
		}
		sszMessages := wrapSignedMessages(t, msgs, spectypes.BNRoleAttester, spectypes.SSVConsensusMsgType, share)

		for i, msg := range sszMessages {
			raw, err := msg.Encode()
			require.NoError(t, err)
			topics := f.ValidatorTopicID(valPK)
			pmsg := newPBMsg(raw, f.GetTopicFullName(topics[0]), []byte("peerID"))
			// if roundchange message, wait timeout
			if i == 4 {
				// TODO: mock time
				<-time.After(2 * time.Second)
			}
			validationResult := mv(context.Background(), "peerID", pmsg)
			if i == 5 {
				// reject message with past round
				require.Equal(t, pubsub.ValidationReject, validationResult, "failed on message %d", i+1)
			} else {
				// TODO: change test when slack changes
				require.Equal(t, pubsub.ValidationAccept, validationResult, "failed on message %d", i+1)
			}
		}
	})
}

func BenchmarkSSVMsgValidator(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()
	logger := zap.L()
	f := genesis.ForkGenesis{}
	controller := mocks.NewMockController(ctrl)
	ks := testingutils.Testing4SharesSet()
	valPK := ks.ValidatorPK.Serialize()
	controller.EXPECT().GetShare(gomock.Eq(valPK)).Return(&types.SSVShare{
		Share: spectypes.Share{
			OperatorID:      0,
			ValidatorPubKey: valPK,
			SharePubKey:     nil,
			Committee: []*spectypes.Operator{
				{spectypes.OperatorID(1), ks.Shares[1].GetPublicKey().Serialize()},
				{spectypes.OperatorID(2), ks.Shares[2].GetPublicKey().Serialize()},
				{spectypes.OperatorID(3), ks.Shares[3].GetPublicKey().Serialize()},
				{spectypes.OperatorID(4), ks.Shares[4].GetPublicKey().Serialize()},
			},
			Quorum:              ks.Threshold,
			PartialQuorum:       ks.PartialThreshold,
			DomainType:          testingutils.TestingSSVDomainType,
			FeeRecipientAddress: [20]byte{},
			Graffiti:            nil,
		},
	}, nil).AnyTimes()
	share, _ := controller.GetShare(valPK)
	mv := NewSSVMsgValidator(context.Background(), &f, controller, *logger)
	require.NotNil(b, mv)
	msgs := []*qbft.SignedMessage{
		testingutils.TestingProposalMessage(ks.Shares[1], spectypes.OperatorID(1)),

		testingutils.TestingPrepareMessage(ks.Shares[1], spectypes.OperatorID(1)),
		testingutils.TestingPrepareMessage(ks.Shares[2], spectypes.OperatorID(2)),
		testingutils.TestingPrepareMessage(ks.Shares[3], spectypes.OperatorID(3)),

		testingutils.TestingRoundChangeMessageWithRound(ks.Shares[1], spectypes.OperatorID(1), 2),

		testingutils.TestingCommitMessage(ks.Shares[1], spectypes.OperatorID(1)),
		testingutils.TestingCommitMessage(ks.Shares[2], spectypes.OperatorID(2)),
		testingutils.TestingCommitMessage(ks.Shares[3], spectypes.OperatorID(3)),

		//Decided message
		testingutils.TestingCommitMultiSignerMessage(sliceMapValues(ks.Shares, 1, 4), sliceOpIDs(1, 4)),
	}

	sszMessages := wrapSignedMessages(b, msgs, spectypes.BNRoleAttester, spectypes.SSVConsensusMsgType, share)
	for i, msg := range sszMessages {
		raw, err := msg.Encode()
		require.NoError(b, err)
		topics := f.ValidatorTopicID(valPK)
		pmsg := newPBMsg(raw, f.GetTopicFullName(topics[0]), []byte(peerID))
		b.Run(fmt.Sprintf("message %d", i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				mv(context.Background(), peerID, pmsg)
			}
		})
	}
}

func sliceOpIDs(lo int, hi int) []spectypes.OperatorID {
	var res []spectypes.OperatorID
	for i := lo; i < hi; i++ {
		res = append(res, spectypes.OperatorID(i))
	}
	return res
}

// sliceMapValues returns a slice of values from a map with int keys from lo (inclusive) to hi (exclusive)
// TODO: move to utils
func sliceMapValues[K int | uint64, T any](mapToSlice map[K]*T, lo K, hi K) []*T {
	var res []*T
	for i := lo; i < hi; i++ {
		res = append(res, mapToSlice[i])
	}
	return res
}

func wrapSignedMessages(t require.TestingT, msgs []*qbft.SignedMessage, role spectypes.BeaconRole, msgType spectypes.MsgType, share *types.SSVShare) []*spectypes.SSVMessage {
	var res []*spectypes.SSVMessage
	for _, msg := range msgs {
		res = append(res, wrapSignedMessage(t, msg, role, msgType, share))
	}
	return res
}

func wrapSignedMessage(t require.TestingT, msg *qbft.SignedMessage, role spectypes.BeaconRole, msgType spectypes.MsgType, share *types.SSVShare) *spectypes.SSVMessage {
	mid := spectypes.NewMsgID(share.DomainType, share.ValidatorPubKey, role)
	ssz, err := msg.MarshalSSZ()
	require.NoError(t, err, "failed to marshal message")
	return &spectypes.SSVMessage{
		MsgType: msgType,
		MsgID:   mid,
		Data:    ssz,
	}
}

func createSharePublicKeys(n int) []string {
	threshold.Init()

	var res []string
	for i := 0; i < n; i++ {
		sk := bls.SecretKey{}
		sk.SetByCSPRNG()
		pk := sk.GetPublicKey().SerializeToHexStr()
		res = append(res, pk)
	}
	return res
}

func newPBMsg(data []byte, topic string, from []byte) *pubsub.Message {
	pmsg := &pubsub.Message{
		Message: &ps_pb.Message{},
	}
	pmsg.Data = data
	pmsg.Topic = &topic
	pmsg.From = from
	return pmsg
}

func dummySSVConsensusMsg(pkHex string, height int) (*spectypes.SSVMessage, error) {
	pk, err := hex.DecodeString(pkHex)
	if err != nil {
		return nil, err
	}
	id := spectypes.NewMsgID(types.GetDefaultDomain(), pk, spectypes.BNRoleAttester)
	msgData := fmt.Sprintf(`{
	  "message": {
		"type": 3,
		"round": 2,
		"identifier": "%s",
		"height": %d,
		"value": "bk0iAAAAAAACAAAAAAAAAAbYXFSt2H7SQd5q5u+N0bp6PbbPTQjU25H1QnkbzTECahIBAAAAAADmi+NJfvXZ3iXp2cfs0vYVW+EgGD7DTTvr5EkLtiWq8WsSAQAAAAAAIC8dZTEdD3EvE38B9kDVWkSLy40j0T+TtSrrrBqVjo4="
	  },
	  "signature": "sVV0fsvqQlqliKv/ussGIatxpe8LDWhc9uoaM5WpjbiYvvxUr1eCpz0ja7UT1PGNDdmoGi6xbMC1g/ozhAt4uCdpy0Xdfqbv2hMf2iRL5ZPKOSmMifHbd8yg4PeeceyN",
	  "signer_ids": [1,3,4]
	}`, id, height)
	return &spectypes.SSVMessage{
		MsgType: spectypes.SSVConsensusMsgType,
		MsgID:   id,
		Data:    []byte(msgData),
	}, nil
}
