package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func TestAccAzureRMMigrateProject_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_migrate_project", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMMigrateProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMMigrateProject_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMMigrateProjectExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "charset", "utf8"),
					resource.TestCheckResourceAttr(data.ResourceName, "collation", "utf8_general_ci"),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccAzureRMMigrateProject_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_migrate_project", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMMariaDbDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMMigrateProject_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMMigrateProjectExists(data.ResourceName),
				),
			},
			{
				Config:      testAccAzureRMMigrateProject_requiresImport(data),
				ExpectError: acceptance.RequiresImportError("azurerm_migrate_project"),
			},
		},
	})
}

func testCheckAzureRMMigrateProjectExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.AzureProvider.Meta().(*clients.Client).Migrate.ProjectClient
		ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %q", resourceName)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("bad: no resource group found in state for Migrate Project: %q", name)
		}

		resp, err := client.Get(ctx, resourceGroup, name)
		if err != nil {
			if utils.ResponseWasNotFound(resp.Response) {
				return fmt.Errorf("bad: Migrate Project %q (Resource Group: %q) does not exist", name, resourceGroup)
			}
			return fmt.Errorf("bad: get on MigrateProject: %+v", err)
		}

		return nil
	}
}

func testCheckAzureRMMigrateProjectDestroy(s *terraform.State) error {
	client := acceptance.AzureProvider.Meta().(*clients.Client).Migrate.ProjectClient
	ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_migrate_project" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := client.Get(ctx, resourceGroup, name)
		if err != nil {
			if utils.ResponseWasNotFound(resp.Response) {
				return nil
			}
			return fmt.Errorf("error Migrate Project %q (Resource Group %q) still exists:\n%+v", name, resourceGroup, err)
		}
		return fmt.Errorf("Migrate Project %q (Resource Group %q) still exists:\n%#+v", name, resourceGroup, resp)
	}

	return nil
}

func testAccAzureRMMigrateProject_basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = %q
}

resource "azurerm_migrate_project" "test" {
  name                = "migproject-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func testAccAzureRMMigrateProject_requiresImport(data acceptance.TestData) string {
	template := testAccAzureRMMigrateProject_basic(data)
	return fmt.Sprintf(`
%s

resource "azurerm_mariadb_database" "import" {
  name                = azurerm_mariadb_database.test.name
  resource_group_name = azurerm_mariadb_database.test.resource_group_name
}
`, template)
}
