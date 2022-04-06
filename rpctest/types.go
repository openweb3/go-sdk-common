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
	Rpc2FuncSelector map[string]func(params []interface{}) (string, []interface{})
	IgnoreRpcs       map[string]bool
	OnlyTestRpcs     map[string]bool
}
