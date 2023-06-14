package types

import (
	"encoding/hex"
	"sync"
	"time"

	specssv "github.com/bloxapp/ssv-spec/ssv"
	spectypes "github.com/bloxapp/ssv-spec/types"
	"github.com/herumi/bls-eth-go-binary/bls"
	"github.com/pkg/errors"
)

// VerifyByOperators verifies signature by the provided operators
// This is a copy of a function with the same name from the spec, except for it's use of
// DeserializeBLSPublicKey function and bounded.CGO
//
// TODO: rethink this function and consider moving/refactoring it.
func VerifyByOperators(s spectypes.Signature, data spectypes.MessageSignature, domain spectypes.DomainType, sigType spectypes.SignatureType, operators []*spectypes.Operator) error {
	// decode sig
	sign := &bls.Sign{}
	if err := sign.Deserialize(s); err != nil {
		return errors.Wrap(err, "failed to deserialize signature")
	}

	// find operators
	pks := make([]bls.PublicKey, 0)
	for _, id := range data.GetSigners() {
		found := false
		for _, n := range operators {
			if id == n.GetID() {
				pk, err := DeserializeBLSPublicKey(n.GetPublicKey())
				if err != nil {
					return errors.Wrap(err, "failed to deserialize public key")
				}

				pks = append(pks, pk)
				found = true
			}
		}
		if !found {
			return errors.New("unknown signer")
		}
	}

	// compute root
	computedRoot, err := spectypes.ComputeSigningRoot(data, spectypes.ComputeSignatureDomain(domain, sigType))
	if err != nil {
		return errors.Wrap(err, "could not compute signing root")
	}

	// verify
	if res := sign.FastAggregateVerify(pks, computedRoot[:]); !res {
		return errors.New("failed to verify signature")
	}
	return nil
}

func ReconstructSignature(ps *specssv.PartialSigContainer, root [32]byte, validatorPubKey []byte) ([]byte, error) {
	// Reconstruct signatures
	signature, err := spectypes.ReconstructSignatures(ps.Signatures[rootHex(root)])
	if err != nil {
		return nil, errors.Wrap(err, "failed to reconstruct signatures")
	}
	if err := VerifyReconstructedSignature(signature, validatorPubKey, root); err != nil {
		return nil, errors.Wrap(err, "failed to verify reconstruct signature")
	}
	return signature.Serialize(), nil
}

func VerifyReconstructedSignature(sig *bls.Sign, validatorPubKey []byte, root [32]byte) error {
	pk, err := DeserializeBLSPublicKey(validatorPubKey)
	if err != nil {
		return errors.Wrap(err, "could not deserialize validator pk")
	}

	// verify reconstructed sig
	if res := sig.VerifyByte(&pk, root[:]); !res {
		return errors.New("could not reconstruct a valid signature")
	}
	return nil
}

func rootHex(r [32]byte) string {
	return hex.EncodeToString(r[:])
}

type SignatureRequest struct {
	Signature bls.Sign
	PubKey    bls.PublicKey
	Message   []byte
	Result    chan bool
}

type BatchVerifier struct {
	workers   int
	batchSize int
	timeout   time.Duration

	pending []*SignatureRequest
	mu      sync.Mutex

	batches chan []*SignatureRequest
}

func (b *BatchVerifier) AggregateVerify(signature bls.Sign, pks []bls.PublicKey, message []byte) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	pk := pks[0]
	for i := 1; i < len(pks); i++ {
		pk.Add(&pks[i])
	}
	sr := &SignatureRequest{
		Signature: signature,
		PubKey:    pk,
		Message:   message,
		Result:    make(chan bool),
	}

	b.pending = append(b.pending, sr)
	if len(b.pending) == b.batchSize {
		b.batches <- b.pending
		b.pending = nil
	}

	return <-sr.Result
}

func (b *BatchVerifier) Run() {
	for i := 0; i < b.workers; i++ {
		go b.worker()
	}
}
func (b *BatchVerifier) worker() {
	for {
		select {
		case batch := <-b.batches:
			sigs := make([]bls.Sign, len(batch))
			pks := make([]bls.PublicKey, len(batch))
			msg := make([]byte, len(batch))
			msgIndex := 0
			for i, req := range batch {
				sigs[i] = req.Signature
				pks[i] = req.PubKey
				copy(msg[msgIndex:], req.Message)
				msgIndex += len(req.Message)
			}
			if bls.MultiVerify(sigs, pks, msg) {
				for _, req := range batch {
					req.Result <- true
				}
			} else {
				// Fallback to individual verification.
				for _, req := range batch {
					req.Result <- req.Signature.FastAggregateVerify(pks, msg)
				}
			}
		case <-time.After(b.timeout):
			b.mu.Lock()
			batch := b.pending
			b.pending = nil
			b.mu.Unlock()

			b.batches <- batch
		}
	}
}
