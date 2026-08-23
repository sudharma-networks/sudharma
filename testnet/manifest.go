package testnet

import (
	"encoding/json"
	"fmt"
	"os"
)

// LaunchManifest is the public fingerprint operators and clients can use to
// verify that they are connecting to the intended Sudharma testnet.
type LaunchManifest struct {
	Name              string   `json:"name"`
	Slug              string   `json:"slug"`
	ProtocolNetworkID string   `json:"protocol_network_id"`
	GenesisHash       string   `json:"genesis_hash"`
	P2PPort           int      `json:"p2p_port"`
	RPCPort           int      `json:"rpc_port"`
	Seeds             []string `json:"seeds"`
}

func NewLaunchManifest(profile Profile) (LaunchManifest, error) {
	if err := profile.ValidatePublicLaunch(); err != nil {
		return LaunchManifest{}, err
	}
	return LaunchManifest{
		Name:              profile.Name,
		Slug:              profile.Slug,
		ProtocolNetworkID: ProtocolNetworkID,
		GenesisHash:       GenesisHash(),
		P2PPort:           profile.P2PPort,
		RPCPort:           profile.RPCPort,
		Seeds:             append([]string(nil), profile.Seeds...),
	}, nil
}

func LoadProfile(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, fmt.Errorf("read testnet profile: %w", err)
	}
	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("decode testnet profile: %w", err)
	}
	if err := profile.Validate(); err != nil {
		return Profile{}, err
	}
	return profile, nil
}
