package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type configService struct {
	conf         configPart          `yaml:"inline,omitempty"`
	clientConfig *CosmosClientConfig `yaml:"inline,omitempty"`
}

type configPart interface {
	CreateNewConfig(string)
	ValidateConfig() error
}

func NewConfigService(c configPart) configService {
	return configService{
		conf:         c,
		clientConfig: &CosmosClientConfig{},
	}
}

// MustYAML returns the yaml string representation of the config
func (c *configService) MustYAML() []byte {
	out, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}
	return out
}

// initConfig reads in config file and ENV variables if set.
// This is called as a persistent pre-run command of the root command.
func initConfig(cmd *cobra.Command, a *appState, o map[string]ClientOverrides) error {
	if a.Config == nil {
		panic("Must initialize app config with NewConfigService")
	}

	home, err := cmd.PersistentFlags().GetString(flags.FlagHome)
	if err != nil {
		return err
	}

	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return err
	}

	cfgPath := path.Join(home, "config.yaml")
	_, err = os.Stat(cfgPath)
	if err != nil {
		err = a.Config.createNewConfig(home, debug)
		if err != nil {
			return err
		}
	}
	a.Viper.SetConfigFile(cfgPath)
	err = a.Viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("failed to read in config: %w", err)
	}

	// read the config file bytes
	file, err := os.ReadFile(a.Viper.ConfigFileUsed())
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// unmarshall them into the struct
	if err = yaml.Unmarshal(file, &a.Config); err != nil {
		return fmt.Errorf("error unmarshalling config: %w", err)
	}

	initErr := a.Initialize(home, a.Log, cmd, o)
	if initErr != nil {
		return initErr
	}

	// validate configuration
	if err := a.Config.conf.ValidateConfig(); err != nil {
		return fmt.Errorf("error validating config: %w", err)
	}
	return nil
}

// createConfig idempotently creates the config.
func (c *configService) createNewConfig(home string, debug bool) error {
	cfgPath := path.Join(home, "config.yaml")

	// If the config doesn't exist...
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// And the config folder doesn't exist...
		// And the home folder doesn't exist
		if _, err := os.Stat(home); os.IsNotExist(err) {
			// Create the home folder
			if err = os.Mkdir(home, os.ModePerm); err != nil {
				return err
			}
		}
	}

	if c.conf != nil {
		c.conf.CreateNewConfig(home)
	}
	if c.clientConfig != nil {
		c.clientConfig.CreateNewConfig(home)
	}
	content := c.MustYAML()

	// Then create the file...
	//content := defaultConfig(path.Join(home, "keys"), debug)
	if err := os.WriteFile(cfgPath, content, 0600); err != nil {
		return err
	}

	return nil
}
