package main

import (
	"github.com/0xAlchemist/go-flow-tooling/tooling"
	"github.com/onflow/cadence"
)

func main() {

	flow := tooling.NewFlowConfigLocalhost()

	// Deploy Contracts
	flow.DeployContract("nft")
	flow.DeployContract("ft")

	// Send Transaction
	flow.SendTransaction("create_nft_collection", "ft")

	// Send Transaction supports multiple singers, they will all be AuthAccounts
	// TODO This does not work?
	//flow.SendTransaction("signWithMultipleAccounts", "ft", "nft")

	flow.SendTransactionWithArguments("arguments", "ft", cadence.String("argument1"))

	//create an argument that is a cadence.Address from the wallet.json file
	flow.SendTransactionWithArguments("argumentsWithAccount", "ft", flow.FindAddress("nft"))

	// Run Script
	flow.RunScript("test", cadence.String("argument1"))
}
