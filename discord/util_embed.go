package discord

import (
	"encoding/json"
)

// restEmbed matches Discord's embed object (subset used by this provider).
// It is intentionally provider-local so we do not depend on external Discord client libraries.
type restEmbed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	Color       int    `json:"color,omitempty"`

	Footer    *restEmbedFooter    `json:"footer,omitempty"`
	Image     *restEmbedImage     `json:"image,omitempty"`
	Thumbnail *restEmbedThumbnail `json:"thumbnail,omitempty"`
	Video     *restEmbedVideo     `json:"video,omitempty"`
	Provider  *restEmbedProvider  `json:"provider,omitempty"`
	Author    *restEmbedAuthor    `json:"author,omitempty"`
	Fields    []restEmbedField    `json:"fields,omitempty"`
}

type restEmbedFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type restEmbedImage struct {
	URL    string `json:"url,omitempty"`
	Height int    `json:"height,omitempty"`
	Width  int    `json:"width,omitempty"`
	// proxy_url is returned by Discord; it is computed in the schema.
	ProxyURL string `json:"proxy_url,omitempty"`
}

type restEmbedThumbnail struct {
	URL      string `json:"url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
}

type restEmbedVideo struct {
	URL    string `json:"url,omitempty"`
	Height int    `json:"height,omitempty"`
	Width  int    `json:"width,omitempty"`
}

type restEmbedProvider struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type restEmbedAuthor struct {
	Name         string `json:"name,omitempty"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

type restEmbedField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

func buildEmbed(embedList []interface{}) (*restEmbed, error) {
	embedMap := embedList[0].(map[string]interface{})
	e := &restEmbed{
		Title:       embedMap["title"].(string),
		Description: embedMap["description"].(string),
		URL:         embedMap["url"].(string),
		Timestamp:   embedMap["timestamp"].(string),
		Color:       embedMap["color"].(int),
	}

	if len(embedMap["footer"].([]interface{})) > 0 {
		footerMap := embedMap["footer"].([]interface{})[0].(map[string]interface{})
		e.Footer = &restEmbedFooter{
			Text:    footerMap["text"].(string),
			IconURL: footerMap["icon_url"].(string),
		}
	}

	if len(embedMap["image"].([]interface{})) > 0 {
		imageMap := embedMap["image"].([]interface{})[0].(map[string]interface{})
		e.Image = &restEmbedImage{
			URL:    imageMap["url"].(string),
			Width:  imageMap["width"].(int),
			Height: imageMap["height"].(int),
		}
	}

	if len(embedMap["thumbnail"].([]interface{})) > 0 {
		thumbnailMap := embedMap["thumbnail"].([]interface{})[0].(map[string]interface{})
		e.Thumbnail = &restEmbedThumbnail{
			URL:    thumbnailMap["url"].(string),
			Width:  thumbnailMap["width"].(int),
			Height: thumbnailMap["height"].(int),
		}
	}

	if len(embedMap["video"].([]interface{})) > 0 {
		videoMap := embedMap["video"].([]interface{})[0].(map[string]interface{})
		e.Video = &restEmbedVideo{
			URL:    videoMap["url"].(string),
			Width:  videoMap["width"].(int),
			Height: videoMap["height"].(int),
		}
	}

	if len(embedMap["provider"].([]interface{})) > 0 {
		providerMap := embedMap["provider"].([]interface{})[0].(map[string]interface{})
		e.Provider = &restEmbedProvider{
			URL:  providerMap["url"].(string),
			Name: providerMap["name"].(string),
		}
	}

	if len(embedMap["author"].([]interface{})) > 0 {
		authorMap := embedMap["author"].([]interface{})[0].(map[string]interface{})
		e.Author = &restEmbedAuthor{
			Name:    authorMap["name"].(string),
			URL:     authorMap["url"].(string),
			IconURL: authorMap["icon_url"].(string),
		}
	}

	for _, field := range embedMap["fields"].([]interface{}) {
		fieldMap := field.(map[string]interface{})
		e.Fields = append(e.Fields, restEmbedField{
			Name:   fieldMap["name"].(string),
			Value:  fieldMap["value"].(string),
			Inline: fieldMap["inline"].(bool),
		})
	}

	return e, nil
}

func unbuildEmbed(embed *restEmbed) []interface{} {
	var ret interface{}

	// Marshal/unmarshal to drop zero-values and match the schema's map/list representation.
	j, _ := json.MarshalIndent(embed, "", "    ")
	_ = json.Unmarshal(j, &ret)
	return []interface{}{ret}
}
