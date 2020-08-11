import FungibleToken from 0x179b6b1cb6755e31
import NonFungibleToken from  0x01cf0e2f2f715450

pub contract Test {
    pub var vault: @FungibleToken.Vault?
    pub var collection: @NonFungibleToken.Collection?

    init() {
        self.vault <- nil
        self.collection <- nil
    }
}