package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAwsSsmParametersByPath() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSsmParametersReadByPath,

		Schema: map[string]*schema.Schema{
			"arns": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"names": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"path": {
				Type:     schema.TypeString,
				Required: true,
			},
			"types": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"values": {
				Type:      schema.TypeList,
				Computed:  true,
				Sensitive: true,
				Elem:      &schema.Schema{Type: schema.TypeString},
			},
			"with_decryption": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func dataSourceAwsSsmParametersReadByPath(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	path := d.Get("path").(string)
	input := &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		WithDecryption: aws.Bool(d.Get("with_decryption").(bool)),
	}

	arns := make([]string, 0)
	names := make([]string, 0)
	types := make([]string, 0)
	values := make([]string, 0)

	err := ssmconn.GetParametersByPathPages(input, func(page *ssm.GetParametersByPathOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, param := range page.Parameters {
			arns = append(arns, aws.StringValue(param.ARN))
			names = append(names, aws.StringValue(param.Name))
			types = append(types, aws.StringValue(param.Type))
			values = append(values, aws.StringValue(param.Value))
		}

		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("error getting SSM parameters by path (%s): %w", path, err)
	}

	d.SetId(path)
	d.Set("arns", arns)
	d.Set("names", names)
	d.Set("types", types)
	d.Set("values", values)

	return nil
}
