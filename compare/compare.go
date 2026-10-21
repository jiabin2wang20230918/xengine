// Package compare 提供了对比的服务，基于 github.com/antonmedv/expr 库对属性值进行符号上的比对。
//
// 支持的符号参考 https://docs.qq.com/sheet/DQVJGSGFGT3FVc0N2?tab=18qnzv
package compare

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/jiabin2wang20230918/xengine/utils/log"
)

// PropertyMatch 属性对比
// 将传入的属性值和目标属性值进行符号比对，返回是否满足。
// 任何异常时，都会返回false
func PropertyMatch(propertyValue interface{}, targetPropertyValue interface{}, symbol string) bool {
	env := map[string]interface{}{}
	input := ""
	switch symbol {
	case "between":
		// 范围比较
		env["p"] = propertyValue
		//s, ok := targetPropertyValue.(string)
		arr, ok := targetPropertyValue.([]interface{})
		if !ok {
			log.Errorf("parse error, should be []interface")
			return false
		}
		if len(arr) != 2 {
			log.Errorf("between arr length should be 2")
			return false
		}
		env["small"] = arr[0]
		env["big"] = arr[1]
		input = fmt.Sprintf("p >= small && p <= big")
	case "in":
		// 范围比较
		env["p"] = propertyValue
		env["t"] = targetPropertyValue
		input = fmt.Sprintf("p in t")
	case "not in":
		// 范围比较
		env["p"] = propertyValue
		env["t"] = targetPropertyValue
		input = fmt.Sprintf("p not in t")
	case "==", "!=", ">=", "<=", ">", "<":
		env["p"] = propertyValue
		env["t"] = targetPropertyValue
		input = fmt.Sprintf("p %s t", symbol)
	case "like":
		env["p"] = propertyValue
		env["t"] = targetPropertyValue
		input = "p contains t"
	default:
		log.Errorf("unsupported")
		return false
	}
	log.Debugf("property value:%v symbol:%v target :%v", propertyValue, symbol, targetPropertyValue)
	eval, err := expr.Eval(input, env)
	if err != nil {
		log.Errorf("err:%v", err)
		return false
	}
	result, ok := eval.(bool)
	if ok {
		return result
	} else {
		return false
	}
}
