package rpctest

type MockRPC struct {
	Version  string `json:"version"`
	Examples map[string][]struct {
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Params      []interface{} `json:"params"`
		Result      interface{}   `json:"result"`
		Error       interface{}   `json:"error"`
	} `json:"examples"`
}

type RpcTestConfig struct {
	ExamplesUrl string
	Client      interface{}

	Rpc2Func         map[string]string
	Rpc2FuncSelector map[string]func(params []interface{}) (realFuncName string, realParams []interface{})
	// convert sdk rpc result back to pre-unmarshal for comparing with example result, becasue sdk may change result type for user convinent, such as web3go
	Rpc2FuncResultHandler map[string]func(result interface{}) (handlerdResult interface{})
	// ignoreRpc priority is higher than onlyTestRpc
	IgnoreRpcs map[string]bool
	// onlyTestRpc priority is lower than ignoreRpc
	OnlyTestRpcs map[string]bool
}
