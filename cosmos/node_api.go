/*
 * Copyright 2018 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package cosmos

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/blocktree/openwallet/v2/log"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
)

type ClientInterface interface {
	Call(path string, request []interface{}) (*gjson.Result, error)
}

// A Client is a Bitcoin RPC client. It performs RPCs over HTTP using JSON
// request and responses. A Client must be configured with a secret token
// to authenticate with other Cores on the network.
type Client struct {
	BaseURL     string
	AccessToken string
	Debug       bool
	client      *req.Req
	//Client *req.Req
}

type Response struct {
	Code    int         `json:"code,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Message string      `json:"message,omitempty"`
	Id      string      `json:"id,omitempty"`
}

func NewClient(url string, debug bool) *Client {
	c := Client{
		BaseURL: url,
		//AccessToken: token,
		Debug: debug,
	}

	api := req.New()
	//trans, _ := api.Client().Transport.(*http.Transport)
	//trans.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	c.client = api

	return &c
}

// Call calls a remote procedure on another node, specified by the path.
func (c *Client) Call(path string, request interface{}, method string) (*gjson.Result, error) {

	if c.client == nil {
		return nil, errors.New("API url is not setup. ")
	}

	if c.Debug {
		log.Std.Debug("Start Request API...")
	}

	url := c.BaseURL + path

	r, err := c.client.Do(method, url, request)

	if c.Debug {
		log.Std.Debug("Request API Completed")
	}

	if c.Debug {
		log.Std.Debug("%+v", r)
	}

	err = c.isError(r)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())

	return &resp, nil
}

func (b *Client) isError(resp *req.Resp) error {

	if resp == nil || resp.Response() == nil {
		return errors.New("Response is empty! ")
	}

	if resp.Response().StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.Response().StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.String())
	}

	return nil
}

// See 2 (end of page 4) http://www.ietf.org/rfc/rfc2617.txt
// "To receive authorization, the client sends the userid and password,
// separated by a single colon (":") character, within a base64
// encoded string in the credentials."
// It is not meant to be urlencoded.
func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

//isError 是否报错
func isError(result *gjson.Result) error {
	var (
		err error
	)

	/*
		 //failed 返回错误
		 {
			 "result": null,
			 "error": {
				 "code": -8,
				 "message": "Block height out of range"
			 },
			 "id": "foo"
		 }
	*/

	if !result.Get("error").IsObject() {

		if !result.Get("result").Exists() {
			return errors.New("Response is empty! ")
		}

		return nil
	}

	errInfo := fmt.Sprintf("[%d]%s",
		result.Get("error.code").Int(),
		result.Get("error.message").String())
	err = errors.New(errInfo)

	return err
}

// 获取当前区块高度
func (c *Client) getBlockHeight() (uint64, error) {
	resp, err := c.Call("/blocks/latest", nil, "GET")

	if err != nil {
		return 0, err
	}

	return resp.Get("block_meta").Get("header").Get("height").Uint(), nil
}

// 通过高度获取区块哈希
func (c *Client) getBlockHash(height uint64) (string, error) {

	path := fmt.Sprintf("/blocks/%d", height)

	resp, err := c.Call(path, nil, "GET")

	if err != nil {
		return "", err
	}

	return resp.Get("block_meta").Get("block_id").Get("hash").String(), nil
}

func (c *Client) getAccountNumberAndSequence(address string) (int, int, error) {

	path := "/auth/accounts/" + address
	r, err := c.Call(path, nil, "GET")
	if err != nil {
		return 0, 0, errors.New("Failed to get address' account number and sequence!")
	}
	accountNumber := int(r.Get("result").Get("value").Get("account_number").Uint())
	sequence := int(r.Get("result").Get("value").Get("sequence").Uint())
	if accountNumber == 0 {
		return 0, 0, errors.New("Failed to get account number, or node sync is stoped!")
	}

	return accountNumber, sequence, nil
}

// 获取地址余额
func (c *Client) getBalance(address string, denom string) (*AddrBalance, error) {
	path := "/bank/balances/" + address

	r, err := c.Call(path, nil, "GET")

	if err != nil {
		return nil, err
	}

	if r.Raw == "" {
		return &AddrBalance{Address: address, Balance: big.NewInt(0)}, nil
	}

	coins := r.Get("result").Array()

	for _, coin := range coins {
		if coin.Get("denom").String() == denom {
			return &AddrBalance{Address: address, Balance: big.NewInt(coin.Get("amount").Int())}, nil
		}
	}

	return &AddrBalance{Address: address, Balance: big.NewInt(0)}, nil
}

// 获取区块信息
func (c *Client) getBlock(hash string) (*Block, error) {
	path := "blocks/signature/" + hash

	resp, err := c.Call(path, nil, "GET")

	if err != nil {
		return nil, err
	}

	return NewBlock(resp), nil
}

func (c *Client) getBlockByHeight(height uint64) (*Block, error) {
	path := fmt.Sprintf("/blocks/%d", height)

	resp, err := c.Call(path, nil, "GET")

	if err != nil {
		return nil, err
	}

	return NewBlock(resp), nil
}

func (c *Client) sendTransaction(jsonStr string) (string, error) {

	// sequence := gjson.Get(jsonStr, "tx").Get("signatures").Array()[0].Get("sequence").Uint()
	// from := gjson.Get(jsonStr, "tx").Get("msg").Array()[0].Get("value").Get("from_address").String()
	path := "/txs"

	var dat map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &dat)

	resp, err := c.Call(path, req.BodyJSON(&dat), "POST")
	if err != nil {
		return "", err
	}
	if resp.Get("code").Uint() != 0 && resp.Get("raw_log").String() != "" {
		return "", errors.New("send transaction failed with error:" + resp.Get("raw_log").String())
	}

	// timeCount := 1
	// for {
	// 	time.Sleep(time.Second * 5)
	// 	_, sequenceNew, _ := c.getAccountNumberAndSequence(from)
	// 	if sequenceNew > int(sequence) {
	// 		break
	// 	} else {
	// 		timeCount++
	// 		if timeCount == 120 {
	// 			return "", errors.New("Submit transaction failed!")
	// 		}
	// 	}
	// }
	return resp.Get("txhash").String(), nil
}
