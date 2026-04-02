package security

type CommandErrResponse struct {
	CmdOutput string `json:"output"`
	Error     error  `json:"error"`
}

type TotalNumberOfPackagesWithUpdateResponse struct {
	TotalNumberOfPackagesWithUpdate int  `json:"total_number_of_packages_with_update"`
	HasKernelUpdate                 bool `json:"has_kernel_update"`
}

type ScanResponse struct {
	Success               bool   `json:"success"`
	TotalCVEs             int    `json:"total_cves"`
	CriticalCVEs          int    `json:"critical_cves"`
	HighCVEs              int    `json:"high_cves"`
	MediumCVEs            int    `json:"medium_cves"`
	LowCVEs               int    `json:"low_cves"`
	PackagesWithUpdates   int    `json:"packages_with_updates"`
	KernelUpdateAvailable bool   `json:"kernel_update_available"`
	PreviousTotalCVEs     int    `json:"previous_total_cves"`
	CVEsFixed             int    `json:"cves_fixed"`
	CriticalCVEsFixed     int    `json:"critical_cves_fixed"`
	HighCVEsFixed         int    `json:"high_cves_fixed"`
	MediumCVEsFixed       int    `json:"medium_cves_fixed"`
	LowCVEsFixed          int    `json:"low_cves_fixed"`
	Error                 string `json:"error,omitempty"`
}
