package security

type CommandErrResponse struct {
	CmdOutput string `json:"output"`
	Error     error  `json:"error"`
}

type TotalNumberOfPackagesWithUpdateResponse struct {
	TotalNumberOfPackagesWithUpdate int  `json:"total_number_of_packages_with_update"`
	HasKernelUpdate                 bool `json:"has_kernel_update"`
}
