package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/url"
	"strings"
)

// discord_api_resource is a generic "escape hatch" resource for JSON-based Discord API endpoints.
//
// This is intentionally flexible and therefore potentially dangerous. Prefer first-class resources.
func resourceDiscordAPIResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordAPIResourceCreate,
		ReadContext:   resourceDiscordAPIResourceRead,
		UpdateContext: resourceDiscordAPIResourceUpdate,
		DeleteContext: resourceDiscordAPIResourceDelete,
		CustomizeDiff: resourceDiscordAPIResourceCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"id_field": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "id",
				Description: "Field name to extract as resource ID from create response (JSON object).",
			},
			"id_override": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "If set, uses this value as the Terraform resource ID instead of parsing the create response.",
			},

			"create_method": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "POST",
				ValidateFunc: validateHTTPMethodOrSkip,
			},
			"create_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path for create call. May include '{id}' placeholder.",
			},
			"create_body_json": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"read_path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path for read call. Should include '{id}' placeholder.",
			},
			"read_query_json": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "JSON object of query parameters for read.",
			},

			"update_method": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "PATCH",
				ValidateFunc: validateHTTPMethodOrSkip,
			},
			"update_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path for update call. Defaults to read_path.",
			},
			"update_body_json": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"delete_method": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "DELETE",
				ValidateFunc: validateHTTPMethodOrSkip,
			},
			"delete_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path for delete call. Defaults to read_path.",
			},
			"delete_body_json": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"reason": {
				Type:     schema.TypeString,
				Optional: true,
				// Not readable
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
			},

			"response_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Normalized JSON response from read.",
			},
		},
	}
}

func validateHTTPMethodOrSkip(val interface{}, key string) (warns []string, errs []error) {
	s := strings.ToUpper(val.(string))
	switch s {
	case "SKIP", "GET", "POST", "PUT", "PATCH", "DELETE":
		return nil, nil
	default:
		return nil, []error{fmt.Errorf("%s must be one of SKIP, GET, POST, PUT, PATCH, DELETE", key)}
	}
}

func resourceDiscordAPIResourceCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, _ interface{}) error {
	createMethod := strings.ToUpper(d.Get("create_method").(string))
	createPath := strings.TrimSpace(d.Get("create_path").(string))
	idOverride := strings.TrimSpace(d.Get("id_override").(string))

	if createMethod == "SKIP" {
		if idOverride == "" {
			return fmt.Errorf("id_override is required when create_method=SKIP")
		}
	} else {
		if createPath == "" {
			return fmt.Errorf("create_path is required when create_method is not SKIP")
		}
		if strings.Contains(createPath, "{id}") && idOverride == "" {
			return fmt.Errorf("id_override is required when create_path contains {id}")
		}
	}

	return nil
}

func substituteID(path, id string) string {
	return strings.ReplaceAll(path, "{id}", id)
}

func parseJSONBody(s string) (interface{}, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	return v, nil
}

func parseQueryJSON(s string) (url.Values, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	q := url.Values{}
	for k, v := range m {
		switch t := v.(type) {
		case string:
			q.Set(k, t)
		case bool:
			if t {
				q.Set(k, "true")
			} else {
				q.Set(k, "false")
			}
		default:
			b, _ := json.Marshal(t)
			q.Set(k, string(b))
		}
	}
	return q, nil
}

func resourceDiscordAPIResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	reason := d.Get("reason").(string)

	method := strings.ToUpper(d.Get("create_method").(string))
	idOverride := strings.TrimSpace(d.Get("id_override").(string))

	if method == "SKIP" {
		d.SetId(idOverride)
		return resourceDiscordAPIResourceRead(ctx, d, m)
	}

	path := d.Get("create_path").(string)
	if strings.Contains(path, "{id}") {
		path = substituteID(path, idOverride)
	}
	body, err := parseJSONBody(d.Get("create_body_json").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var out interface{}
	if err := c.DoJSONWithReason(ctx, method, path, nil, body, &out, reason); err != nil {
		return diag.FromErr(err)
	}

	if idOverride != "" {
		d.SetId(idOverride)
		return resourceDiscordAPIResourceRead(ctx, d, m)
	}

	// If response is an object and has id_field, use it.
	idField := d.Get("id_field").(string)
	if mm, ok := out.(map[string]interface{}); ok {
		if idv, ok := mm[idField]; ok {
			if ids, ok := idv.(string); ok && ids != "" {
				d.SetId(ids)
				return resourceDiscordAPIResourceRead(ctx, d, m)
			}
		}
	}

	// Fallback: use hash of create path + normalized response.
	b, _ := json.Marshal(out)
	norm, _ := normalizeJSON(string(b))
	d.SetId(fmt.Sprintf("%d", Hashcode(path+"|"+norm)))
	return resourceDiscordAPIResourceRead(ctx, d, m)
}

func resourceDiscordAPIResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	path := substituteID(d.Get("read_path").(string), d.Id())
	q, err := parseQueryJSON(d.Get("read_query_json").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var out interface{}
	if err := c.DoJSON(ctx, "GET", path, q, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
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
	_ = d.Set("response_json", norm)
	return nil
}

func resourceDiscordAPIResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	reason := d.Get("reason").(string)

	method := strings.ToUpper(d.Get("update_method").(string))
	if method == "SKIP" {
		return resourceDiscordAPIResourceRead(ctx, d, m)
	}
	pathT := d.Get("update_path").(string)
	if strings.TrimSpace(pathT) == "" {
		pathT = d.Get("read_path").(string)
	}
	path := substituteID(pathT, d.Id())

	body, err := parseJSONBody(d.Get("update_body_json").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	if body == nil {
		// nothing to do
		return resourceDiscordAPIResourceRead(ctx, d, m)
	}

	if err := c.DoJSONWithReason(ctx, method, path, nil, body, nil, reason); err != nil {
		return diag.FromErr(err)
	}
	return resourceDiscordAPIResourceRead(ctx, d, m)
}

func resourceDiscordAPIResourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	reason := d.Get("reason").(string)

	method := strings.ToUpper(d.Get("delete_method").(string))
	if method == "SKIP" {
		return nil
	}
	pathT := d.Get("delete_path").(string)
	if strings.TrimSpace(pathT) == "" {
		pathT = d.Get("read_path").(string)
	}
	path := substituteID(pathT, d.Id())

	body, err := parseJSONBody(d.Get("delete_body_json").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DoJSONWithReason(ctx, method, path, nil, body, nil, reason); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
