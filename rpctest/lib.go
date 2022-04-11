package rpctest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/assert"
)

// request rpc
// compare result
//   order both config result and response result by their fields
//   json marshal then amd compare
func DoClientTest(t *testing.T, config RpcTestConfig) {

	rpc2Func, rpc2FuncSelector, rpc2FuncResultHandler := config.Rpc2Func, config.Rpc2FuncSelector, config.Rpc2FuncResultHandler
	ignoreRpc, ignoreExamples, onlyTestRpc := config.IgnoreRpcs, config.IgnoreExamples, config.OnlyTestRpcs

	// read json config
	httpClient := &http.Client{}
	resp, err := httpClient.Get(config.ExamplesUrl)
	if err != nil {
		t.Fatal(err)
	}
	source := resp.Body
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(source)
	if err != nil {
		t.Fatal(err)
	}

	m := &MockRPC{}
	err = json.Unmarshal(b, m)
	if err != nil {
		t.Fatal(err)
	}

	for rpcName, subExamps := range m.Examples {
		if ignoreRpc[rpcName] {
			continue
		}

		if len(onlyTestRpc) > 0 && !onlyTestRpc[rpcName] {
			continue
		}

		for _, subExamp := range subExamps {

			if ignoreExamples[subExamp.Name] {
				continue
			}

			var sdkFunc string
			var params []interface{}

			if _sdkFunc, ok := rpc2Func[rpcName]; ok {
				sdkFunc, params = _sdkFunc, subExamp.Params
			}

			if sdkFuncSelector, ok := rpc2FuncSelector[rpcName]; ok {
				sdkFunc, params = sdkFuncSelector(subExamp.Params)
			}

			if sdkFunc == "" {
				t.Fatalf("no sdk func for rpc:%s", rpcName)
			}

			fmt.Printf("\n========== example: %v === rpc: %s === params: %s ==========\n", subExamp.Name, rpcName, mustJsonMarshalForTest(params))
			// reflect call sdkFunc
			rpcReuslt, rpcError, err := reflectCall(config.Client, sdkFunc, params)
			if err != nil {
				var tmp interface{} = err
				switch tmp.(type) {
				case ConvertParamError:
					if subExamp.Error != nil {
						continue
					}
				}
				t.Fatal(err)
			}

			if sdkFuncResultHandler, ok := rpc2FuncResultHandler[rpcName]; ok {
				rpcReuslt = sdkFuncResultHandler(rpcReuslt)
			}

			if subExamp.Error != nil || rpcError != nil {
				assert.Equal(t, mustJsonMarshalForTest(subExamp.Error), mustJsonMarshalForTest(rpcError))
				continue
			}
			assert.Equal(t, mustJsonMarshalForTest(subExamp.Result), mustJsonMarshalForTest(rpcReuslt))
		}
	}
}

func reflectCall(c interface{}, sdkFunc string, params []interface{}) (resp interface{}, respError interface{}, err error) {
	typeOfClient := reflect.TypeOf(c)
	if method, ok := typeOfClient.MethodByName(sdkFunc); ok {
		in := make([]reflect.Value, len(params)+1)
		in[0] = reflect.ValueOf(c)
		// params marshal/unmarshal -> func params type
		for i, param := range params {
			// unmarshal params
			pType := method.Type.In(i + 1)

			// get element type if is variadic function for last param
			if method.Type.IsVariadic() && i == method.Type.NumIn()-2 {
				pType = pType.Elem()
			}

			vPtr := reflect.New(pType).Interface()
			vPtr, err = convertType(param, vPtr)
			if err != nil {
				return nil, nil, ConvertParamError(err)
			}
			v := reflect.ValueOf(vPtr).Elem().Interface()
			in[i+1] = reflect.ValueOf(v)
		}
		out := method.Func.Call(in)
		fmt.Printf("func name: %v, \nparams: %v, \nresp type: %T, respError type: %T, \nresp value: %v, \nrespError value: %v\n",
			sdkFunc,
			mustJsonMarshalForTest(getReflectValuesInterfaces(in[1:])),
			out[0].Interface(),
			out[1].Interface(),
			mustJsonMarshalForTest(out[0].Interface(), true),
			mustJsonMarshalForTest(out[1].Interface(), true),
		)
		return out[0].Interface(), out[1].Interface(), nil
	}
	return nil, nil, errors.Errorf("not found method %v", sdkFunc)
}

func getReflectValuesInterfaces(values []reflect.Value) []interface{} {
	var result []interface{}
	for _, v := range values {
		result = append(result, v.Interface())
	}
	return result
}

// cfx_getBlockByEpochNumber  GetBlockSummaryByEpoch 0x0, false
// rpc_name => func(params) sdkFuncName sdkFuncParams
func convertType(from interface{}, to interface{}) (interface{}, error) {
	jp, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jp, &to)
	if err != nil {
		return nil, err
	}
	return to, nil
}

func mustConvertType(from interface{}, to interface{}) interface{} {
	v, err := convertType(from, to)
	if err != nil {
		panic(err)
	}
	return v
}

func mustJsonMarshalForTest(v interface{}, indent ...bool) string {
	j, err := jsonMarshalForTest(v, indent...)
	if err != nil {
		panic(err)
	}
	return string(j)
}

// Block
// 	BlockHash
// 	[]Transactions
// 		Creates 'testomit:false'

// handle struct field by 'testomit' tag and order json
func jsonMarshalForTest(v interface{}, indent ...bool) ([]byte, error) {

	fmt.Printf("reflect.ValueOf(v).Kind(): %v\n", reflect.ValueOf(v).Kind())

	// reflectV := reflect.ValueOf(v)

	// if reflectV.Kind() != reflect.Ptr && reflectV.Kind() != reflect.Struct {
	// 	return json.Marshal(v)
	// }

	// if reflectV.Kind() == reflect.Ptr && reflectV.Elem().Kind() != reflect.Struct {
	// 	fmt.Printf("reflect.ValueOf(v).Elem().Kind(): %v\n", reflectV.Elem().Kind())
	// 	return json.Marshal(v)
	// }

	if isSelfOrElemBeStruct(v) {
		return json.Marshal(v)
	}

	// b, err := json.Marshal(v)
	// if err != nil {
	// 	return nil, err
	// }
	// m := map[string]interface{}{}

	// err = json.Unmarshal(b, &m)
	// if err != nil {
	// 	return nil, err
	// }

	m := mustConvertType(v, map[string]interface{}{}).(map[string]interface{})

	// reflectV := reflect.ValueOf(v)
	// t := reflect.TypeOf(v)
	// if reflectV.Kind() == reflect.Ptr {
	// 	t = t.Elem()
	// }
	t := getCoreType(v)

	m = setTestOmit(m, t).(map[string]interface{})
	// for i := 0; i < t.NumField(); i++ {
	// 	tf := t.Field(i)
	// 	isOmit, ok := tf.Tag.Lookup("testomit")
	// 	fName := tf.Name
	// 	if jsonTag, ok := tf.Tag.Lookup("json"); ok {
	// 		fName, _ = paserJsonTag(jsonTag)
	// 	}

	// 	// m[fName], err = jsonMarshalForTest(m[fName], indent...)
	// 	// fmt.Printf("m[%v]: %v\n", fName, m[fName])
	// 	// if err != nil {
	// 	// 	return nil, err
	// 	// }

	// 	if !ok {
	// 		continue
	// 	}

	// 	if m[fName] != nil {
	// 		continue
	// 	}

	// 	if isOmit == "true" {
	// 		delete(m, fName)
	// 		continue
	// 	}

	// 	m[fName] = nil
	// }

	if isIndent(indent...) {
		return json.MarshalIndent(m, "", "  ")
	} else {
		return json.Marshal(m)
	}
}

func setTestOmit(v interface{}, t reflect.Type) interface{} {
	switch v.(type) {
	case map[string]interface{}:
		break
	case []interface{}:
		raw := v.([]interface{})
		for i, vv := range raw {
			raw[i] = setTestOmit(vv, t.Elem())
		}
		return raw
	default:
		return v
	}

	m := v.(map[string]interface{})

	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)
		fName := tf.Name
		if jsonTag, ok := tf.Tag.Lookup("json"); ok {
			fName, _ = parseJsonTag(jsonTag)
		}

		// 不为空则向下递归, 字段类型为map, array 则递归
		if m[fName] != nil {
			m[fName] = setTestOmit(m[fName], tf.Type)
			continue
		}

		isOmit, ok := tf.Tag.Lookup("testomit")
		if !ok {
			continue
		}

		if isOmit == "true" {
			delete(m, fName)
			continue
		}

		m[fName] = nil
	}
	return m
}

func isSelfOrElemBeStruct(v interface{}) bool {
	reflectV := reflect.ValueOf(v)

	if reflectV.Kind() != reflect.Ptr && reflectV.Kind() != reflect.Struct {
		return true
	}

	if reflectV.Kind() == reflect.Ptr && reflectV.Elem().Kind() != reflect.Struct {
		fmt.Printf("reflect.ValueOf(v).Elem().Kind(): %v\n", reflectV.Elem().Kind())
		return true
	}
	return false
}

// Get type of self or elem type if pointer
func getCoreType(v interface{}) reflect.Type {
	reflectV := reflect.ValueOf(v)
	t := reflect.TypeOf(v)
	if reflectV.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func parseJsonTag(jsonTag string) (jsonName string, isOmitEmpty bool) {
	splits := strings.Split(jsonTag, ",")
	if len(splits) == 1 {
		return splits[0], false
	}
	return splits[0], strings.Contains(splits[1], "omitempty")
}

func isIndent(indent ...bool) bool {
	_isIndent := false
	if len(indent) > 0 {
		_isIndent = indent[0]
	}
	return _isIndent
}
