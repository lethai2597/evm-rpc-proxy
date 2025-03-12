package status

import (
	"fmt"
	"strings"
)

type Color string

const (
	LRed    Color = "#ffdddd"
	LGreen        = "#ddffdd"
	LYellow       = "#ffffdd"
	LGray         = "#aaaaaa"
	LOrange       = "#ffa500"

	Red    Color = "#dd4444"
	Green        = "#449944"
	Yellow       = "#dddd44"
	Gray         = "#666666"
	Orange       = "#cc6a00"
	Purple       = "#9966cc"
	Blue         = "#4444dd"
)

type Status struct {
	header string
	color  string

	icon       string
	icon_color string

	badge       []string
	badge_color []string
	badge_info  []string

	content string
}

func Create(is_paused, is_throttled, is_unhealthy bool) (*Status, string) {
	ret := &Status{}
	ret.badge = make([]string, 0, 10)
	ret.badge_color = make([]string, 0, 10)
	ret.badge_info = make([]string, 0, 10)

	status_description := ""
	if is_paused {
		ret.color = string(LRed)
		ret.icon = "⏸"
		ret.icon_color = string(Red)
		status_description = "Node is paused"
	} else if is_throttled {
		ret.color = string(LYellow)
		ret.icon = "⏱"
		ret.icon_color = string(Orange)
		status_description = "Node is throttled"
	} else if is_unhealthy {
		ret.color = string(LRed)
		ret.icon = "⚠"
		ret.icon_color = string(Red)
		status_description = "Node is unhealthy"
	} else {
		ret.color = string(LGreen)
		ret.icon = "✓"
		ret.icon_color = string(Green)
		status_description = "Node is healthy"
	}

	return ret, status_description
}

func (this *Status) SetHeader(content string) {
	this.header = content
}

func (this *Status) AddContent(content string) {
	this.content += content
}

func (this *Status) AddBadge(text string, color Color, info string) {
	this.badge = append(this.badge, text)
	this.badge_color = append(this.badge_color, string(color))
	this.badge_info = append(this.badge_info, info)
}

func (this *Status) GetHTML() string {
	ret := ""
	ret += fmt.Sprintf("<div style='background-color: %s; padding: 5px; margin-bottom: 5px; border-radius: 5px'>", this.color)
	ret += fmt.Sprintf("<div style='margin-bottom: 5px'><span style='color: %s; font-size: 1.5em'>%s</span> %s</div>", this.icon_color, this.icon, this.header)

	if len(this.badge) > 0 {
		ret += "<div style='margin-bottom: 5px'>"
		for i, badge := range this.badge {
			info := ""
			if len(this.badge_info[i]) > 0 {
				info = fmt.Sprintf("title='%s'", strings.Replace(this.badge_info[i], "'", "&#39;", -1))
			}
			ret += fmt.Sprintf("<span style='background-color: %s; color: white; padding: 2px 5px; margin-right: 5px; border-radius: 3px' %s>%s</span>", this.badge_color[i], info, badge)
		}
		ret += "</div>"
	}

	if len(this.content) > 0 {
		ret += fmt.Sprintf("<div>%s</div>", this.content)
	}

	ret += "</div>"
	return ret
}
