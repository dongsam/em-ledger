package main

import (
	"fmt"
	"io"

	app "emoney"
	apptypes "emoney/types"
	tmtypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	db "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/server"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configureConsensus = func() {
	viper.Set("consensus.create_empty_blocks_interval", "60s")
	viper.Set("consensus.create_empty_blocks", false)
	viper.Set("consensus.timeout_commit", "250ms")
	viper.Set("consensus.timeout_propose", "2s")
	viper.Set("consensus.peer_gossip_sleep_duration", "25ms")
}

func main() {
	cobra.EnableCommandSorting = false

	apptypes.ConfigureSDK()
	cdc := app.MakeCodec()

	ctx := server.NewDefaultContext()
	// Add application to logging configuration
	logLevel := ctx.Config.BaseConfig.LogLevel
	ctx.Config.BaseConfig.LogLevel = fmt.Sprintf("emz:info,x/inflation:info,x/liquidityprovider:debug,%v", logLevel)

	configureConsensus()
	viper.Set("p2p.flush_throttle_timeout", "25ms")

	rootCmd := &cobra.Command{
		Use:               "daemon",
		Short:             "e-money validator node",
		PersistentPreRunE: persistentPreRunEFn(ctx),
	}

	rootCmd.AddCommand(genutilcli.InitCmd(ctx, cdc, app.ModuleBasics, app.DefaultNodeHome))
	rootCmd.AddCommand(addGenesisAccountCmd(ctx, cdc, app.DefaultNodeHome, app.DefaultCLIHome))
	rootCmd.AddCommand(testnetCmd(ctx, cdc, app.ModuleBasics))

	server.AddCommands(ctx, cdc, rootCmd, newAppCreator(ctx), nil)

	executor := cli.PrepareBaseCmd(rootCmd, "EMD", app.DefaultNodeHome)
	err := executor.Execute()
	if err != nil {
		panic(err)
	}
}

func newAppCreator(ctx *server.Context) func(log.Logger, db.DB, io.Writer) tmtypes.Application {
	return func(logger log.Logger, db db.DB, _ io.Writer) tmtypes.Application {
		return app.NewApp(logger, db, ctx)
	}
}
