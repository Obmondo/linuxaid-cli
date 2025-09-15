package api

type InstallScriptFailureInput struct {
	Certname string
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
