package client

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/verifa/bubbly/env"
)

// PostSchema uses the bubbly api endpoint to get a resource
func (c *Client) PostSchema(bCtx *env.BubblyContext, schema []byte) error {

	_, err := handleResponse(
		http.Post(fmt.Sprintf("%s/api/schema", c.HostURL), "application/json", bytes.NewBuffer(schema)),
	)
	return err
}