package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/templates"
	"google.golang.org/grpc"
)

// Account represents a Flow account
type Account struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
	SigAlgo    string `json:"sigAlgorithm"`
	HashAlgo   string `json:"hashAlgorithm"`
}

// Wallet represents the accounts in a Flow wallet
type Wallet struct {
	Accounts struct {
		Service          Account
		DemoToken        Account
		Rocks            Account
		VoteyAuction     Account
		NonFungibleToken Account
	}
}

// readFile reads the file contents from a provided file path
func readFile(path string) []byte {
	contents, err := ioutil.ReadFile(path)
	handle(err)
	return contents
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
}

// getWalletAccounts returns a Wallet struct with a list of accounts
func getWalletAccounts() Wallet {
	f, err := os.Open("./wallet.json")
	if err != nil {
		handle(err)
	}

	d := json.NewDecoder(f)

	var accountsInWallet Wallet

	err = d.Decode(&accountsInWallet)
	if err != nil {
		handle(err)
	}

	return accountsInWallet
}

func accountInfo(account Account) (crypto.PrivateKey, crypto.SignatureAlgorithm, crypto.HashAlgorithm) {

	sigAlgo := crypto.StringToSignatureAlgorithm(account.SigAlgo)
	hashAlgo := crypto.StringToHashAlgorithm(account.HashAlgo)
	privateKey, err := crypto.DecodePrivateKeyHex(sigAlgo, account.PrivateKey)
	handle(err)

	return privateKey, sigAlgo, hashAlgo
}

func createAccount(node string, user Account, service Account, code []byte) string {
	ctx := context.Background()

	// User Account
	privateKey, sigAlgo, hashAlgo := accountInfo(user)
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

	return address.Hex()
}

func main() {
	var wallet = getWalletAccounts()
	var node = "127.0.0.1:3569"
	var accounts = wallet.Accounts
	var account = createAccount(node, accounts.DemoToken, accounts.Service, nil)

	log.Println(account)
	// Create NFT standard account
	// Create DemoToken account
	// Create Rocks account
	// Create Auction account

	// Setup DemoToken Vault capabilities
	// Setup Rock Collection capabilities

	// Mint and deposit NFTs
	// Mint and deposit FTs

	// Run Check Setup script

	// Create auction collection

	// Run check listings script

	// Start the auction
}
