package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/templates"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// FlowConfig holds all information to work on flow with a given set of accounts in a wallet
type FlowConfig struct {
	Service *Account
	Wallet  *Wallet
	Host    string
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
func NewWalletDefault() (*Wallet, error) {
	return NewWallet("./wallet.json")
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

	contractPath := fmt.Sprintf("./contracts/%s.cdc", contractName)
	code, err := ioutil.ReadFile(contractPath)
	if err != nil {
		log.Fatalf("Could not read contract file from path=%s", contractPath)
	}
	f.apply(contractName, code)
}

// CreateAccount will create an account for running transactions without a contract
func (f *FlowConfig) CreateAccount(accountName string) {
	f.apply(accountName, nil)
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
	tx.SetGasLimit(uint64(100))

	err = tx.SignEnvelope(serviceAddress, serviceAccountKey.ID, serviceSigner)
	handle(err)

	err = c.SendTransaction(ctx, *tx)
	handle(err)

	blockTime := 10 * time.Second
	time.Sleep(blockTime)

	result, err := c.GetTransactionResult(ctx, tx.ID())
	handle(err)

	var address flow.Address

	if result.Status == flow.TransactionStatusSealed {
		for _, event := range result.Events {
			if event.Type == flow.EventAccountCreated {
				accountCreatedEvent := flow.AccountCreatedEvent(event)
				address = accountCreatedEvent.Address()
			}
		}
	} else {
		spew.Dump(result)
	}

	hexAddress := address.Hex()
	if hexAddress != user.Address {
		log.Fatalf("The address in the walletName=%s wallet=%s is not the same as the one generated=%s", contractName, user.Address, hexAddress)
	}
}

// NewFlowConfigLocalhost will create a flow configuration from local emulator and default files
func NewFlowConfigLocalhost() *FlowConfig {
	node := "127.0.0.1:3569"
	serviceAccount, err := NewFlowAccountDefault()
	if err != nil {
		log.Fatalf("run 'flow emulator init' errorMessage=%v", err)
	}

	wallet, err := NewWalletDefault()
	if err != nil {
		log.Fatal(err, "copy flow.json to wallet.json and specify new accounts with a given name")
	}

	return &FlowConfig{
		Service: serviceAccount,
		Wallet:  wallet,
		Host:    node,
	}

}
