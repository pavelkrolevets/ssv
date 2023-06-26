package eth1client

import (
	"context"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloxapp/ssv/eth1_refactor/eth1client/backends"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	contractAddr = ethcommon.HexToAddress("0xdB7d6AB1f17c6b31909aE466702703dAEf9269Cf")
)

// the following is based on this contract:
//
/*
Example contract to test event emission:

	pragma solidity >=0.7.0 <0.9.0;
	contract Callable {
		event Called();
		function Call() public { emit Called(); }
	}
*/
const callableAbi = "[{\"anonymous\":false,\"inputs\":[],\"name\":\"Called\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"Call\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

const callableBin = "6080604052348015600f57600080fd5b5060998061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c806334e2292114602d575b600080fd5b60336035565b005b7f81fab7a4a0aa961db47eefc81f143a5220e8c8495260dd65b1356f1d19d3c7b860405160405180910390a156fea2646970667358221220029436d24f3ac598ceca41d4d712e13ced6d70727f4cdc580667de66d2f51d8b64736f6c63430008010033"

func TestEth1Client(t *testing.T) {
	ctx := context.Background()

	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := backends.NewSimulatedBackend(core.GenesisAlloc{
		testAddr: {Balance: big.NewInt(10000000000000000)},
	}, 10000000,)
	defer sim.Close()
	
	parsed, err := abi.JSON(strings.NewReader(callableAbi))
	if err != nil {
		t.Errorf("could not get code at test addr: %v", err)
	}
	auth, _ := bind.NewKeyedTransactorWithChainID(testKey, big.NewInt(1337))
	_, _, contract, err := bind.DeployContract(auth, parsed, ethcommon.FromHex(callableBin), sim)
	if err != nil {
		t.Errorf("deploying contract: %v", err)
	}
	sim.Commit()
	// 2.
	logs, sub, err := contract.WatchLogs(nil, "Called")
	if err != nil {
		t.Errorf("watching logs: %v", err)
	}
	defer sub.Unsubscribe()

	tx, err := contract.Transact(auth, "Call")
	if err != nil {
		t.Errorf("transacting: %v", err)
	}
	sim.Commit()

	log := <-logs
	t.Log("Log topics", log.Topics)

	if log.TxHash != tx.Hash() {
		t.Fatal("wrong event tx hash")
	}
	if log.Removed {
		t.Fatal("Event should be included")
	}

	rpcServer, _ := sim.Node.RPCHandler()
	httpsrv := httptest.NewServer(rpcServer.WebsocketHandler([]string{"*"}))
	defer rpcServer.Stop()
	defer httpsrv.Close()

	cl, err := sim.Node.Attach()
	ec := ethclient.NewClient(cl)
	
	query := ethereum.FilterQuery{
		FromBlock: nil,
		ToBlock:   nil,
	}
	sim.Commit()
	l, err := ec.FilterLogs(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("LOGS", l)

	addr := "ws:" + strings.TrimPrefix(httpsrv.URL, "http:")
	logger := zaptest.NewLogger(t)

	client := New(addr, contractAddr, WithLogger(logger))

	require.NoError(t, client.Connect(ctx))

	ready, err := client.IsReady(ctx)
	require.NoError(t, err)
	require.True(t, ready)

	tests := map[string]struct {
		test func(t *testing.T)
	}{
		"Logs": {
			func(t *testing.T) { testFetchHistoricalLogs(t, ctx, client) },
		},
	}

	t.Parallel()
	for name, tt := range tests {
		t.Run(name, tt.test)
	}

	require.NoError(t, client.Close())
}

func testFetchHistoricalLogs(t *testing.T, ctx context.Context, client *Eth1Client) {
	logs, _, err := client.FetchHistoricalLogs(ctx, 0)
	if err != nil {
		t.Fatalf("FetchHistoricalLogs error = %q", err)
	}
	require.NoError(t, err)
	require.NotEmpty(t, logs)
}
