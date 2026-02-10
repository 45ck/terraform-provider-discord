package discord

import (
	"github.com/andersfylling/disgord"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
	"strings"
	"time"
)

func resourceDiscordMessage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMessageCreate,
		ReadContext:   resourceMessageRead,
		UpdateContext: resourceMessageUpdate,
		DeleteContext: resourceMessageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"server_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"author": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"content": {
				AtLeastOneOf: []string{"content", "embed"},
				Type:         schema.TypeString,
				Optional:     true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return old == strings.TrimSuffix(new, "\r\n")
				},
			},
			"timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"edited_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tts": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"embed": {
				AtLeastOneOf: []string{"content", "embed"},
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"title": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"url": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"timestamp": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"color": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"footer": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"text": {
										Type:     schema.TypeString,
										Required: true,
									},
									"icon_url": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"image": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:     schema.TypeString,
										Required: true,
									},
									"proxy_url": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"height": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"width": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
						"thumbnail": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:     schema.TypeString,
										Required: true,
									},
									"proxy_url": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"height": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"width": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
						"video": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:     schema.TypeString,
										Required: true,
									},
									"height": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"width": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
						"provider": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"url": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"author": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"url": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"icon_url": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"proxy_icon_url": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"fields": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"value": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"inline": {
										Type:     schema.TypeBool,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"pinned": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"type": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceMessageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channelId := getId(d.Get("channel_id").(string))
	params := &disgord.CreateMessageParams{
		Content: d.Get("content").(string),
		Tts:     d.Get("tts").(bool),
	}

	if v, ok := d.GetOk("embed"); ok {
		embed, err := buildEmbed(v.([]interface{}))
		if err != nil {
			return diag.Errorf("Failed to create message in %s: %s", channelId.String(), err.Error())
		}

		params.Embed = embed
	}

	message, err := client.CreateMessage(ctx, channelId, params)
	if err != nil {
		return diag.Errorf("Failed to create message in %s: %s", channelId.String(), err.Error())
	}

	d.SetId(message.ID.String())
	d.Set("type", int(message.Type))
	d.Set("timestamp", message.Timestamp.Format(time.RFC3339))
	d.Set("author", message.Author.ID.String())
	if len(message.Embeds) > 0 {
		d.Set("embed", unbuildEmbed(message.Embeds[0]))
	} else {
		d.Set("embed", nil)
	}
	if !message.GuildID.IsZero() {
		d.Set("server_id", message.GuildID.String())
	}

	if d.Get("pinned").(bool) {
		err = client.PinMessage(ctx, message)
		if err != nil {
			diags = append(diags, diag.Errorf("Failed to pin message %s in %s: %s", message.ID.String(), channelId.String(), err.Error())...)
		}
	}

	return diags
}

func resourceMessageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channelId := getId(d.Get("channel_id").(string))
	messageId := getId(d.Id())
	message, err := client.GetMessage(ctx, channelId, messageId)
	if err != nil {
		return diag.Errorf("Failed to fetch message %s in %s: %s", messageId.String(), channelId.String(), err.Error())
	}

	if !message.GuildID.IsZero() {
		d.Set("server_id", message.GuildID.String())
	}
	d.Set("type", int(message.Type))
	d.Set("tts", message.Tts)
	d.Set("timestamp", message.Timestamp.Format(time.RFC3339))
	d.Set("author", message.Author.ID.String())
	d.Set("content", message.Content)
	d.Set("pinned", message.Pinned)

	if len(message.Embeds) > 0 {
		d.Set("embed", unbuildEmbed(message.Embeds[0]))
	} else {
		d.Set("embed", nil)
	}
	if message.EditedTimestamp.IsZero() {
		d.Set("edited_timestamp", nil)
	} else {
		d.Set("edited_timestamp", message.EditedTimestamp.Format(time.RFC3339))
	}

	return diags
}

func resourceMessageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channelId := getId(d.Get("channel_id").(string))
	messageId := getId(d.Id())
	builder := client.UpdateMessage(ctx, channelId, messageId)
	anyEdit := false

	if d.HasChange("content") {
		builder.SetContent(d.Get("content").(string))
		anyEdit = true
	}
	if d.HasChange("embed") {
		var embed *disgord.Embed
		_, n := d.GetChange("embed")
		if len(n.([]interface{})) > 0 {
			e, err := buildEmbed(n.([]interface{}))
			if err != nil {
				return diag.Errorf("Failed to edit message %s in %s: %s", messageId.String(), channelId.String(), err.Error())
			}

			embed = e
		}

		builder.SetEmbed(embed)
		anyEdit = true
	}

	if anyEdit {
		message, err := builder.Execute()
		if err != nil {
			return diag.Errorf("Failed to update message %s in %s: %s", channelId.String(), messageId.String(), err.Error())
		}

		if len(message.Embeds) > 0 {
			d.Set("embed", unbuildEmbed(message.Embeds[0]))
		} else {
			d.Set("embed", nil)
		}

		if message.EditedTimestamp.IsZero() {
			d.Set("edited_timestamp", nil)
		} else {
			d.Set("edited_timestamp", message.EditedTimestamp.Format(time.RFC3339))
		}
	}

	if d.HasChange("pinned") {
		// Pin/unpin must be performed via dedicated endpoints.
		pinned := d.Get("pinned").(bool)
		msg := &disgord.Message{ID: messageId, ChannelID: channelId}
		if pinned {
			if err := client.PinMessage(ctx, msg); err != nil {
				diags = append(diags, diag.Errorf("Failed to pin message %s in %s: %s", messageId.String(), channelId.String(), err.Error())...)
			}
		} else {
			if err := client.UnpinMessage(ctx, msg); err != nil {
				diags = append(diags, diag.Errorf("Failed to unpin message %s in %s: %s", messageId.String(), channelId.String(), err.Error())...)
			}
		}
	}

	return diags
}

func resourceMessageDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channelId := getId(d.Get("channel_id").(string))
	messageId := getId(d.Id())
	err := client.DeleteMessage(ctx, channelId, messageId)
	if err != nil {
		return diag.Errorf("Failed to delete message %s in %s: %s", messageId.String(), channelId.String(), err.Error())
	}

	return diags
}
