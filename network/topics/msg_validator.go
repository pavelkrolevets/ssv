package topics

import (
	"context"
	"github.com/bloxapp/ssv-spec/qbft"
	spectypes "github.com/bloxapp/ssv-spec/types"
	"github.com/bloxapp/ssv/logging/fields"
	"github.com/bloxapp/ssv/network/forks"
	"github.com/bloxapp/ssv/operator/validator"
	"github.com/bloxapp/ssv/protocol/v2/qbft/controller"
	"github.com/bloxapp/ssv/protocol/v2/types"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"

	"go.uber.org/zap"
)

// MsgValidatorFunc represents a message validator
type MsgValidatorFunc = func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult

// TODO change ugly scope ...
//var sigChan = make(chan *signatureVerifier, verifierLimit)

// NewSSVMsgValidator creates a new msg validator that validates message structure,
// and checks that the message was sent on the right topic.
// TODO - copying plogger may cause GC issues, consider using a pool
func NewSSVMsgValidator(fork forks.Fork, valController validator.Controller, plogger zap.Logger) MsgValidatorFunc {
	schedule := NewMessageSchedule()
	//go verifierRoutine(ctx, sigChan)
	return func(ctx context.Context, p peer.ID, pmsg *pubsub.Message) pubsub.ValidationResult {
		plog := plogger.With(zap.String("peerID", pmsg.GetFrom().String()))
		plog.Debug("msg validation started")
		topic := pmsg.GetTopic()
		metricPubsubActiveMsgValidation.WithLabelValues(topic).Inc()
		defer metricPubsubActiveMsgValidation.WithLabelValues(topic).Dec()
		if len(pmsg.GetData()) == 0 {
			reportValidationResult(validationResultNoData, plog, nil, "")
			return pubsub.ValidationReject
		}
		msg, err := fork.DecodeNetworkMsg(pmsg.GetData())
		if err != nil {
			// can't decode message
			// logger.Debug("invalid: can't decode message", zap.Error(err))
			reportValidationResult(validationResultEncoding, plog, err, "")
			return pubsub.ValidationReject
		}
		if msg == nil {
			reportValidationResult(validationResultEncoding, plog, nil, "")
			return pubsub.ValidationReject
		}
		pmsg.ValidatorData = *msg

		//if valFunc == nil {
		//	plogger.Warn("no message validator provided", zap.String("topic", topic))
		//	return pubsub.ValidationAccept
		//}

		return ValidateSSVMsg(msg, valController, schedule, plog)

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

func ValidateSSVMsg(msg *spectypes.SSVMessage, valController validator.Controller, schedule *MessageSchedule, plogger *zap.Logger) pubsub.ValidationResult {
	plogger = plogger.With(fields.PubKey(msg.MsgID.GetPubKey()), fields.Role(msg.MsgID.GetRoleType()))
	switch msg.MsgType {
	case spectypes.SSVConsensusMsgType:
		share, err := valController.GetShare(msg.MsgID.GetPubKey())
		if err != nil || share == nil {
			reportValidationResult(validationResultNoValidator, plogger, err, "")
			return pubsub.ValidationReject
		}
		signedMsg := qbft.SignedMessage{}
		err = signedMsg.Decode(msg.GetData())
		if err != nil {
			reportValidationResult(validationResultEncoding, plogger, err, "")
			return pubsub.ValidationReject
		}
		return validateConsensusMsg(&signedMsg, share, schedule, plogger)
	default:
		return pubsub.ValidationAccept
	}
}

/*
Main controller processing flow

All decided msgs are processed the same, out of instance
All valid future msgs are saved in a container and can trigger the highest decided future msg
All other msgs (not future or decided) are processed normally by an existing instance (if found)
*/
func validateConsensusMsg(signedMsg *qbft.SignedMessage, share *types.SSVShare, schedule *MessageSchedule, plogger *zap.Logger) pubsub.ValidationResult {
	plogger = plogger.With(zap.Any("msgType", signedMsg.Message.MsgType), zap.Any("msgRound", signedMsg.Message.Round),
		zap.Any("msgHeight", signedMsg.Message.Height), zap.Any("signers", signedMsg.Signers))

	plogger.Info("validating consensus message")

	// syntactic check
	if signedMsg.Validate() != nil {
		reportValidationResult(ValidationResultSyntacticCheck, plogger, nil, "")
		return pubsub.ValidationReject
	}

	// TODO can this be inside another syntactic check? or maybe we can move more syntactic checks here that happen later on?
	if len(signedMsg.Signers) > len(share.Committee) {
		reportValidationResult(ValidationResultSyntacticCheck, plogger, nil, "too many signers")
		return pubsub.ValidationReject
	}

	// if isDecided msg (this propagates to all topics)
	if controller.IsDecidedMsg(&share.Share, signedMsg) {
		return validateDecideMessage(signedMsg, schedule, plogger, share)
	}

	// if non-decided msg and I am a committee-validator
	// is message is timely?
	// if no instance of qbft
	// base validation
	// sig validation
	// mark message

	// if qbft instance is decided (I am a validator)
	// base commit message validation
	// sig commit message validation
	// is commit msg aggratable
	// mark message

	// If qbft instance is not decided (I am a validator)
	// Full validation of messages
	//mark consensus message

	return validateQbftMessage(schedule, signedMsg, plogger, share)
}

func validateQbftMessage(schedule *MessageSchedule, signedMsg *qbft.SignedMessage, plogger *zap.Logger, share *types.SSVShare) pubsub.ValidationResult {
	plogger.Info("validating qbft message")
	markLockID := markLockID(signedMsg.Message.Identifier, signedMsg.Signers[0])
	// If we don't lock we may have a race between findMark and markConsensusMsg
	schedule.sto.Lock(markLockID)
	defer schedule.sto.Unlock(markLockID)

	signerMark, found := schedule.findMark(signedMsg.Message.Identifier, signedMsg.Signers[0])
	if found {
		highestRound := signerMark.HighestRound
		plogger = plogger.With(zap.Any("markedRound", highestRound), zap.Any("markedHeight", signerMark.HighestDecided))
		minRoundTime := schedule.minFirstMsgTimeForRound(signedMsg, highestRound, plogger)

		isTimely, valResult := signerMark.isConsensusMsgTimely(signedMsg, plogger, minRoundTime)
		if !isTimely {
			reportValidationResult(ValidationResultNotTimely, plogger, nil, "non timely qbft message")
			return valResult
		}
		if signerMark.tooManyMessagesPerRound(signedMsg, share, plogger) {
			reportValidationResult(ValidationResultTooManyMsgs, plogger, nil, "too many messages per round")
			return pubsub.ValidationReject
		}
	}
	// sig validation
	//_, err := validateWithBatchVerifier(ctx, signedMsg, share.DomainType, spectypes.QBFTSignatureType, share.Committee, sigChan, plogger)
	//if err != nil {
	//	reportValidationResult(ValidationResultInvalidSig, plogger, err, "invalid signature on qbft message")
	//	return pubsub.ValidationReject
	//}

	// mark message
	schedule.MarkConsensusMessage(signerMark, signedMsg.Message.Identifier, signedMsg.Signers[0], signedMsg.Message.Round, signedMsg.Message.MsgType, plogger)

	// base commit message validation
	// sig commit message validation
	// is commit msg aggratable
	// mark message
	reportValidationResult(validationResultOk, plogger, nil, "")
	return pubsub.ValidationAccept
}

func validateDecideMessage(signedCommit *qbft.SignedMessage, schedule *MessageSchedule, plogger *zap.Logger, share *types.SSVShare) pubsub.ValidationResult {
	plogger = plogger.
		With(zap.String("msgType", "decided")).
		With(zap.Any("signers", signedCommit.Signers)).
		With(zap.Any("height", signedCommit.Message.Height))

	plogger.Info("validate decided message")

	// TODO we don't need to lock the scheduler here, because even if validateQbftMessage creates a mark here it shouldn't affect the result
	schedule.sto.Lock(markID(signedCommit.Message.Identifier))
	defer schedule.sto.Unlock(markID(signedCommit.Message.Identifier))
	// this decision is made to reduce lock retention, but it may make the code more fragile to changes
	// when we create the mark we do lock the scheduler, so we are safe

	// check if better decided
	// Mainly to have better statistics on commitments with a full signer set
	// TODO can it cause liveness failure?
	// if a different message is not better, then theoretically a better message should be broadcasted by at least one node
	if schedule.hasBetterMsg(signedCommit) {
		reportValidationResult(ValidationResultBetterMessage, plogger, nil, "better decided message")
		return pubsub.ValidationIgnore
	}

	//check if timely decided
	if !schedule.isTimelyDecidedMsg(signedCommit, plogger) {
		reportValidationResult(ValidationResultNotTimely, plogger, nil, "not timely decided message")
		return pubsub.ValidationReject
	}

	// validate decided message
	// TODO calls signedMsg.validate() again, this call does some useless stuff
	// TODO should do deeper syntax validation for commit data? Or eth protocol protects well enough?
	// IMO such validation matters less for DOS protection so we shouldn't spend too much time on it
	if err := controller.ValidateDecidedSyntactically(signedCommit, &share.Share); err != nil {
		reportValidationResult(ValidationResultSyntacticCheck, plogger, err, "decided message is not syntactically valid")
		return pubsub.ValidationReject
	}

	// sig validation
	//_, err := validateWithBatchVerifier(ctx, signedCommit, share.DomainType, spectypes.QBFTSignatureType, share.Committee, sigChan, plogger)
	//if err != nil {
	//	reportValidationResult(ValidationResultInvalidSig, plogger, err, "invalid signature on decided message")
	//	return pubsub.ValidationReject
	//}
	//mark decided message
	schedule.markDecidedMsg(signedCommit, share, plogger)
	return pubsub.ValidationAccept
}
