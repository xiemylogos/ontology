package main

import (
	"encoding/json"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/common/password"
	"github.com/urfave/cli"
	"io/ioutil"
)

func GetAccountByPassword(ctx *cli.Context, path string) (*account.Account, bool) {
	wallet, err := account.Open(path)
	if err != nil {
		log.Error("open wallet error:", err)
		return nil, false
	}
	pwd, err := password.GetPassword()
	if err != nil {
		log.Error("getPassword error:", err)
		return nil, false
	}
	user, err := wallet.GetDefaultAccount(pwd)
	if err != nil {
		log.Error("getDefaultAccount error:", err)
		return nil, false
	}
	return user, true
}

type ConfigParam struct {
	Path []string
}

func LoadAccount(ctx *cli.Context) ([]*account.Account, error) {
	data, err := ioutil.ReadFile("./wallet_config.json")
	if err != nil {
		log.Errorf("ioutil.ReadFile failed ", err)
		return nil, err
	}
	configParam := new(ConfigParam)
	err = json.Unmarshal(data, configParam)
	if err != nil {
		log.Error("json.Unmarshal failed ", err)
		return nil, err
	}
	var accs []*account.Account
	for _, path := range configParam.Path {
		user, ok := GetAccountByPassword(ctx, path)
		if !ok {
			return nil, err
		}
		accs = append(accs, user)
	}
	return accs, nil
}
