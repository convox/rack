package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/convox/logger"
)

var (
	skipParams     = []string{}
	redactedParams = strings.Join(skipParams, ",")
	fileName       = "telemetry-sync.json"
)

func (p *Provider) RackParamsToSync(params map[string]string) map[string]string {
	var log = logger.New("ns=workers.heartbeat")

	// check if telemetry sync file exists in settings s3 bucket
	exists, err := p.SettingExists(fileName)
	if err != nil {
		log.Error(err)
		return map[string]string{}
	}

	// creates if it doesn't exist yet
	if !exists {
		err = p.createNewSyncFile(params)
		if err != nil {
			log.Error(err)
			return map[string]string{}
		}
	}

	// get telemetry sync content
	fileContent, err := p.SettingGet(fileName)
	if err != nil {
		log.Error(err)
		return map[string]string{}
	}

	var paramsMap map[string]interface{}
	err = json.Unmarshal([]byte(fileContent), &paramsMap)
	if err != nil {
		log.Error(err)
		return map[string]string{}
	}

	// check which params are not sync yet
	var nSync []string
	for k, v := range paramsMap {
		if !v.(bool) {
			nSync = append(nSync, k)
		}
	}

	// create map of params that will be sync to segment
	toSync := make(map[string]string)
	for _, s := range nSync {
		if val, ok := params[s]; ok {
			if strings.Contains(redactedParams, s) {
				toSync[s] = hashParamValue(val)
			} else {
				toSync[s] = val
			}
		}
	}

	return toSync
}

func (p *Provider) createNewSyncFile(params map[string]string) error {
	c := make(map[string]interface{})
	for k := range params {
		c[k] = false
	}

	nc, err := json.Marshal(c)
	if err != nil {
		return err
	}

	if err := p.SettingPut(fileName, string(nc)); err != nil {
		return err
	}

	return nil
}

func (p *Provider) UpdateSyncFile() error {
	// mark all params as sync
	fileContent, err := p.SettingGet(fileName)
	if err != nil {
		return err
	}

	var paramsMap map[string]interface{}
	err = json.Unmarshal([]byte(fileContent), &paramsMap)
	if err != nil {
		return err
	}

	for k := range paramsMap {
		paramsMap[k] = true
	}

	nc, err := json.Marshal(paramsMap)
	if err != nil {
		return err
	}

	if err := p.SettingPut(fileName, string(nc)); err != nil {
		return err
	}

	return nil
}

func hashParamValue(value string) string {
	hasher := sha256.New()
	hasher.Write([]byte(value))
	return hex.EncodeToString(hasher.Sum(nil))
}
