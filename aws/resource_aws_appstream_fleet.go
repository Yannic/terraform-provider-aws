package aws

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appstream"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/appstream/waiter"
)

func resourceAwsAppStreamFleet() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceAwsAppStreamFleetCreate,
		ReadWithoutTimeout:   resourceAwsAppStreamFleetRead,
		UpdateWithoutTimeout: resourceAwsAppStreamFleetUpdate,
		DeleteWithoutTimeout: resourceAwsAppStreamFleetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"compute_capacity": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"available": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"desired_instances": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"in_use": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"running": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"created_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringLenBetween(0, 256),
			},
			"disconnect_timeout_in_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(60, 360000),
			},
			"display_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringLenBetween(0, 100),
			},
			"domain_join_info": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"directory_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"organizational_unit_distinguished_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"enable_default_internet_access": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"fleet_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(appstream.FleetType_Values(), false),
			},
			"iam_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateArn,
			},
			"idle_disconnect_timeout_in_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				ValidateFunc: validation.IntBetween(60, 3600),
			},
			"image_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"image_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"instance_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"max_user_duration_in_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(600, 360000),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"stream_view": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(appstream.StreamView_Values(), false),
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_config": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"security_group_ids": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"subnet_ids": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"tags":     tagsSchema(),
			"tags_all": tagsSchemaComputed(),
		},
	}
}

func resourceAwsAppStreamFleetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*AWSClient).appstreamconn
	input := &appstream.CreateFleetInput{
		Name:            aws.String(d.Get("name").(string)),
		InstanceType:    aws.String(d.Get("instance_type").(string)),
		ComputeCapacity: expandComputeCapacity(d.Get("compute_capacity").([]interface{})),
	}

	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("disconnect_timeout_in_seconds"); ok {
		input.DisconnectTimeoutInSeconds = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("idle_disconnect_timeout_in_seconds"); ok {
		input.IdleDisconnectTimeoutInSeconds = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("display_name"); ok {
		input.DisplayName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("domain_join_info"); ok {
		input.DomainJoinInfo = expandDomainJoinInfo(v.([]interface{}))
	}

	if v, ok := d.GetOk("enable_default_internet_access"); ok {
		input.EnableDefaultInternetAccess = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("fleet_type"); ok {
		input.FleetType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("image_name"); ok {
		input.ImageName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("iam_role_arn"); ok {
		input.IamRoleArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("max_user_duration_in_seconds"); ok {
		input.MaxUserDurationInSeconds = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("vpc_config"); ok {
		input.VpcConfig = expandVpcConfig(v.([]interface{}))
	}

	if len(tags) > 0 {
		input.Tags = tags.IgnoreAws().AppstreamTags()
	}

	var err error
	var output *appstream.CreateFleetOutput
	err = resource.RetryContext(ctx, waiter.FleetOperationTimeout, func() *resource.RetryError {
		output, err = conn.CreateFleetWithContext(ctx, input)
		if err != nil {
			if tfawserr.ErrCodeEquals(err, appstream.ErrCodeResourceNotFoundException) {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isResourceTimeoutError(err) {
		output, err = conn.CreateFleetWithContext(ctx, input)
	}
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating Appstream Fleet (%s): %w", d.Id(), err))
	}

	// Start fleet workflow
	_, err = conn.StartFleetWithContext(ctx, &appstream.StartFleetInput{
		Name: output.Fleet.Name,
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("error starting Appstream Fleet (%s): %w", d.Id(), err))
	}

	if _, err = waiter.FleetStateRunning(ctx, conn, aws.StringValue(output.Fleet.Name)); err != nil {
		return diag.FromErr(fmt.Errorf("error waiting for Appstream Fleet (%s) to be running: %w", d.Id(), err))
	}

	d.SetId(aws.StringValue(output.Fleet.Name))

	return resourceAwsAppStreamFleetRead(ctx, d, meta)
}

func resourceAwsAppStreamFleetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*AWSClient).appstreamconn

	defaultTagsConfig := meta.(*AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	resp, err := conn.DescribeFleetsWithContext(ctx, &appstream.DescribeFleetsInput{Names: []*string{aws.String(d.Id())}})

	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, appstream.ErrCodeResourceNotFoundException) {
		log.Printf("[WARN] Appstream Fleet (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading Appstream Fleet (%s): %w", d.Id(), err))
	}

	if len(resp.Fleets) == 0 {
		return diag.FromErr(fmt.Errorf("error reading Appstream Fleet (%s): %s", d.Id(), "empty response"))
	}

	if len(resp.Fleets) > 1 {
		return diag.FromErr(fmt.Errorf("error reading Appstream Fleet (%s): %s", d.Id(), "multiple fleets found"))
	}

	fleet := resp.Fleets[0]

	d.Set("arn", fleet.Arn)

	if err = d.Set("compute_capacity", flattenComputeCapacity(fleet.ComputeCapacityStatus)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting `%s` for AppStream Fleet (%s): %w", "compute_capacity", d.Id(), err))
	}

	d.Set("created_time", aws.TimeValue(fleet.CreatedTime).Format(time.RFC3339))
	d.Set("description", fleet.Description)
	d.Set("display_name", fleet.DisplayName)
	d.Set("disconnect_timeout_in_seconds", fleet.DisconnectTimeoutInSeconds)

	if err = d.Set("domain_join_info", flattenDomainInfo(fleet.DomainJoinInfo)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting `%s` for AppStream Fleet (%s): %w", "domain_join_info", d.Id(), err))
	}

	d.Set("idle_disconnect_timeout_in_seconds", fleet.IdleDisconnectTimeoutInSeconds)
	d.Set("enable_default_internet_access", fleet.EnableDefaultInternetAccess)
	d.Set("fleet_type", fleet.FleetType)
	d.Set("iam_role_arn", fleet.IamRoleArn)
	d.Set("image_name", fleet.ImageName)
	d.Set("image_arn", fleet.ImageArn)
	d.Set("instance_type", fleet.InstanceType)
	d.Set("max_user_duration_in_seconds", fleet.MaxUserDurationInSeconds)
	d.Set("name", fleet.Name)
	d.Set("state", fleet.State)
	d.Set("stream_view", fleet.StreamView)

	if err = d.Set("vpc_config", flattenVpcConfig(fleet.VpcConfig)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting `%s` for AppStream Fleet (%s): %w", "vpc_config", d.Id(), err))
	}

	tg, err := conn.ListTagsForResource(&appstream.ListTagsForResourceInput{
		ResourceArn: fleet.Arn,
	})

	if err != nil {
		return diag.FromErr(fmt.Errorf("error listing stack tags for AppStream Stack (%s): %w", d.Id(), err))
	}

	if tg.Tags == nil {
		log.Printf("[DEBUG] AppStream Stack tags (%s) not found", d.Id())
		return nil
	}

	tags := keyvaluetags.AppstreamKeyValueTags(tg.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	if err = d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return diag.FromErr(fmt.Errorf("error setting `%s` for AppStream Stack (%s): %w", "tags", d.Id(), err))
	}

	if err = d.Set("tags_all", tags.Map()); err != nil {
		return diag.FromErr(fmt.Errorf("error setting `%s` for AppStream Stack (%s): %w", "tags_all", d.Id(), err))
	}

	return nil
}

func resourceAwsAppStreamFleetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*AWSClient).appstreamconn
	input := &appstream.UpdateFleetInput{
		Name: aws.String(d.Id()),
	}
	shouldStop := false

	if d.HasChanges("description", "domain_join_info", "enable_default_internet_access", "iam_role_arn", "instance_type", "max_user_duration_in_seconds", "stream_view", "vpc_config") {
		shouldStop = true
	}

	// Stop fleet workflow if needed
	if shouldStop {
		_, err := conn.StopFleetWithContext(ctx, &appstream.StopFleetInput{
			Name: aws.String(d.Id()),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("error stopping Appstream Fleet (%s): %w", d.Id(), err))
		}
		if _, err = waiter.FleetStateStopped(ctx, conn, d.Id()); err != nil {
			return diag.FromErr(fmt.Errorf("error waiting for Appstream Fleet (%s) to be stopped: %w", d.Id(), err))
		}
	}

	if d.HasChange("compute_capacity") {
		input.ComputeCapacity = expandComputeCapacity(d.Get("compute_capacity").([]interface{}))
	}

	if d.HasChange("description") {
		input.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("domain_join_info") {
		input.DomainJoinInfo = expandDomainJoinInfo(d.Get("domain_join_info").([]interface{}))
	}

	if d.HasChange("disconnect_timeout_in_seconds") {
		input.DisconnectTimeoutInSeconds = aws.Int64(int64(d.Get("disconnect_timeout_in_seconds").(int)))
	}

	if d.HasChange("enable_default_internet_access") {
		input.EnableDefaultInternetAccess = aws.Bool(d.Get("enable_default_internet_access").(bool))
	}

	if d.HasChange("idle_disconnect_timeout_in_seconds") {
		input.IdleDisconnectTimeoutInSeconds = aws.Int64(int64(d.Get("idle_disconnect_timeout_in_seconds").(int)))
	}

	if d.HasChange("display_name") {
		input.DisplayName = aws.String(d.Get("display_name").(string))
	}

	if d.HasChange("image_name") {
		input.ImageName = aws.String(d.Get("image_name").(string))
	}

	if d.HasChange("image_arn") {
		input.ImageArn = aws.String(d.Get("image_arn").(string))
	}

	if d.HasChange("iam_role_arn") {
		input.IamRoleArn = aws.String(d.Get("iam_role_arn").(string))
	}

	if d.HasChange("stream_view") {
		input.StreamView = aws.String(d.Get("stream_view").(string))
	}

	if d.HasChange("instance_type") {
		input.InstanceType = aws.String(d.Get("instance_type").(string))
	}

	if d.HasChange("max_user_duration_in_seconds") {
		input.MaxUserDurationInSeconds = aws.Int64(int64(d.Get("max_user_duration_in_seconds").(int)))
	}

	if d.HasChange("vpc_config") {
		input.VpcConfig = expandVpcConfig(d.Get("vpc_config").([]interface{}))
	}

	resp, err := conn.UpdateFleetWithContext(ctx, input)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating Appstream Fleet (%s): %w", d.Id(), err))
	}

	if d.HasChange("tags") {
		arn := aws.StringValue(resp.Fleet.Arn)

		o, n := d.GetChange("tags")
		if err := keyvaluetags.AppstreamUpdateTags(conn, arn, o, n); err != nil {
			return diag.FromErr(fmt.Errorf("error updating Appstream Fleet tags (%s): %w", d.Id(), err))
		}
	}

	// Start fleet workflow if stopped
	if shouldStop {
		_, err = conn.StartFleetWithContext(ctx, &appstream.StartFleetInput{
			Name: aws.String(d.Id()),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("error starting Appstream Fleet (%s): %w", d.Id(), err))
		}

		if _, err = waiter.FleetStateRunning(ctx, conn, d.Id()); err != nil {
			return diag.FromErr(fmt.Errorf("error waiting for Appstream Fleet (%s) to be running: %w", d.Id(), err))
		}
	}

	return resourceAwsAppStreamFleetRead(ctx, d, meta)
}

func resourceAwsAppStreamFleetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*AWSClient).appstreamconn

	// Stop fleet workflow
	_, err := conn.StopFleetWithContext(ctx, &appstream.StopFleetInput{
		Name: aws.String(d.Id()),
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("error stopping Appstream Fleet (%s): %w", d.Id(), err))
	}

	if _, err = waiter.FleetStateStopped(ctx, conn, d.Id()); err != nil {
		return diag.FromErr(fmt.Errorf("error waiting for Appstream Fleet (%s) to be stopped: %w", d.Id(), err))
	}

	_, err = conn.DeleteFleetWithContext(ctx, &appstream.DeleteFleetInput{
		Name: aws.String(d.Id()),
	})

	if err != nil {
		if tfawserr.ErrCodeEquals(err, appstream.ErrCodeResourceNotFoundException) {
			return nil
		}
		return diag.FromErr(fmt.Errorf("error deleting Appstream Fleet (%s): %w", d.Id(), err))
	}
	return nil
}

func expandComputeCapacity(tfList []interface{}) *appstream.ComputeCapacity {
	if len(tfList) == 0 {
		return nil
	}

	apiObject := &appstream.ComputeCapacity{}

	attr := tfList[0].(map[string]interface{})
	if v, ok := attr["desired_instances"]; ok {
		apiObject.DesiredInstances = aws.Int64(int64(v.(int)))
	}

	return apiObject
}

func flattenComputeCapacity(apiObject *appstream.ComputeCapacityStatus) []interface{} {
	if apiObject == nil {
		return nil
	}

	tfList := map[string]interface{}{}
	tfList["desired_instances"] = aws.Int64Value(apiObject.Desired)
	tfList["available"] = aws.Int64Value(apiObject.Available)
	tfList["in_use"] = aws.Int64Value(apiObject.InUse)
	tfList["running"] = aws.Int64Value(apiObject.Running)

	return []interface{}{tfList}
}

func expandDomainJoinInfo(tfList []interface{}) *appstream.DomainJoinInfo {
	if len(tfList) == 0 {
		return nil
	}

	apiObject := &appstream.DomainJoinInfo{}

	tfMap := tfList[0].(map[string]interface{})
	if v, ok := tfMap["directory_name"]; ok {
		apiObject.DirectoryName = aws.String(v.(string))
	}
	if v, ok := tfMap["organizational_unit_distinguished_name"]; ok {
		apiObject.OrganizationalUnitDistinguishedName = aws.String(v.(string))
	}

	return apiObject
}

func flattenDomainInfo(apiObject *appstream.DomainJoinInfo) []interface{} {
	if apiObject == nil {
		return nil
	}

	tfList := map[string]interface{}{}
	tfList["directory_name"] = aws.StringValue(apiObject.DirectoryName)
	tfList["organizational_unit_distinguished_name"] = aws.StringValue(apiObject.OrganizationalUnitDistinguishedName)

	return []interface{}{tfList}
}

func expandVpcConfig(tfList []interface{}) *appstream.VpcConfig {
	if len(tfList) == 0 {
		return nil
	}

	apiObject := &appstream.VpcConfig{}

	tfMap := tfList[0].(map[string]interface{})
	if v, ok := tfMap["security_group_ids"]; ok {
		apiObject.SecurityGroupIds = expandStringList(v.([]interface{}))
	}
	if v, ok := tfMap["subnet_ids"]; ok {
		apiObject.SubnetIds = expandStringList(v.([]interface{}))
	}

	return apiObject
}

func flattenVpcConfig(apiObject *appstream.VpcConfig) []interface{} {
	if apiObject == nil {
		return nil
	}

	tfList := map[string]interface{}{}
	tfList["security_group_ids"] = aws.StringValueSlice(apiObject.SecurityGroupIds)
	tfList["subnet_ids"] = aws.StringValueSlice(apiObject.SubnetIds)

	return []interface{}{tfList}
}
