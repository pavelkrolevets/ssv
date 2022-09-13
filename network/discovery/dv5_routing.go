package discovery

import (
	"context"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strconv"
	"time"
)

// implementing discovery.Discovery

// Advertise advertises a service
// implementation of discovery.Advertiser
func (dvs *DiscV5Service) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (time.Duration, error) {
	opts := discovery.Options{}
	if err := opts.Apply(opt...); err != nil {
		return 0, errors.Wrap(err, "could not apply options")
	}
	if opts.Ttl == 0 {
		opts.Ttl = time.Hour
	}

	baseName := dvs.fork.GetTopicBaseName(ns)
	if baseName == ns {
		return 0, errors.Errorf("invalid topic: %s", ns)
	}
	subnet, err := strconv.Atoi(baseName)
	if subnet < 0 || err != nil {
		dvs.logger.Debug("not a subnet", zap.String("ns", ns))
		return opts.Ttl, nil
	}

	if err := dvs.RegisterSubnets(subnet); err != nil {
		return 0, err
	}

	return opts.Ttl, nil
}

// FindPeers discovers peers providing a service
// implementation of discovery.Discoverer
func (dvs *DiscV5Service) FindPeers(pctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	baseName := dvs.fork.GetTopicBaseName(ns)
	if baseName == ns {
		return nil, errors.Errorf("invalid topic: %s", ns)
	}
	subnet, err := strconv.Atoi(baseName)
	if subnet < 0 || err != nil {
		return nil, errors.Wrap(err, "not a subnet")
	}
	cn := make(chan peer.AddrInfo, 32)
	var opts discovery.Options
	if err := opts.Apply(opt...); err != nil {
		return nil, err
	}
	if opts.Ttl == 0 {
		opts.Ttl = time.Minute
	}
	go func() {
		ctx, cancel := context.WithTimeout(pctx, opts.Ttl)
		defer cancel()
		dvs.discover(ctx, func(e PeerEvent) {
			cn <- e.AddrInfo
		}, time.Millisecond, dvs.badNodeFilter, dvs.subnetFilter(uint64(subnet)))
	}()

	return cn, nil
}
