package template

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type Input struct {
	TemplatePath  string
	ParameterPath string
}

type Output struct {
	Template   []byte
	Parameters []types.Parameter
}

func Read(input Input) (*Output, error) {
	template, err := os.ReadFile(input.TemplatePath)
	if err != nil {
		return nil, err
	}

	output := &Output{
		Template: template,
	}

	params, err := readParameters(input.ParameterPath)
	if err != nil {
		return nil, err
	}

	output.Parameters = params

	return output, nil
}

func readTemplate(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func readParameters(filePath string) ([]types.Parameter, error) {
	parameters := []types.Parameter{}
	rawParameters, err := os.ReadFile(filePath)
	if err != nil {
		return parameters, err
	}

	rawJson := make(map[string]interface{})
	err = json.Unmarshal(rawParameters, &rawJson)
	if err != nil {
		return parameters, err
	}

	flattenedParams := make(map[string]interface{})

	flattenNestedParams("", rawJson, flattenedParams)

	for key, value := range flattenedParams {
		p := types.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value.(string))}
		parameters = append(parameters, p)
	}

	return parameters, nil
}

func flattenNestedParams(prefix string, src map[string]interface{}, dest map[string]interface{}) {
	if len(prefix) > 0 {
		prefix += "."
	}
	for k, v := range src {
		switch child := v.(type) {
		case map[string]interface{}:
			flattenNestedParams(prefix+k, child, dest)
		case []interface{}:
			for i := 0; i < len(child); i++ {
				dest[prefix+k+"."+strconv.Itoa(i)] = child[i]
			}
		default:
			dest[prefix+k] = v
		}
	}
}
