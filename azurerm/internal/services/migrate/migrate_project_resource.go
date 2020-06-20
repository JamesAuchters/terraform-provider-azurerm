package migrate

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/migrate/mgmt/2018-02-02/migrate"
	"github.com/hashicorp/go-azure-helpers/response"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/dns/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	azSchema "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmMigrateProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmMigrateProjectCreateUpdate,
		Read:   resourceAArmMigrateProjectRead,
		Update: resourceArmMigrateProjectCreateUpdate,
		Delete: resourceArmMigrateProjectDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Importer: azSchema.ValidateResourceIDPriorToImport(func(id string) error {
			_, err := parse.MigrateProjectID(id)
			return err
		}),

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile("^[a-zA-Z]{1}[-a-z0-9]{2,23}$"),
					"Migrate Project name must be 3 - 24 characters long, start with a letter, contain only letters, numbers and -.",
				),
			},

			"location": azure.SchemaLocation(),

			"resource_group_name": azure.SchemaResourceGroupNameDiffSuppress(),

			"tags": tags.Schema(),
		},
	}
}

func resourceArmMigrateProjectCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Migrate.ProjectClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	if d.IsNewResource() {
		existing, err := client.Get(ctx, resGroup, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Migrate Project %q (Resource Group %q): %s", name, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_migrate_project", *existing.ID)
		}
	}

	location := d.Get("location")
	t := d.Get("tags").(map[string]interface{})

	parameters := migrate.Project{
		Location: &location,
		Tags:     tags.Expand(t),
	}

	//suspect not required, resource copy from dns zone, specifc to dns zones.
	//etag := ""
	//ifNoneMatch := "" // set to empty to allow updates to records after creation
	//TODO:
	if _, err := client.Create(ctx, resGroup, name, parameters); err != nil {
		return fmt.Errorf("Error creating/updating Migrate Project %q (Resource Group %q): %s", name, resGroup, err)
	}

	resp, err := client.Get(ctx, resGroup, name)
	if err != nil {
		return fmt.Errorf("Error retrieving Migrate Project %q (Resource Group %q): %s", name, resGroup, err)
	}

	if resp.ID == nil {
		return fmt.Errorf("Cannot read Migrate Project %q (Resource Group %q) ID", name, resGroup)
	}
	//SetId? I don't know where this function is
	d.SetId(*resp.ID)

	return resourceArmMigrateProjectRead(d, meta)
}

func resourceArmMigrateProjectRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Migrate.ProjectClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.MigrateProjectID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading Migrate Project %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	d.Set("name", id.Name)
	d.Set("resource_group_name", id.ResourceGroup)

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmMigrateProjectDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Migrate.ProjectClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.MigrateProjectID(d.Id())
	if err != nil {
		return err
	}

	etag := ""
	future, err := client.Delete(ctx, id.ResourceGroup, id.Name, etag)
	if err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}
		return fmt.Errorf("Error deleting Migrate Project %s (resource group %s): %+v", id.Name, id.ResourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}
		return fmt.Errorf("Error deleting migrate project %s (resource group %s): %+v", id.Name, id.ResourceGroup, err)
	}

	return nil
}
