package cosmos

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2020-04-01/documentdb"
	"github.com/hashicorp/go-azure-helpers/response"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/cosmos/common"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/cosmos/migration"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/cosmos/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmCosmosGremlinDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmCosmosGremlinDatabaseCreate,
		Update: resourceArmCosmosGremlinDatabaseUpdate,
		Read:   resourceArmCosmosGremlinDatabaseRead,
		Delete: resourceArmCosmosGremlinDatabaseDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    migration.ResourceGremlinDatabaseUpgradeV0Schema().CoreConfigSchema().ImpliedType(),
				Upgrade: migration.ResourceGremlinDatabaseStateUpgradeV0ToV1,
				Version: 0,
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosEntityName,
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			"account_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosAccountName,
			},

			"throughput": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validate.CosmosThroughput,
			},
		},
	}
}

func resourceArmCosmosGremlinDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.GremlinClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)
	account := d.Get("account_name").(string)

	if features.ShouldResourcesBeImported() {
		existing, err := client.GetGremlinDatabase(ctx, resourceGroup, account, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of creating Gremlin Database %q (Account: %q): %+v", name, account, err)
			}
		} else {
			if existing.ID == nil && *existing.ID == "" {
				return fmt.Errorf("Error generating import ID for Cosmos Gremlin Database %q (Account: %q)", name, account)
			}

			return tf.ImportAsExistsError("azurerm_cosmosdb_gremlin_database", *existing.ID)
		}
	}

	db := documentdb.GremlinDatabaseCreateUpdateParameters{
		GremlinDatabaseCreateUpdateProperties: &documentdb.GremlinDatabaseCreateUpdateProperties{
			Resource: &documentdb.GremlinDatabaseResource{
				ID: &name,
			},
			Options: &documentdb.CreateUpdateOptions{},
		},
	}

	if throughput, hasThroughput := d.GetOk("throughput"); hasThroughput {
		db.GremlinDatabaseCreateUpdateProperties.Options.Throughput = common.ConvertThroughputFromResourceData(throughput)
	}

	future, err := client.CreateUpdateGremlinDatabase(ctx, resourceGroup, account, name, db)
	if err != nil {
		return fmt.Errorf("Error issuing create/update request for Gremlin Database %q (Account: %q): %+v", name, account, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting on create/update future for Cosmos Gremlin Database %q (Account: %q): %+v", name, account, err)
	}

	resp, err := client.GetGremlinDatabase(ctx, resourceGroup, account, name)
	if err != nil {
		return fmt.Errorf("Error making get request for Cosmos Gremlin Database %q (Account: %q) ID: %v", name, account, err)
	}

	if resp.ID == nil {
		return fmt.Errorf("Error getting ID from Cosmos Gremlin Database %q (Account: %q)", name, account)
	}

	d.SetId(*resp.ID)

	return resourceArmCosmosGremlinDatabaseRead(d, meta)
}

func resourceArmCosmosGremlinDatabaseUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.GremlinClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.GremlinDatabaseID(d.Id())
	if err != nil {
		return err
	}

	db := documentdb.GremlinDatabaseCreateUpdateParameters{
		GremlinDatabaseCreateUpdateProperties: &documentdb.GremlinDatabaseCreateUpdateProperties{
			Resource: &documentdb.GremlinDatabaseResource{
				ID: &id.Name,
			},
			Options: &documentdb.CreateUpdateOptions{},
		},
	}

	future, err := client.CreateUpdateGremlinDatabase(ctx, id.ResourceGroup, id.Account, id.Name, db)
	if err != nil {
		return fmt.Errorf("Error issuing create/update request for Cosmos Gremlin Database %q (Account: %q): %+v", id.Name, id.Account, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting on create/update future for Cosmos Gremlin Database %q (Account: %q): %+v", id.Name, id.Account, err)
	}

	if d.HasChange("throughput") {
		throughputParameters := documentdb.ThroughputSettingsUpdateParameters{
			ThroughputSettingsUpdateProperties: &documentdb.ThroughputSettingsUpdateProperties{
				Resource: &documentdb.ThroughputSettingsResource{
					Throughput: common.ConvertThroughputFromResourceData(d.Get("throughput")),
				},
			},
		}

		throughputFuture, err := client.UpdateGremlinDatabaseThroughput(ctx, id.ResourceGroup, id.Account, id.Name, throughputParameters)
		if err != nil {
			if response.WasNotFound(throughputFuture.Response()) {
				return fmt.Errorf("Error setting Throughput for Cosmos Gremlin Database %q (Account: %q): %+v - "+
					"If the collection has not been created with and initial throughput, you cannot configure it later.", id.Name, id.Account, err)
			}
		}

		if err = throughputFuture.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("Error waiting on ThroughputUpdate future for Cosmos Gremlin Database %q (Account: %q, Database %q): %+v", id.Name, id.Account, id.Name, err)
		}
	}

	if _, err = client.GetGremlinDatabase(ctx, id.ResourceGroup, id.Account, id.Name); err != nil {
		return fmt.Errorf("Error making get request for Cosmos Gremlin Database %q (Account: %q): %+v", id.Name, id.Account, err)
	}

	return resourceArmCosmosGremlinDatabaseRead(d, meta)
}

func resourceArmCosmosGremlinDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.GremlinClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.GremlinDatabaseID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.GetGremlinDatabase(ctx, id.ResourceGroup, id.Account, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Error reading Cosmos Gremlin Database %q (Account: %q) - removing from state", id.Name, id.Account)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Cosmos Gremlin Database %q (Account: %q): %+v", id.Name, id.Account, err)
	}

	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("account_name", id.Account)
	if props := resp.GremlinDatabaseGetProperties; props != nil {
		if res := props.Resource; res != nil {
			d.Set("name", res.ID)
		}
	}

	throughputResp, err := client.GetGremlinDatabaseThroughput(ctx, id.ResourceGroup, id.Account, id.Name)
	if err != nil {
		if !utils.ResponseWasNotFound(throughputResp.Response) {
			return fmt.Errorf("Error reading Throughput on Cosmos Gremlin Database %q (Account: %q): %+v", id.Name, id.Account, err)
		} else {
			d.Set("throughput", nil)
		}
	} else {
		d.Set("throughput", common.GetThroughputFromResult(throughputResp))
	}

	return nil
}

func resourceArmCosmosGremlinDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.GremlinClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.GremlinDatabaseID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.DeleteGremlinDatabase(ctx, id.ResourceGroup, id.Account, id.Name)
	if err != nil {
		if !response.WasNotFound(future.Response()) {
			return fmt.Errorf("Error deleting Cosmos Gremlin Database %q (Account: %q): %+v", id.Name, id.Account, err)
		}
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting on delete future for Cosmos Gremlin Database %q (Account: %q): %+v", id.Name, id.Account, err)
	}

	return nil
}
