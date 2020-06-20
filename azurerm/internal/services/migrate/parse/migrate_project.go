package parse

import (
	"fmt"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

type MigrateProjectId struct {
	ResourceGroup string
	Name          string
}

func MigrateProjectID(input string) (*MediaServicesAccountId, error) {
	id, err := azure.ParseAzureResourceID(input)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Unable to parse Migrate Project ID %q: %+v", input, err)
	}

	migrate := MigrateProjectId{
		ResourceGroup: id.ResourceGroup,
	}

	if migrate.Name, err = id.PopSegment("migrateprojects"); err != nil {
		return nil, err
	}

	if err := id.ValidateNoEmptySegments(input); err != nil {
		return nil, err
	}

	return &migrate, nil
}
