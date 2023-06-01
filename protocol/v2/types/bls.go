package types

import (
	"github.com/bloxapp/ssv/logging/fields"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/herumi/bls-eth-go-binary/bls"
	"go.uber.org/zap"
)

var blsPublicKeyCache *lru.Cache[string, bls.PublicKey]

func init() {
	var err error
	blsPublicKeyCache, err = lru.New[string, bls.PublicKey](10_000)
	if err != nil {
		panic(err)
	}
}

// DeserializeBLSPublicKey deserializes a bls.PublicKey from bytes,
// caching the result to avoid repeated deserialization.
func DeserializeBLSPublicKey(b []byte) (bls.PublicKey, error) {
	pkStr := string(b)
	if pk, ok := blsPublicKeyCache.Get(pkStr); ok {
		return pk, nil
	}

	zap.L().Debug("deserializing bls public key", fields.PubKey(b), zap.Int("cache_size", blsPublicKeyCache.Len()))
	pk := bls.PublicKey{}
	if err := pk.Deserialize(b); err != nil {
		return bls.PublicKey{}, err
	}
	blsPublicKeyCache.Add(pkStr, pk)

	return pk, nil
}
