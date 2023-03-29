<h1 align="center">SSV: distributed staking ecosystem for Ethereum</h1>

<p align="center">
A decentralized, trust-minimized and scalable protocol that empowers Ethereum validators<br/>
to break beyond the constraints of a single node and flourish across a vibrant network of interconnected parties.
</p>

<center>

[![API Reference](
https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://pkg.go.dev/github.com/ethereum/eth2-ssv?tab=doc)
![Github Actions](https://github.com/ethereum/eth2-ssv/actions/workflows/full-test.yml/badge.svg?branch=stage)
![Github Actions](https://github.com/ethereum/eth2-ssv/actions/workflows/lint.yml/badge.svg?branch=stage)
![Test Coverage](./docs/resources/cov-badge.svg)
[![Discord](https://img.shields.io/badge/discord-join%20chat-blue.svg)](https://discord.gg/eDXSP9R)

[comment]: <> ([![Go Report Card]&#40;https://goreportcard.com/badge/github.com/ethereum/eth2-ssv&#41;]&#40;https://goreportcard.com/report/github.com/ethereum/eth2-ssv&#41;)

[comment]: <> ([![Travis]&#40;https://travis-ci.com/ethereum/eth2-ssv.svg?branch=stage&#41;]&#40;https://travis-ci.com/ethereum/eth2-ssv&#41;)
</center>

<center>
<img src="./ssv-network-full-logo.png" width="200">

[Documentation](https://docs.ssv.network/learn/introduction)
</center>

## Repositories

- 🔑 [ssv-keys](https://github.com/bloxapp/ssv-keys): CLI to break apart a validator's private key into a set of shares to be held by different operators.
- 📜 [ssv-network](https://github.com/bloxapp/ssv-network/): smart contracts which onboard validators, operators and their shares.
- ⚙️ [ssv](https://github.com/bloxapp/ssv) (this repo): p2p node which operators run to participate in the network and coordinate to perform the duties for their validators. Governed by [QBFT](https://entethalliance.github.io/client-spec/qbft_spec.html).

## Solo vs. Centralized vs. SSV

| Scenario | Solo | Centralized | SSV |
| --- | --- | --- | --- |
| Hardware/network failure | ❌ Single machine | ⚠️ Opaque | ✅ 3+ machines |
| Liveness failure in 1 of the clients | ❌ Single client | ⚠️ Opaque | ✅ 3+ clients |
| Slashing protection failure | ⚠️ Single signer | ⚠️ Opaque | ✅ 3+ signers |
| Setup complexity | ★ Hard | ★★★ Easy | ★★ Reasonable |

## Getting Started

Choose your path!

### ⚔ Become an Operator
Operators 

Join the network by [installing an SSV node](https://docs.ssv.network/run-a-node/operator-node/installation) and [registering as an operator](https://docs.ssv.network/run-a-node/operator-node/joining-the-network).

### 🏹 Become a Liquidator

Liquidators run a special node that scouts for out-of-balance validators and liquidates them. [Get started here](https://docs.ssv.network/run-a-node/liquidator-node).

### 🛠️ Build on SSV
Anyone can register a distributed validator via the smart contracts. [Get started here](https://docs.ssv.network/developers/get-started).

## Links

### Articles
- [An Introduction to Secret Shared Validators (SSV) for Ethereum 2.0](https://medium.com/bloxstaking/an-introduction-to-secret-shared-validators-ssv-for-ethereum-2-0-faf49efcabee)

### Papers
* [iBFT Paper](https://arxiv.org/pdf/2002.03613.pdf)
    * [Fast sync for current instance](./ibft/sync/speedup/README.md)
* [iBFT annotated paper (By Blox)](./ibft/IBFT.md)
* [EIP650](https://github.com/ethereum/EIPs/issues/650)
* [Security proof for n-t honest parties](https://notes.ethereum.org/DYU-NrRBTxS3X0fu_MidnA)
* [MEV Research - Block proposer/builder separation in SSV](https://hackmd.io/DHt98PC_S_60NbnW4Wgssg)


## Getting Started

The following documents contain instructions and information on how to get started:
* [Operator Node Installation](https://docs.ssv.network/run-a-node/operator-node/installation)
* [Developers' Guide](./docs/DEV_GUIDE.md)

## Contribution

Thank you for considering a contribution to the source code.

In order to contribute to eth2-ssv, please fork, add your code, commit and send a pull request
for the maintainers to review and merge into the main code base.\
If you wish to submit more complex changes though, please check up with the core devs first on [our discord](https://discord.gg/eDXSP9R)
to ensure those changes are in line with the general philosophy of the project and/or get
some early feedback which can make both your efforts much lighter as well as our review
and merge procedures quick and simple.

Please see the [Developers' Guide](./docs/DEV_GUIDE.md)
for more details on configuring your environment, managing project dependencies, and
testing procedures.

## License

The eth2-ssv library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `LICENSE` file.

The eth2-ssv binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `LICENSE` file.
