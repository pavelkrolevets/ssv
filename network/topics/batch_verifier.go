package topics

import (
	"context"
	specqbft "github.com/bloxapp/ssv-spec/qbft"
	spectypes "github.com/bloxapp/ssv-spec/types"
	bls "github.com/herumi/bls-eth-go-binary/bls"
	"go.uber.org/zap"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
)

const signatureVerificationInterval = 50 * time.Millisecond

// Same as prysm
const verifierLimit = 50

type signatureVerifier struct {
	data []byte
	sig  *bls.Sign
	pk   *bls.PublicKey

	//TODO: use result channel to communicate results to qbft layer
	resChan chan bool
}

func (s *signatureVerifier) verify() bool {
	return s.sig.FastAggregateVerify([]bls.PublicKey{*s.pk}, s.data)
}

// TODO maybe rewrite use VerifyMultiPubKey
func createQbftSignatureVerifier(signedMessage *specqbft.SignedMessage, domain spectypes.DomainType, sigType spectypes.SignatureType, committee []*spectypes.Operator, resChan chan bool) (*signatureVerifier, error) {
	// TODO: arguably this should be moved to spec crypto.go (copied forme there)
	validateThatSignersAreInCommittee := func(signers []spectypes.OperatorID, committee []*spectypes.Operator) ([]*bls.PublicKey, error) {
		// find operators
		pks := make([]*bls.PublicKey, 0)
		for _, id := range signers {
			found := false
			for _, n := range committee {
				if id == n.GetID() {
					pk := &bls.PublicKey{}
					if err := pk.Deserialize(n.GetPublicKey()); err != nil {
						return nil, errors.Wrap(err, "failed to deserialize public key")
					}
					pks = append(pks, pk)
					found = true
				}
			}
			if !found {
				return nil, errors.New("unknown signer")
			}
		}

		return pks, nil
	}
	// decode sig
	sign := &bls.Sign{}
	if err := sign.Deserialize(signedMessage.Signature); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize signature")
	}

	pks, err := validateThatSignersAreInCommittee(signedMessage.GetSigners(), committee)
	if err != nil {
		return nil, err
	}

	computedRoot, err := spectypes.ComputeSigningRoot(signedMessage, spectypes.ComputeSignatureDomain(domain, sigType))
	if err != nil {
		return nil, errors.Wrap(err, "could not compute signing root")
	}

	pk := pks[0]
	if len(pks) != 1 {
		// Aggregate public keys
		// TODO: maybe there is a better way to do this, but since committee is small, it should be fine
		// TODO: maybe we should use the same aggregation as in prysm
		// TODO: research PublicKey.Recover()
		for i := 1; i < len(pks); i++ {
			pk.Add(pks[i])
		}
	}
	return &signatureVerifier{computedRoot, sign, pk, resChan}, nil
}

// A routine that runs in the background to perform batch
// verifications of incoming messages from gossip.
func verifierRoutine(ctx context.Context, signatureChan chan *signatureVerifier, plogger zap.Logger) {
	verifierBatch := make([]*signatureVerifier, 0)
	ticker := time.NewTicker(signatureVerificationInterval)
	for {
		select {
		case <-ctx.Done():
			// Clean up currently utilised resources.
			ticker.Stop()
			for i := 0; i < len(verifierBatch); i++ {
				verifierBatch[i].resChan <- false
			}
			return
		case sig := <-signatureChan:
			verifierBatch = append(verifierBatch, sig)
			if len(verifierBatch) >= verifierLimit {
				verifyBatch(verifierBatch, plogger)
				verifierBatch = []*signatureVerifier{}
			}
		case <-ticker.C:
			if len(verifierBatch) > 0 {
				verifyBatch(verifierBatch, plogger)
				verifierBatch = []*signatureVerifier{}
			}
		}
	}
}

// TODO maybe add more tracing to errors here?
func validateWithBatchVerifier(ctx context.Context, signedMessage *specqbft.SignedMessage, domain spectypes.DomainType,
	sigType spectypes.SignatureType, committee []*spectypes.Operator, signatureChan chan *signatureVerifier, plog *zap.Logger) (pubsub.ValidationResult, error) {
	resChan := make(chan bool)
	verificationSet, err := createQbftSignatureVerifier(signedMessage, domain, sigType, committee, resChan)
	if err != nil {
		close(resChan)
		return pubsub.ValidationReject, errors.Wrap(err, "Could not create signature verifier")
	}
	signatureChan <- verificationSet

	verified := <-resChan
	close(resChan)
	// If verification fails we fallback to individual verification
	// of each signature set.

	// TODO create channel per peers and give up on single verification
	// This will be tricky though because messages from peers must be removed from network cache
	if !verified {
		plog.Error("batch verification failed")
		if verified := verificationSet.verify(); !verified {
			verErr := errors.Errorf("Single verification failed")
			return pubsub.ValidationReject, verErr
		}
	}
	return pubsub.ValidationAccept, nil
}

func verifyBatch(verifierBatch []*signatureVerifier, plogger zap.Logger) {
	if len(verifierBatch) == 0 {
		return
	}
	var sigs []bls.Sign
	var pubs []bls.PublicKey
	var concatenatedMsg []byte
	for _, batch := range verifierBatch {
		sigs = append(sigs, *batch.sig)
		pubs = append(pubs, *batch.pk)
		concatenatedMsg = append(concatenatedMsg, batch.data...)
	}

	verified := bls.MultiVerify(sigs, pubs, concatenatedMsg)
	for i := 0; i < len(verifierBatch); i++ {
		verifierBatch[i].resChan <- verified
	}
	plogger.Debug("batch verification", zap.Int("batch_size", len(verifierBatch)), zap.Bool("verified", verified))
}
