package commands

import (
	"code.cloudfoundry.org/credhub-cli/errors"
)

type FindCommand struct {
	PartialCredentialIdentifier string `short:"n" long:"name-like" description:"Find credentials whose name contains the query string"`
	PathIdentifier              string `short:"p" long:"path" description:"Find credentials that exist under the provided path"`
	AllPaths                    bool   `short:"a" long:"all-paths" description:"List all existing credential paths"`
	OutputJSON                  bool   `short:"j" long:"output-json" description:"Return response in JSON format"`
	ClientCommand
}

func (c *FindCommand) Execute([]string) error {
	if c.AllPaths {
		paths, err := c.client.FindAllPaths()

		if err != nil {
			return err
		}

		if len(paths.Paths) == 0 {
			return errors.NewNoMatchingCredentialsFoundError()
		}

		printCredential(c.OutputJSON, paths)

	} else if c.PartialCredentialIdentifier != "" {
		results, err := c.client.FindByPartialName(c.PartialCredentialIdentifier)
		if err != nil {
			return err
		}

		if len(results.Credentials) == 0 {
			return errors.NewNoMatchingCredentialsFoundError()
		}

		printCredential(c.OutputJSON, results)
	} else {
		output, err := c.client.FindByPath(c.PathIdentifier)
		if err != nil {
			return err
		}

		printCredential(c.OutputJSON, output)
	}

	return nil
}
