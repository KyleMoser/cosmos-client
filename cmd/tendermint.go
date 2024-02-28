/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/cosmos-client/client/query"
	"go.uber.org/zap"
)

// tendermintCmd represents the tendermint command
func tendermintCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tendermint",
		Aliases: []string{"tm"},
		Short:   "all tendermint query commands",
	}
	cmd.AddCommand(
		healthCmd(a),
		netInfoCmd(a),
		statusCmd(a),
	)
	return cmd
}

func healthCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "health",
		Aliases: []string{"h", "ok"},
		Short:   "query to see if node server is online",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := a.GetDefaultClient()
			block, err := cl.RPCClient.Health(cmd.Context())
			if err != nil {
				return err
			}
			if err := writeJSON(cmd.OutOrStdout(), block); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func netInfoCmd(a *appState) *cobra.Command {
	// TODO: add flag for pulling out comma seperated list of peers
	// and also filter out private IPs and other ill formed peers
	// _{*extraCredit*}_
	cmd := &cobra.Command{
		Use:     "net-info",
		Aliases: []string{"ni", "net", "netinfo", "peers"},
		Short:   "query for p2p network connection information",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := a.GetDefaultClient()
			peers, err := cmd.Flags().GetBool("peers")
			if err != nil {
				return err
			}
			block, err := cl.RPCClient.NetInfo(cmd.Context())
			if err != nil {
				return err
			}
			if !peers {
				if err := writeJSON(cmd.OutOrStdout(), block); err != nil {
					return err
				}
				return nil
			}
			peersList := make([]string, 0, len(block.Peers))
			for _, peer := range block.Peers {
				url, err := url.Parse(peer.NodeInfo.ListenAddr)
				if err != nil {
					a.Log.Info(
						"Failed to parse URL",
						zap.String("url", peer.NodeInfo.ListenAddr),
						zap.Error(err),
					)
					continue
				}
				peersList = append(peersList, fmt.Sprintf("%s@%s:%s", peer.NodeInfo.ID(), peer.RemoteIP, url.Port()))
			}
			fmt.Fprintln(cmd.OutOrStdout(), strings.Join(peersList, ","))
			return nil
		},
	}
	return peersFlag(cmd, a.Viper)
}

func statusCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"stat", "s"},
		Short:   "query the status of a node",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := a.GetDefaultClient()
			query := query.Query{Client: cl, Options: query.DefaultOptions()}

			status, err := query.Status()
			if err != nil {
				return err
			}
			return cl.PrintObject(status)
		},
	}
	return cmd
}
