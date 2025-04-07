package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/invopop/jsonschema"
)

func main() {
	if err := jsonSchemaConfiguration(); err != nil {
		panic(fmt.Errorf("failed to write jsonschema for configuration: %w", err))
	}
}

func jsonSchemaConfiguration() error {
	r := new(jsonschema.Reflector)
	if err := r.AddGoComments("github.com/hasura/ndc-prometheus/connector/client", "../connector/client"); err != nil {
		return err
	}

	if err := r.AddGoComments("github.com/hasura/ndc-prometheus/connector/metadata", "../connector/metadata"); err != nil {
		return err
	}

	if err := r.AddGoComments("github.com/hasura/ndc-prometheus/connector/types", "../connector/types"); err != nil {
		return err
	}

	reflectSchema := r.Reflect(&metadata.Configuration{})

	schemaBytes, err := json.MarshalIndent(reflectSchema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("configuration.json", schemaBytes, 0o644)
}
