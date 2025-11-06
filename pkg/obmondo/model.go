package api

type InstallScriptFailureInput struct {
	Certname    string
	VerifyToken bool
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

type ServiceWindow struct {
	IsWindowOpen bool   `json:"is_window_open"`
	WindowType   string `json:"window_type"`
	Timezone     string `json:"timezone"`
}
