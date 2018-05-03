package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/lino-network/lino/client"
	acc "github.com/lino-network/lino/tx/account"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
)

// FollowTxCmd will create a follow tx and sign it with the given key
func FollowTxCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "follow",
		Short: "Create and sign a follow/unfollow tx",
		RunE:  sendFollowTx(cdc),
	}
	cmd.Flags().String(client.FlagFollower, "", "signer of this transaction")
	cmd.Flags().Bool(client.FlagIsFollow, true, "false if this is unfollow")
	cmd.Flags().String(client.FlagFollowee, "", "target to follow or unfollow")
	return cmd
}

// send follow transaction to the blockchain
func sendFollowTx(cdc *wire.Codec) client.CommandTxCallback {
	return func(cmd *cobra.Command, args []string) error {
		ctx := client.NewCoreContextFromViper()
		follower := viper.GetString(client.FlagFollower)
		followee := viper.GetString(client.FlagFollowee)

		var msg sdk.Msg
		isFollow := viper.GetBool(client.FlagIsFollow)
		if isFollow {
			msg = acc.NewFollowMsg(follower, followee)
		} else {
			msg = acc.NewUnfollowMsg(follower, followee)
		}

		// build and sign the transaction, then broadcast to Tendermint
		res, err := ctx.SignBuildBroadcast(msg, cdc)

		if err != nil {
			return err
		}

		fmt.Printf("Committed at block %d. Hash: %s\n", res.Height, res.Hash.String())
		return nil
	}
}
