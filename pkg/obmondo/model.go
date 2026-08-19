package api

import (
	"strings"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
)

type InstallScriptInput struct {
	Certname string
	Token    string
}

type UpdateScriptFailureInput struct {
	Certname string
}

type PuppetLastRunReport struct {
	Time                        string `yaml:"time" json:"time"`
	Status                      string `yaml:"status" json:"status"`
	TransactionCompleted        bool   `yaml:"transaction_completed" json:"transaction_completed"`
	IsLastRunYamlFileNotPresent bool   `yaml:"-" json:"is_last_run_yaml_file_not_present"`
}

type ObmondoAPIResponse[T any] struct {
	Status     int    `json:"status"`
	Success    bool   `json:"success"`
	Data       T      `json:"data"`
	Message    string `json:"message"`
	Resolution string `json:"resolution"`
	ErrorText  string `json:"error_text"`
}

type LinuxWindowDetails struct {
	LinuxAidTag string `json:"linuxaid_tag,omitempty"`
	NeedsReboot bool   `json:"needs_reboot"`
}

type ServiceWindow struct {
	IsWindowOpen             bool                `json:"is_window_open"`
	DoesWindowExist          bool                `json:"does_window_exist"`
	WindowType               string              `json:"window_type"`
	Timezone                 string              `json:"timezone"`
	NextWindowLocalStartTime string              `json:"next_window_local_start_time,omitempty"`
	Linux                    *LinuxWindowDetails `json:"linux,omitempty"`
}

// PuppetEnvironment returns the environment the run inside this window has to use. An automatic
// window pins the linuxaid tag its whole update cycle runs with, so every node of the cycle
// updates with the same release instead of whatever tag happens to be the latest when its own
// window opens. Windows with no pinned tag - adhoc bookings, and automatic ones the API could not
// resolve a tag for - return an empty string, leaving the caller to resolve the environment itself.
func (s *ServiceWindow) PuppetEnvironment() string {
	if s == nil || s.WindowType != constant.ServiceWindowTypeAutomatic || s.Linux == nil {
		return ""
	}

	return sanitizePuppetEnvironment(s.Linux.LinuxAidTag)
}

// sanitizePuppetEnvironment maps a linuxaid tag to the environment puppet deploys it as: puppet
// only accepts alphanumerics and underscores in an environment name, so every other character
// becomes an underscore. This mirrors what Obmondo does when it resolves an environment itself,
// which is why the tag from the service window needs no round trip to the API.
func sanitizePuppetEnvironment(tag string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			return r
		default:
			return '_'
		}
	}, tag)
}

type LinuxAidSettings struct {
	PrometheusURL   string `json:"prometheus_url"`
	PuppetserverURL string `json:"puppetserver_url"`
}

type CustomerSettings struct {
	CustomerID string            `json:"customer_id"`
	LinuxAid   *LinuxAidSettings `json:"linuxaid,omitempty"`
}

// ServerEnvironment is the puppet environment the API resolved for a certname: the pinned
// override when one exists, otherwise the latest linuxaid tag.
type ServerEnvironment struct {
	Certname    string `json:"certname"`
	Environment string `json:"environment"`
}
