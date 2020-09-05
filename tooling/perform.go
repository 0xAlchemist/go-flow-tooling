package tooling

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/enescakir/emoji"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/templates"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// CreateAccount will create an account for running transactions without a contract
func (f *GoWithTheFlow) CreateAccount(accountName string) {
	err := f.apply(accountName, nil)
	if err != nil {
		log.Fatalf("%v error creating account %s %+v", emoji.PileOfPoo, accountName, err)
	}
	log.Printf("%v Account created: %s \n", emoji.Scroll, accountName)
}

// DeployContract will deploy a contract with the given name to an account with the same name from wallet.json
func (f *GoWithTheFlow) DeployContract(contractName string) {
	contractPath := fmt.Sprintf("./contracts/%s.cdc", contractName)
	//log.Printf("Deploying contract: %s at %s", contractName, contractPath)
	code, err := ioutil.ReadFile(contractPath)
	if err != nil {
		log.Fatalf("%v Could not read contract file from path=%s", emoji.PileOfPoo, contractPath)
	}
	err = f.apply(contractName, code)
	if err != nil {
		log.Fatalf("%v error creating account %s %+v", emoji.PileOfPoo, contractName, err)
	}
	log.Printf("%v Contract: %s successfully deployed\n", emoji.Scroll, contractName)
}

func (f *GoWithTheFlow) apply(contractName string, code []byte) error {

	user := f.Accounts[contractName]
	service := f.Service
	ctx := context.Background()

	c, err := client.New(f.Address, grpc.WithInsecure())
	if err != nil {
		return err
	}

	//The first time the service has not fetched the account or the signer
	if service.Account == nil {
		service.EnrichWithAccountSignerAndKey(c)
	}

	tx := templates.CreateAccount([]*flow.AccountKey{user.NewAccountKey()}, code, service.Address)

	// everything from here and almost down is EXACTLY the same as transaction
	blockHeader, err := c.GetLatestBlockHeader(ctx, true)
	if err != nil {
		return err
	}
	tx.SetReferenceBlockID(blockHeader.ID)

	tx.SetProposalKey(service.Address, service.Key.ID, service.Key.SequenceNumber)
	tx.SetPayer(service.Address)
	tx.SetGasLimit(f.Gas)
	err = tx.SignEnvelope(service.Address, service.Key.ID, service.Signer)
	if err != nil {
		return err
	}

	err = c.SendTransaction(ctx, *tx)
	if err != nil {
		return err
	}

	result, err := WaitForSeal(ctx, c, tx.ID())
	if err != nil {
		return err
	}

	if result.Error != nil {
		return result.Error
	}

	var address flow.Address
	for _, event := range result.Events {
		if event.Type == flow.EventAccountCreated {
			accountCreatedEvent := flow.AccountCreatedEvent(event)
			address = accountCreatedEvent.Address()
		}
	}

	//TODO is this the correct thing to do?
	if address != user.Address {
		return errors.Errorf("The address for account=%s does not match %s != %s", contractName, user.Address, address)
	}
	return nil
}

// SendTransaction executes a transaction file with a given name and signs it with the provided account
func (f *GoWithTheFlow) SendTransaction(filename string, signerAccountNames ...string) {

	err := f.sendTransactionRaw(filename, signerAccountNames, []cadence.Value{})
	if err != nil {
		log.Fatalf("%v error sending transaction %s %+v", emoji.PileOfPoo, filename, err)
	}
}

// SendTransactionWithArguments executes a transaction file with a given name and signs it with the provided account and send in the provided arguments
func (f *GoWithTheFlow) SendTransactionWithArguments(filename string, signer string, arguments ...cadence.Value) {

	err := f.sendTransactionRaw(filename, []string{signer}, arguments)
	if err != nil {
		log.Fatalf("%v error sending transaction %s %+v", emoji.PileOfPoo, filename, err)
	}
}

// SendTransactionWithMultipleSignersAndArguments executes a transaction file with a given name and signs it with the provided accounts and sends in the provided arguments
func (f *GoWithTheFlow) SendTransactionWithMultipleSignersAndArguments(filename string, signers []string, arguments ...cadence.Value) {
	err := f.sendTransactionRaw(filename, signers, arguments)
	if err != nil {
		log.Fatalf("%v error sending transaction %s %+v", emoji.PileOfPoo, filename, err)
	}
}

// SendTransaction executes a transaction file with a given name and signs it with the provided account
func (f *GoWithTheFlow) sendTransactionRaw(filename string, signers []string, arguments []cadence.Value) error {

	if len(signers) == 0 {
		log.Fatalf("Need atleast one signer to sign")
	}

	c, err := client.New(f.Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("%v Could create a new Flow client", emoji.PileOfPoo)
	}

	ctx := context.Background()

	// TODO: Support multiple signers
	signerAccountName := signers[0]
	signerAccount := f.Accounts[signerAccountName]
	if signerAccount.Signer == nil {
		signerAccount.EnrichWithAccountSignerAndKey(c)
	}
	txFilePath := fmt.Sprintf("./transactions/%s.cdc", filename)
	code, err := ioutil.ReadFile(txFilePath)
	if err != nil {
		log.Fatalf("%v Could not read transaction file from path=%s", emoji.PileOfPoo, txFilePath)
	}

	account := signerAccount.Account
	key := signerAccount.Key

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
	if err != nil {
		return err
	}

	tx.SetReferenceBlockID(blockHeader.ID)
	err = tx.SignEnvelope(account.Address, key.ID, signerAccount.Signer)
	if err != nil {
		return err
	}

	err = c.SendTransaction(ctx, *tx)
	if err != nil {
		return err
	}
	result, err := WaitForSeal(ctx, c, tx.ID())
	if err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}

	log.Printf("%v Transaction %s successfull applied with signer %s:%s\n", emoji.OkHand, txFilePath, signerAccountName, signerAccount.Address)
	return nil
}

// RunScript executes a read only script with a given filename on the blockchain
func (f *GoWithTheFlow) RunScript(filename string, arguments ...cadence.Value) {
	_ = f.RunScriptReturns(filename, arguments...)
}

// RunScriptReturns executes a read only script with a given filename on the blockchain
func (f *GoWithTheFlow) RunScriptReturns(filename string, arguments ...cadence.Value) cadence.Value {

	c, err := client.New(f.Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("%v Error creating flow client", emoji.PileOfPoo)
	}

	scriptFilePath := fmt.Sprintf("./scripts/%s.cdc", filename)
	code, err := ioutil.ReadFile(scriptFilePath)
	if err != nil {
		log.Fatalf("%v Could not read script file from path=%s", emoji.PileOfPoo, scriptFilePath)
	}

	log.Printf("Arguments %v\n", arguments)
	ctx := context.Background()
	result, err := c.ExecuteScriptAtLatestBlock(ctx, code, arguments)
	if err != nil {
		log.Fatalf("%v Error executing script: %s output %v", emoji.PileOfPoo, filename, err)
	}

	log.Printf("%v Script run from path %s result: %v\n", emoji.Star, scriptFilePath, result)
	return result
}

// WaitForSeal wait fot the process to seal
func WaitForSeal(ctx context.Context, c *client.Client, id flow.Identifier) (*flow.TransactionResult, error) {
	result, err := c.GetTransactionResult(ctx, id)
	if err != nil {
		return nil, err
	}

	//log.Printf("Waiting for transaction %s to be sealed...", id)

	for result.Status != flow.TransactionStatusSealed {
		time.Sleep(time.Second)
		//fmt.Print(".")
		result, err = c.GetTransactionResult(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	//log.Printf("Transaction %s sealed\n", id)
	return result, nil
}
