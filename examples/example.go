package main

import (
	"log"

	"github.com/versus-flow/go-flow-tooling/tooling"
	"github.com/onflow/cadence"
)

func main() {

	flow := tooling.NewFlowConfigLocalhost()

	// Deploy Contracts
	flow.DeployContract("nft")
	flow.DeployContract("ft")

	// Send Transaction
	flow.SendTransaction("create_nft_collection", "ft")

	//send transaction with a string argument
	flow.SendTransactionWithArguments("arguments", "ft", cadence.String("argument1"))

	//create an argument that is a cadence.Address from the wallet.json file
	flow.SendTransactionWithArguments("argumentsWithAccount", "ft", flow.FindAddress("nft"))

	// Run Script
	flow.RunScript("test", cadence.String("argument1"))

	result := flow.RunScriptReturns("test", flow.FindAddress("nft"))
	log.Printf("Script returned %s", result)

}
