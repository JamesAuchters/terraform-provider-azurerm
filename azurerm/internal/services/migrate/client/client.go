package client

import (
	"github.com/Azure/azure-sdk-for-go/services/migrate/mgmt/2018-02-02/migrate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/common"
)

type Client struct {
	ProjectClient *migrate.ProjectClient
}

func NewClient(o *common.ClientOptions) *Client {
	ProjectClient := migrate.NewWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&ProjectClient.Client, o.ResourceManagerAuthorizer)

	return &Client{
		ProjectClient: &ProjectClient,
	}
}
