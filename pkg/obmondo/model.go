package api

type InstallScriptInput struct {
	Certname string
	Token    string
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
