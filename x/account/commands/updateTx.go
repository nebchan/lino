package commands

import (
	"fmt"

	"github.com/lino-network/lino/client"

	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	acc "github.com/lino-network/lino/x/account"
)

// UpdateTxCmd will create a transfer tx and sign it with the given key
func UpdateTxCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-account",
		Short: "Update account meta data",
		RunE:  sendUpdateTx(cdc),
	}
	cmd.Flags().String(client.FlagUser, "", "update user")
	cmd.Flags().String(client.FlagJsonMeta, "", "user json meta")
	return cmd
}

// send transfer transaction to the blockchain
func sendUpdateTx(cdc *wire.Codec) client.CommandTxCallback {
	return func(cmd *cobra.Command, args []string) error {
		ctx := client.NewCoreContextFromViper()
		username := viper.GetString(client.FlagUser)
		jsonmeta := viper.GetString(client.FlagJsonMeta)
		msg := acc.NewUpdateAccountMsg(
			username, jsonmeta)

		// build and sign the transaction, then broadcast to Tendermint
		res, err := ctx.SignBuildBroadcast([]sdk.Msg{msg}, cdc)

		if err != nil {
			return err
		}

		fmt.Printf("Committed at block %d. Hash: %s\n", res.Height, res.Hash.String())
		return nil
	}
}
