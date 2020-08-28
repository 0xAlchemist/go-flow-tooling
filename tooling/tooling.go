package tooling

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/enescakir/emoji"
	"github.com/mitchellh/go-homedir"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/templates"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// FlowConfig holds all information to work on flow with a given set of accounts in a wallet
type FlowConfig struct {
	Service    *Account
	Wallet     *Wallet
	Host       string
	Gas        uint64
	ParentPath string
}

// Account represents a Flow account
type Account struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
	SigAlgo    string `json:"sigAlgorithm"`
	HashAlgo   string `json:"hashAlgorithm"`
}

// Flow represents the contents of the flow.json file with an addition of host
type Flow struct {
	Accounts struct {
		Service Account
	}
}

// Wallet represents the accounts in a Flow wallet
type Wallet struct {
	Accounts map[string]Account
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// NewFlowAccountDefault will read the flow.json file from the default location
func NewFlowAccountDefault() (*Account, error) {
	return NewFlowAccount("./flow.json")
}

// NewFlowAccount will read the flow.json file and fetch the service account from there.
func NewFlowAccount(path string) (*Account, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "Could not read flow json file")
	}

	d := json.NewDecoder(f)

	var flowConfig Flow
	err = d.Decode(&flowConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Could not decode json info Flow")
	}

	return &flowConfig.Accounts.Service, nil
}

// NewWalletDefault will create a default wallet
func NewWalletDefault(path string) (*Wallet, error) {
	return NewWallet(fmt.Sprintf("%s/wallet.json", path))
}

// NewWallet will creat a wallet from the given path
func NewWallet(path string) (*Wallet, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "Could not read wallet json file")
	}

	d := json.NewDecoder(f)

	var wallet Wallet
	err = d.Decode(&wallet)
	if err != nil {
		return nil, errors.Wrap(err, "Could not decode json info Wallet")
	}

	return &wallet, nil
}

func accountInfo(account *Account) (crypto.PrivateKey, crypto.SignatureAlgorithm, crypto.HashAlgorithm) {

	sigAlgo := crypto.StringToSignatureAlgorithm(account.SigAlgo)
	hashAlgo := crypto.StringToHashAlgorithm(account.HashAlgo)
	privateKey, err := crypto.DecodePrivateKeyHex(sigAlgo, account.PrivateKey)
	handle(err)

	return privateKey, sigAlgo, hashAlgo
}

// DeployContract will deploy a contract with the given name to an account with the same name from wallet.json
func (f *FlowConfig) DeployContract(contractName string) {
	contractPath := fmt.Sprintf("%s/contracts/%s.cdc", f.ParentPath, contractName)
	//log.Printf("Deploying contract: %s at %s", contractName, contractPath)
	code, err := ioutil.ReadFile(contractPath)
	if err != nil {
		log.Fatalf("%v Could not read contract file from path=%s", emoji.PileOfPoo, contractPath)
	}
	f.apply(contractName, code)
	log.Printf("%v Contract: %s successfully deployed\n", emoji.Scroll, contractName)
}

// CreateAccount will create an account for running transactions without a contract
func (f *FlowConfig) CreateAccount(accountName string) {
	f.apply(accountName, nil)
}

//FindAddress finds an candence.Address value from a given key in your wallet
func (f *FlowConfig) FindAddress(key string) cadence.Address {
	address := f.Wallet.Accounts[key].Address

	byteAddress, err := hex.DecodeString(address)
	if err != nil {
		panic(err)
	}
	return cadence.BytesToAddress(byteAddress)
}

func (f *FlowConfig) apply(contractName string, code []byte) {

	node := f.Host
	user := f.Wallet.Accounts[contractName]
	service := f.Service
	ctx := context.Background()

	// User Account
	privateKey, sigAlgo, hashAlgo := accountInfo(&user)
	publicKey := privateKey.PublicKey()

	accountKey := flow.NewAccountKey().
		SetPublicKey(publicKey).
		SetSigAlgo(sigAlgo).
		SetHashAlgo(hashAlgo).
		SetWeight(flow.AccountKeyWeightThreshold)

	// Service Account
	servicePrivateKey, _, _ := accountInfo(service)
	serviceAddress := flow.HexToAddress(service.Address)

	c, err := client.New(node, grpc.WithInsecure())
	handle(err)

	serviceAccount, err := c.GetAccount(ctx, serviceAddress)
	handle(err)

	serviceAccountKey := serviceAccount.Keys[0]
	serviceSigner := crypto.NewInMemorySigner(servicePrivateKey, serviceAccountKey.HashAlgo)

	tx := templates.CreateAccount([]*flow.AccountKey{accountKey}, code, serviceAddress)
	tx.SetProposalKey(serviceAddress, serviceAccountKey.ID, serviceAccountKey.SequenceNumber)
	tx.SetPayer(serviceAddress)
	tx.SetGasLimit(f.Gas)

	blockHeader, err := c.GetLatestBlockHeader(ctx, true)
	handle(err)

	tx.SetReferenceBlockID(blockHeader.ID)

	err = tx.SignEnvelope(serviceAddress, serviceAccountKey.ID, serviceSigner)
	handle(err)

	err = c.SendTransaction(ctx, *tx)
	handle(err)

	result := WaitForSeal(ctx, c, tx.ID())
	handle(result.Error)

	var address flow.Address

	for _, event := range result.Events {
		if event.Type == flow.EventAccountCreated {
			accountCreatedEvent := flow.AccountCreatedEvent(event)
			address = accountCreatedEvent.Address()
		}
	}

	hexAddress := address.Hex()
	if hexAddress != user.Address {
		log.Fatalf("%v The address in the walletName=%s wallet=%s is not the same as the one generated=%s", emoji.PileOfPoo, contractName, user.Address, hexAddress)
	}
}

// SendTransactionWithArguments executes a transaction file with a given name and signs it with the provided account and send in the provided arguments
func (f *FlowConfig) SendTransactionWithArguments(filename string, signer string, arguments ...cadence.Value) {

	f.sendTransactionRaw(filename, []string{signer}, arguments)
}

// SendTransactionWithMultipleSignersAndArguments executes a transaction file with a given name and signs it with the provided accounts and sends in the provided arguments
func (f *FlowConfig) SendTransactionWithMultipleSignersAndArguments(filename string, signers []string, arguments ...cadence.Value) {

	f.sendTransactionRaw(filename, signers, arguments)
}

// SendTransaction executes a transaction file with a given name and signs it with the provided account
func (f *FlowConfig) SendTransaction(filename string, signerAccountNames ...string) {

	f.sendTransactionRaw(filename, signerAccountNames, []cadence.Value{})
}

//GetAccount gets the account
func (f *FlowConfig) GetAccount(name string) *flow.Account {
	node := f.Host

	c, err := client.New(node, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("%v Could create a new Flow client", emoji.PileOfPoo)
	}

	ctx := context.Background()
	signerAccount := f.Wallet.Accounts[name]
	if signerAccount.Address == "" {
		log.Fatalf("%v Invalid name %s", emoji.PileOfPoo, name)
	}
	account, err := c.GetAccount(ctx, flow.HexToAddress(signerAccount.Address))
	if err != nil {
		log.Fatalf("%v Could not get public account object for address: %s", emoji.PileOfPoo, signerAccount.Address)
	}
	return account
}

// SendTransaction executes a transaction file with a given name and signs it with the provided account
func (f *FlowConfig) sendTransactionRaw(filename string, signers []string, arguments []cadence.Value) {

	if len(signers) == 0 {
		log.Fatalf("Need atleast one signer to sign")
	}

	node := f.Host

	c, err := client.New(node, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("%v Could create a new Flow client", emoji.PileOfPoo)
	}

	ctx := context.Background()

	// TODO: Support multiple signers
	signerAccountName := signers[0]
	signerAccount := f.Wallet.Accounts[signerAccountName]
	if signerAccount.Address == "" {
		log.Fatalf("%v Invalid signerAccountName %s", emoji.PileOfPoo, signerAccountName)
	}
	account, err := c.GetAccount(ctx, flow.HexToAddress(signerAccount.Address))
	if err != nil {
		log.Fatalf("%v Could not get public account object for address: %s", emoji.PileOfPoo, signerAccount.Address)
	}

	key := account.Keys[0]

	txFilePath := fmt.Sprintf("%s/transactions/%s.cdc", f.ParentPath, filename)
	code, err := ioutil.ReadFile(txFilePath)
	if err != nil {
		log.Fatalf("%v Could not read transaction file from path=%s", emoji.PileOfPoo, txFilePath)
	}

	tx := flow.NewTransaction().
		SetScript(code).
		SetGasLimit(f.Gas).
		SetProposalKey(account.Address, key.ID, key.SequenceNumber).
		SetPayer(account.Address).
		AddAuthorizer(account.Address)

	for _, argument := range arguments {
		tx.AddArgument(argument)
	}

	blockHeader, err := c.GetLatestBlockHeader(ctx, true)
	handle(err)

	tx.SetReferenceBlockID(blockHeader.ID)

	//TODO: Refactor
	for _, signerName := range signers {
		envelopeAccount := f.Wallet.Accounts[signerAccountName]
		if envelopeAccount.Address == "" {
			log.Fatalf("%v Invalid signerAccountName %s", emoji.PileOfPoo, signerName)
		}

		account, err := c.GetAccount(ctx, flow.HexToAddress(envelopeAccount.Address))
		key := account.Keys[0]
		if err != nil {
			log.Fatalf("%v Could not get public account object for address: %s", emoji.PileOfPoo, account.Address)
		}

		privateKey, _, _ := accountInfo(&envelopeAccount)
		signer := crypto.NewInMemorySigner(privateKey, key.HashAlgo)
		err = tx.SignEnvelope(account.Address, key.ID, signer)
		if err != nil {
			log.Fatalf("%v Error signing the transaction. Transaction was not sent.", emoji.PileOfPoo)
		}
	}
	err = c.SendTransaction(ctx, *tx)
	if err != nil {
		log.Fatalf("%v Error sending the transaction. %v", emoji.PileOfPoo, err)
	}
	result := WaitForSeal(ctx, c, tx.ID())
	if result.Error != nil {
		log.Fatalf("%v There was an error completing transaction: %s error: %v", emoji.PileOfPoo, txFilePath, result.Error)
	}
	log.Printf("%v Transaction %s successfull applied with signer %s:%s\n", emoji.OkHand, txFilePath, signerAccountName, signerAccount.Address)
}

// RunScript executes a read only script with a given filename on the blockchain
func (f *FlowConfig) RunScript(filename string, arguments ...cadence.Value) {
	_ = f.RunScriptReturns(filename, arguments...)
}

// RunScriptReturns executes a read only script with a given filename on the blockchain
func (f *FlowConfig) RunScriptReturns(filename string, arguments ...cadence.Value) cadence.Value {
	node := f.Host

	c, err := client.New(node, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("%v Error creating flow client", emoji.PileOfPoo)
	}

	scriptFilePath := fmt.Sprintf("%s/scripts/%s.cdc", f.ParentPath, filename)
	code, err := ioutil.ReadFile(scriptFilePath)
	if err != nil {
		log.Fatalf("%v Could not read script file from path=%s", emoji.PileOfPoo, scriptFilePath)
	}

	log.Printf("Arguments %v\n", arguments)
	log.Println(code)
	ctx := context.Background()
	result, err := c.ExecuteScriptAtLatestBlock(ctx, code, arguments)
	if err != nil {
		log.Fatalf("%v Error executing script: %s output %v", emoji.PileOfPoo, filename, err)
	}

	log.Printf("%v Script run from path %s result: %v\n", emoji.Star, scriptFilePath, result)
	return result
}

// WaitForSeal wait fot the process to seal
func WaitForSeal(ctx context.Context, c *client.Client, id flow.Identifier) *flow.TransactionResult {
	result, err := c.GetTransactionResult(ctx, id)
	handle(err)

	//log.Printf("Waiting for transaction %s to be sealed...", id)

	for result.Status != flow.TransactionStatusSealed {
		time.Sleep(time.Second)
		//fmt.Print(".")
		result, err = c.GetTransactionResult(ctx, id)
		handle(err)
	}

	//log.Printf("Transaction %s sealed\n", id)

	return result
}

// NewFlowConfigLocalhostWithGas will create a flow configuration from local emulator and default files
func NewFlowConfigLocalhostWithGas(gas int) *FlowConfig {
	host := "127.0.0.1:3569"
	serviceAccount, err := NewFlowAccount("./flow.json")
	if err != nil {
		log.Fatalf("%v run 'flow emulator init' errorMessage=%v", emoji.PileOfPoo, err)
	}

	return createFlowConfig(serviceAccount, host, uint64(gas), ".")

}

// NewFlowConfigLocalhostWithParentPath will create a flow configuration from local emulator and default files from a subdir
func NewFlowConfigLocalhostWithParentPath(path string) *FlowConfig {
	host := "127.0.0.1:3569"
	serviceAccount, err := NewFlowAccount(fmt.Sprintf("%s/flow.json", path))
	if err != nil {
		log.Fatalf("%v run 'flow emulator init' errorMessage=%v", emoji.PileOfPoo, err)
	}

	return createFlowConfig(serviceAccount, host, uint64(9999), path)

}

// NewFlowConfigLocalhost will create a flow configuration from local emulator and default files
func NewFlowConfigLocalhost() *FlowConfig {
	host := "127.0.0.1:3569"
	serviceAccount, err := NewFlowAccount("./flow.json")
	if err != nil {
		log.Fatalf("%v run 'flow emulator init' errorMessage=%v", emoji.PileOfPoo, err)
	}

	return createFlowConfig(serviceAccount, host, uint64(9999), ".")

}

func createFlowConfig(serviceAccount *Account, node string, gas uint64, path string) *FlowConfig {
	wallet, err := NewWalletDefault(path)
	if err != nil {
		log.Fatalf("%v copy flow.json to wallet.json and specify new accounts with a given name %v", emoji.PileOfPoo, err)
	}

	return &FlowConfig{
		Service:    serviceAccount,
		Wallet:     wallet,
		Host:       node,
		Gas:        gas,
		ParentPath: path,
	}
}

// NewFlowConfigDevNet setup devnot like in https://www.notion.so/Accessing-Flow-Devnet-ad35623797de48c08d8b88102ea38131
func NewFlowConfigDevNet() *FlowConfig {
	host := "access-001.devnet12.nodes.onflow.org:9000"

	flowConfigFile, err := homedir.Expand("~/.flow-dev.json")
	if err != nil {
		log.Fatalf("%v error %v", emoji.PileOfPoo, err)
	}
	serviceAccount, err := NewFlowAccount(flowConfigFile)
	if err != nil {
		log.Fatalf("%v Create a file in the location %s with your dev net credentials error:%v", emoji.PileOfPoo, flowConfigFile, err)
	}
	return createFlowConfig(serviceAccount, host, uint64(9999), ".")
}
