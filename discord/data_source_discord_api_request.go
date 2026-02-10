package discord

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/url"
	"strconv"
)

// discord_api_request is a generic GET-only data source for covering endpoints not yet modeled.
// This should be treated as an escape hatch; prefer first-class resources when available.
func dataSourceDiscordAPIRequest() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordAPIRequestRead,
		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "API path starting with '/'. Example: '/guilds/{guild_id}/channels'.",
			},
			"query_json": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "JSON object encoded query params. Example: jsonencode({ limit = 100 })",
			},
			"response_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Normalized JSON response body.",
			},
		},
	}
}

func dataSourceDiscordAPIRequestRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	path := d.Get("path").(string)

	var q map[string]interface{}
	if v := d.Get("query_json").(string); v != "" {
		if err := json.Unmarshal([]byte(v), &q); err != nil {
			return diag.FromErr(err)
		}
	}

	query := url.Values{}
	for k, v := range q {
		switch t := v.(type) {
		case string:
			query.Set(k, t)
		case bool:
			if t {
				query.Set(k, "true")
			} else {
				query.Set(k, "false")
			}
		case float64:
			// JSON numbers decode to float64.
			query.Set(k, strconv.FormatFloat(t, 'f', -1, 64))
		case []interface{}:
			for _, item := range t {
				query.Add(k, stringifyQueryValue(item))
			}
		default:
			// Objects or other complex types: pass JSON.
			b, _ := json.Marshal(t)
			query.Set(k, string(b))
		}
	}

	var out interface{}
	if err := c.DoJSON(ctx, "GET", path, query, nil, &out); err != nil {
		return diag.FromErr(err)
	}

	b, err := json.Marshal(out)
	if err != nil {
		return diag.FromErr(err)
	}
	norm, err := normalizeJSON(string(b))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(Hashcode(path + "|" + norm)))
	_ = d.Set("response_json", norm)
	return nil
}

func stringifyQueryValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}
