package topics

import (
	"context"

	specqbft "github.com/bloxapp/ssv-spec/qbft"
	spectypes "github.com/bloxapp/ssv-spec/types"
	"github.com/bloxapp/ssv/logging/fields"
	"github.com/bloxapp/ssv/network/forks"
	operatorstorage "github.com/bloxapp/ssv/operator/storage"
	"github.com/bloxapp/ssv/protocol/v2/ssv/queue"
	"github.com/bloxapp/ssv/protocol/v2/types"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

// MsgValidatorFunc represents a message validator
type MsgValidatorFunc = func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult

// NewSSVMsgValidator creates a new msg validator that validates message structure,
// and checks that the message was sent on the right topic.
// TODO: enable post SSZ change, remove logs, break into smaller validators?
func NewSSVMsgValidator(storage operatorstorage.Storage, fork forks.Fork) func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	return func(ctx context.Context, p peer.ID, pmsg *pubsub.Message) pubsub.ValidationResult {
		topic := pmsg.GetTopic()
		metricPubsubActiveMsgValidation.WithLabelValues(topic).Inc()
		defer metricPubsubActiveMsgValidation.WithLabelValues(topic).Dec()
		if len(pmsg.GetData()) == 0 {
			reportValidationResult(validationResultNoData)
			return pubsub.ValidationReject
		}
		msg, err := fork.DecodeNetworkMsg(pmsg.GetData())
		if err != nil {
			// can't decode message
			// logger.Debug("invalid: can't decode message", zap.Error(err))
			reportValidationResult(validationResultEncoding)
			return pubsub.ValidationReject
		}
		if msg == nil {
			reportValidationResult(validationResultEncoding)
			return pubsub.ValidationReject
		}
		pmsg.ValidatorData = *msg

		ssvMsg, err := queue.DecodeSSVMessage(zap.NewNop(), msg)
		if err != nil {
			zap.L().Debug("invalid: can't decode message", zap.Error(err))
			return pubsub.ValidationIgnore
		}
		switch m := ssvMsg.Body.(type) {
		case *specqbft.SignedMessage:
			share := storage.Shares().Get(msg.MsgID.GetPubKey())
			if share == nil {
				zap.L().Debug("msgval: share not found", fields.PubKey(msg.MsgID.GetPubKey()), zap.Error(err))
				return pubsub.ValidationIgnore
			}
			err := types.VerifyByOperators(m.Signature, m, [4]byte{0x00, 0x00, 0x30, 0x12}, spectypes.QBFTSignatureType, share.Committee)
			if err != nil {
				zap.L().Debug("msgval: invalid signature", fields.PubKey(msg.MsgID.GetPubKey()), zap.Error(err))
				return pubsub.ValidationIgnore
			}
		}

		return pubsub.ValidationAccept

		// Check if the message was sent on the right topic.
		// currentTopic := pmsg.GetTopic()
		// currentTopicBaseName := fork.GetTopicBaseName(currentTopic)
		// topics := fork.ValidatorTopicID(msg.GetID().GetPubKey())
		// for _, tp := range topics {
		//	if tp == currentTopicBaseName {
		//		reportValidationResult(validationResultValid)
		//		return pubsub.ValidationAccept
		//	}
		//}
		// reportValidationResult(validationResultTopic)
		// return pubsub.ValidationReject
	}
}

//// CombineMsgValidators executes multiple validators
// func CombineMsgValidators(validators ...MsgValidatorFunc) MsgValidatorFunc {
//	return func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
//		res := pubsub.ValidationAccept
//		for _, v := range validators {
//			if res = v(ctx, p, msg); res == pubsub.ValidationReject {
//				break
//			}
//		}
//		return res
//	}
//}
