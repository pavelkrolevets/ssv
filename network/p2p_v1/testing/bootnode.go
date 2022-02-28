package testing

import (
	"context"
	"github.com/bloxapp/ssv/network/p2p_v1/discovery"
	"github.com/bloxapp/ssv/utils"
	"go.uber.org/zap"
)

// Options contains options to create the node
type Options struct {
	Logger     *zap.Logger
	PrivateKey string `yaml:"PrivateKey" env:"BOOTNODE_NETWORK_KEY" env-description:"Bootnode private key (default will generate new)"`
	ExternalIP string `yaml:"ExternalIP" env:"BOOTNODE_EXTERNAL_IP" env-description:"Override boot node's IP' "`
	Port       int    `yaml:"Port" env:"BOOTNODE_PORT" env-description:"Override boot node's port' "`
}

// Bootnode represents a bootnode used for tests
type Bootnode struct {
	ctx    context.Context
	cancel context.CancelFunc
	disc   discovery.Service

	ENR string
}

// NewBootnode creates a new bootnode
func NewBootnode(pctx context.Context, opts *Options) (*Bootnode, error) {
	ctx, cancel := context.WithCancel(pctx)
	disc, err := createDiscovery(ctx, opts)
	if err != nil {
		cancel()
		return nil, err
	}
	np := disc.(discovery.NodeProvider)
	enr := np.Self().Node().String()
	opts.Logger.Info("bootnode is ready", zap.String("ENR", enr))
	return &Bootnode{
		ctx:    ctx,
		cancel: cancel,
		disc:   disc,
		ENR:    enr,
	}, nil
}

// Close implements io.Closer
func (b *Bootnode) Close() error {
	b.cancel()
	return nil
}

func createDiscovery(ctx context.Context, opts *Options) (discovery.Service, error) {
	privKey, err := utils.ECDSAPrivateKey(opts.Logger.With(zap.String("who", "bootnode")), opts.PrivateKey)
	if err != nil {
		return nil, err
	}
	discOpts := discovery.Options{
		Logger: opts.Logger,
		DiscV5Opts: &discovery.DiscV5Options{
			IP:         opts.ExternalIP,
			BindIP:     "", // net.IPv4zero.String()
			Port:       opts.Port,
			TCPPort:    0,
			NetworkKey: privKey,
			Bootnodes:  nil,
			Logger:     opts.Logger,
		},
	}
	return discovery.NewService(ctx, discOpts)
}
