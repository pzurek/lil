package linear

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/Khan/genqlient/graphql"

	"github.com/pzurek/lil/internal/linear/schema"
)

// authTransport is a custom transport that adds the Authorization header correctly.
type authTransport struct {
	apiKey string
	base   http.RoundTripper
}

// RoundTrip adds the Authorization header to the request without the "Bearer" prefix.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Use the original request to avoid modifying it globally if base transport reuses it
	reqClone := req.Clone(req.Context())
	reqClone.Header.Set("Authorization", t.apiKey)          // Set header directly
	reqClone.Header.Set("Content-Type", "application/json") // Ensure content type is set
	// Use the base transport (e.g., http.DefaultTransport) to execute the request
	return t.base.RoundTrip(reqClone)
}

// GetClient creates and returns a new GraphQL client configured for Linear.
func GetClient() (graphql.Client, error) {
	apiKey := os.Getenv("LINEAR_API_KEY")
	if apiKey == "" {
		return nil, errors.New("LINEAR_API_KEY environment variable not set")
	}

	// Create the custom transport
	authTransport := &authTransport{
		apiKey: apiKey,
		base:   http.DefaultTransport, // Use default transport as base
	}

	// Create an http.Client using the custom transport
	httpClient := &http.Client{
		Transport: authTransport,
	}

	// Create the genqlient client using the custom http.Client
	client := graphql.NewClient("https://api.linear.app/graphql", httpClient)
	return client, nil
}

// FetchAssignedIssues retrieves the assigned issues for the current user.
func FetchAssignedIssues(ctx context.Context) ([]schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue, error) {
	client, err := GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get linear client: %w", err)
	}

	resp, err := schema.GetAssignedIssues(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetAssignedIssues query: %w", err)
	}

	if resp == nil {
		return nil, errors.New("received nil response from GetAssignedIssues query")
	}

	if resp.Viewer.AssignedIssues.Nodes == nil {
		return []schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{}, nil
	}

	return resp.Viewer.AssignedIssues.Nodes, nil
}

// Note: All other functions, structs, constants, authTransport removed.
