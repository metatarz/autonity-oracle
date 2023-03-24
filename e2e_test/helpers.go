package test

import (
	"autonity-oracle/config"
	"autonity-oracle/helpers"
	oracleserver "autonity-oracle/oracle_server"
	"autonity-oracle/types"
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"io/fs"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	numberOfValidators        = 4
	numberOfPortsForBindNodes = 3
	numberOfKeys              = 10
	defaultPlugDir            = "../build/bin/plugins"
	defaultHost               = "127.0.0.1"
	defaultDataDirRoot        = "../test_data/autonity_l1_net_config/nodes/"
	defaultBondedStake        = new(big.Int).SetUint64(1000)
)

type AutonityContractGenesis struct {
	Bytecode         string         `json:"bytecode,omitempty" toml:",omitempty"`
	ABI              string         `json:"abi,omitempty" toml:",omitempty"`
	MinBaseFee       uint64         `json:"minBaseFee"`
	EpochPeriod      uint64         `json:"epochPeriod"`
	UnbondingPeriod  uint64         `json:"unbondingPeriod"`
	BlockPeriod      uint64         `json:"blockPeriod"`
	MaxCommitteeSize uint64         `json:"maxCommitteeSize"`
	Operator         common.Address `json:"operator"`
	Treasury         common.Address `json:"treasury"`
	TreasuryFee      uint64         `json:"treasuryFee"`
	DelegationRate   uint64         `json:"delegationRate"`
	Validators       []*Validator   `json:"validators"`
}

type ChainConfig struct {
	ChainID  *big.Int                 `json:"chainId"` // chainId identifies the current chain and is used for replay protection
	Autonity *AutonityContractGenesis `json:"autonity"`
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Balance *big.Int `json:"balance" gencodec:"required"`
}

type GenesisAlloc map[common.Address]GenesisAccount

type Genesis struct {
	Config     *ChainConfig   `json:"config"`
	Nonce      uint64         `json:"nonce"`
	Timestamp  uint64         `json:"timestamp"`
	GasLimit   uint64         `json:"gasLimit"   gencodec:"required"`
	Difficulty *big.Int       `json:"difficulty" gencodec:"required"`
	Mixhash    common.Hash    `json:"mixHash"`
	Coinbase   common.Address `json:"coinbase"`
	Alloc      GenesisAlloc   `json:"alloc"      gencodec:"required"`

	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`
	BaseFee    *big.Int    `json:"baseFee"`
}

type Validator struct {
	Treasury    common.Address `json:"treasury"`
	Enode       string         `json:"enode"`
	Voter       common.Address `json:"voter"`
	BondedStake *big.Int       `json:"bondedStake"`
}

type Oracle struct {
	Key       *Key
	PluginDir string
	Host      string
	HTTPPort  int
	ProcessID int
}

// todo: start the oracle client process.
func (o *Oracle) Start() {
	o.ProcessID = 1
}

// todo: stop the oracle client process.
func (o *Oracle) Stop() {
	o.ProcessID = -1
}

type Key struct {
	KeyFile  string
	Password string
	Key      *keystore.Key
}

type Node struct {
	DataDir      string
	NodeKey      *Key
	Host         string
	P2PPort      int
	WSPort       int
	ProcessID    int
	OracleClient *Oracle
	Validator    *Validator
}

func (n *Node) genConfigs() error {
	// todo: gen configs for autonity client,

	// todo: gen configs for the corresponding oracle client.

	return nil
}

// todo: start the autontiy client process
func (n *Node) Start() {
	n.ProcessID = 2
}

// todo: stop the autonity client process
func (n *Node) Stop() {
	n.ProcessID = -1
}

type Network struct {
	OperatorKey *Key
	TreasuryKey *Key
	Nodes       []*Node
}

// todo: generate the genesis file for the network.
func (net *Network) genGenesisFile() error {
	return nil
}

// prepare configurations for autonity l1 node and oracle client node
func (net *Network) genConfigs() error {
	if err := net.genGenesisFile(); err != nil {
		return err
	}

	for _, n := range net.Nodes {
		if err := n.genConfigs(); err != nil {
			return err
		}
	}
	return nil
}

// todo: start the network
func (net *Network) Start() error {
	return nil
}

// todo: stop the network
func (net *Network) Stop() {

}

// create with a four-nodes autonity l1 network for the integration of oracle service, with each of validator bind with
// an oracle node.
func createNetwork(keystore string, password string) (*Network, error) {
	keys, err := loadKeys(keystore, password)
	if err != nil {
		return nil, err
	}

	if len(keys) != numberOfKeys {
		panic("keystore does not contains enough key for testbed")
	}

	var network = &Network{
		OperatorKey: keys[0],
		TreasuryKey: keys[1],
	}

	freePorts, err := getFreePost(numberOfValidators * numberOfPortsForBindNodes)
	if err != nil {
		return nil, err
	}

	//plan the network with number of validators, allocate configs for L1 node and the corresponding oracle client.
	network, err = prepareResource(network, keys[2:], freePorts, numberOfValidators)
	if err != nil {
		return nil, err
	}

	err = network.genConfigs()
	if err != nil {
		return nil, err
	}

	err = network.Start()
	if err != nil {
		return nil, err
	}

	return network, nil
}

func prepareResource(network *Network, freeKeys []*Key, freePorts []int, nodes int) (*Network, error) {

	for i := 0; i < nodes; i++ {
		// allocate a key and a port for oracle client,
		var oracle = &Oracle{
			Key:       freeKeys[i*2],
			PluginDir: defaultPlugDir,
			Host:      defaultHost,
			HTTPPort:  freePorts[i*3],
			ProcessID: -1,
		}

		// allocate a key and 2 ports for validator client,
		var aut = &Node{
			DataDir:      fmt.Sprintf("%s/node_%d/data", defaultDataDirRoot, i),
			NodeKey:      freeKeys[i*2+1],
			Host:         defaultHost,
			P2PPort:      freePorts[i*3+1],
			WSPort:       freePorts[i*3+2],
			OracleClient: oracle,
		}

		var validator = &Validator{
			Treasury:    aut.NodeKey.Key.Address,
			Enode:       genEnode(&aut.NodeKey.Key.PrivateKey.PublicKey, aut.Host, aut.P2PPort),
			Voter:       crypto.PubkeyToAddress(oracle.Key.Key.PrivateKey.PublicKey),
			BondedStake: defaultBondedStake,
		}

		aut.OracleClient = oracle
		aut.Validator = validator

		network.Nodes = append(network.Nodes, aut)
	}
	return network, nil
}

func makeGenesisConfig(srcTemplate string, dstFile string, vals []*Validator, treasury common.Address, operator common.Address) error {
	file, err := os.Open(srcTemplate)
	if err != nil {
		return err
	}

	defer file.Close()

	genesis := new(Genesis)
	if err = json.NewDecoder(file).Decode(genesis); err != nil {
		return err
	}
	genesis.Config.Autonity.Operator = operator
	genesis.Config.Autonity.Treasury = treasury
	genesis.Config.Autonity.Validators = append(genesis.Config.Autonity.Validators, vals...)

	jsonData, err := json.MarshalIndent(genesis, "", " ")
	if err != nil {
		return err
	}

	if err = os.WriteFile(dstFile, jsonData, 0644); err != nil {
		return err
	}

	return nil
}

// load all keys from keystore with the corresponding password.
func loadKeys(kStore string, password string) ([]*Key, error) {
	files, err := listDir(kStore)
	if err != nil {
		return nil, err
	}

	var keys []*Key
	for _, f := range files {
		keyFile := fmt.Sprintf(fmt.Sprintf("%s/%s", kStore, f))
		keyJson, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}

		key, err := keystore.DecryptKey(keyJson, password)
		if err != nil {
			return nil, err
		}

		var k = &Key{Key: key, KeyFile: keyFile, Password: password}
		keys = append(keys, k)
	}

	return keys, nil
}

// generate enode url
func genEnode(key *ecdsa.PublicKey, host string, port int) string {
	pub := fmt.Sprintf("%x", crypto.FromECDSAPub(key)[1:])
	return fmt.Sprintf("enode://%s@%s:%d", pub, host, port)
}

// get free ports from current system
func getFreePost(numOfPort int) ([]int, error) {
	return freeport.GetFreePorts(numOfPort)
}

func testReplacePlugin(t *testing.T, port int, pluginDir string) {
	// get the plugins before replacement.
	reqMsg := &types.JSONRPCMessage{Method: "list_plugins"}
	respMsg, err := httpPost(t, reqMsg, port)
	require.NoError(t, err)
	var pluginsAtStart types.PluginByName
	err = json.Unmarshal(respMsg.Result, &pluginsAtStart)
	require.NoError(t, err)

	// do the replacement of plugins.
	err = replacePlugins(pluginDir)
	require.NoError(t, err)
	// wait for replaced plugins to be loaded.
	time.Sleep(10 * time.Second)

	respMsg, err = httpPost(t, reqMsg, port)
	require.NoError(t, err)
	var pluginsAfterReplace types.PluginByName
	err = json.Unmarshal(respMsg.Result, &pluginsAfterReplace)
	require.NoError(t, err)

	for k, p := range pluginsAfterReplace {
		require.Equal(t, p.Name, pluginsAtStart[k].Name)
		require.Equal(t, true, p.StartAt.After(pluginsAtStart[k].StartAt))
	}
}

func testAddPlugin(t *testing.T, port int, pluginDir string) {
	clonerPrefix := "clone"
	clonedPlugins, err := clonePlugins(pluginDir, clonerPrefix, pluginDir)
	defer func() {
		for _, f := range clonedPlugins {
			os.Remove(f) // nolint
		}
	}()

	require.NoError(t, err)
	require.Equal(t, true, len(clonedPlugins) > 0)
	// wait for cloned plugins to be loaded.
	time.Sleep(10 * time.Second)
	testListPlugins(t, port, pluginDir)
}

func testGetVersion(t *testing.T, port int) {
	var reqMsg = &types.JSONRPCMessage{
		Method: "get_version",
	}

	respMsg, err := httpPost(t, reqMsg, port)
	require.NoError(t, err)
	type Version struct {
		Version string
	}
	var V Version
	err = json.Unmarshal(respMsg.Result, &V)
	require.NoError(t, err)
	require.Equal(t, oracleserver.Version, V.Version)
}

func testGetPrices(t *testing.T, port int) {
	reqMsg := &types.JSONRPCMessage{
		Method: "get_prices",
	}

	respMsg, err := httpPost(t, reqMsg, port)
	require.NoError(t, err)
	type PriceAndSymbol struct {
		Prices  types.PriceBySymbol
		Symbols []string
	}
	var ps PriceAndSymbol
	err = json.Unmarshal(respMsg.Result, &ps)
	require.NoError(t, err)
	require.Equal(t, strings.Split(config.DefaultSymbols, ","), ps.Symbols)
	for _, s := range ps.Symbols {
		require.Equal(t, s, ps.Prices[s].Symbol)
		require.Equal(t, true, ps.Prices[s].Price.Equal(helpers.ResolveSimulatedPrice(s)))
	}
}

func testListPlugins(t *testing.T, port int, pluginDir string) {
	reqMsg := &types.JSONRPCMessage{Method: "list_plugins"}

	respMsg, err := httpPost(t, reqMsg, port)
	require.NoError(t, err)
	var plugins types.PluginByName
	err = json.Unmarshal(respMsg.Result, &plugins)
	require.NoError(t, err)
	files, err := listDir(pluginDir)
	require.NoError(t, err)
	require.Equal(t, len(files), len(plugins))
}

func httpPost(t *testing.T, reqMsg *types.JSONRPCMessage, port int) (*types.JSONRPCMessage, error) {
	jsonData, err := json.Marshal(reqMsg)
	require.NoError(t, err)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d", port), "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	var respMsg types.JSONRPCMessage
	err = json.NewDecoder(resp.Body).Decode(&respMsg)
	require.NoError(t, err)
	return &respMsg, nil
}

func replacePlugins(pluginDir string) error {
	rawPlugins, err := listDir(pluginDir)
	if err != nil {
		return err
	}

	clonePrefix := "clone"
	clonedPlugins, err := clonePlugins(pluginDir, clonePrefix, fmt.Sprintf("%s/..", pluginDir))
	if err != nil {
		return err
	}

	for _, file := range clonedPlugins {
		for _, info := range rawPlugins {
			if strings.Contains(file, info.Name()) {
				err := os.Rename(file, fmt.Sprintf("%s/%s", pluginDir, info.Name()))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func clonePlugins(pluginDIR string, clonePrefix string, destDir string) ([]string, error) {

	var clonedPlugins []string
	files, err := listDir(pluginDIR)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// read srcFile
		srcContent, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", pluginDIR, file.Name()))
		if err != nil {
			return clonedPlugins, err
		}

		// create dstFile and copy the content
		newPlugin := fmt.Sprintf("%s/%s%s", destDir, clonePrefix, file.Name())
		err = ioutil.WriteFile(newPlugin, srcContent, file.Mode())
		if err != nil {
			return clonedPlugins, err
		}
		clonedPlugins = append(clonedPlugins, newPlugin)
	}
	return clonedPlugins, nil
}

func listDir(pluginDIR string) ([]fs.FileInfo, error) {
	var plugins []fs.FileInfo

	files, err := ioutil.ReadDir(pluginDIR)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		plugins = append(plugins, file)
	}
	return plugins, nil
}
