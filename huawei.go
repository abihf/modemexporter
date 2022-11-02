package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Huawei struct {
	URL    string
	cookie string
	client http.Client
}

var _ Modem = &Huawei{}

// Auth implements Modem
func (h *Huawei) Auth(ctx context.Context, user string, password string) (*time.Time, error) {
	token, err := h.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't get token: %w", err)
	}

	var param = url.Values{}
	param.Set("UserName", user)
	param.Set("PassWord", base64.URLEncoding.EncodeToString([]byte(password)))
	param.Set("x.X_HW_Token", token)
	request, err := http.NewRequestWithContext(ctx, "POST", h.URL+"/login.cgi", bytes.NewBufferString(param.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("cookie", "Cookie=body:Language:english:id=-1")

	res, err := h.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("can't access login page: %w", err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code %d", res.StatusCode)
	}
	cookies := []string{}
	for _, cookie := range res.Cookies() {
		cookies = append(cookies, cookie.Name+"="+cookie.Value)
	}
	h.cookie = strings.Join(cookies, "; ")
	expire := time.Now().Add(1 * time.Minute)
	return &expire, nil
}

func (h *Huawei) getToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", h.URL, nil)
	if err != nil {
		return "", err
	}
	res, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("can't connect to modem homepage: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("invalid status code %d", res.StatusCode)
	}

	scanner := bufio.NewScanner(res.Body)
	re := regexp.MustCompile(`return '([^']+)`)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "function GetRandCnt()") {
			continue
		}
		match := re.FindStringSubmatch(line)
		if len(match) > 1 {
			return match[1], nil
		}
	}
	return "", fmt.Errorf("invalid body response")
}

// Stat implements Modem
func (h *Huawei) Stat(ctx context.Context) (*Stat, error) {
	line, err := h.readStat(ctx)
	if err != nil {
		return nil, err
	}
	stat := Stat{}
	re := regexp.MustCompile(`LANStats\("([^"]+)","([^"]+)","([^"]+)","([^"]+)","([^"]+)"`)
	matches := re.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		stat.TxPackets += Must(strconv.ParseUint(match[2], 10, 64))
		stat.TxBytes += Must(strconv.ParseUint(match[3], 10, 64))
		stat.RxPackets += Must(strconv.ParseUint(match[4], 10, 64))
		stat.RxBytes += Must(strconv.ParseUint(match[5], 10, 64))
	}
	return &stat, nil
}

func (h *Huawei) readStat(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", h.URL+"/html/amp/ethinfo/ethinfo.asp", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("cookie", h.cookie)
	res, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("can't connect to modem stat page: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("invalid status code %d", res.StatusCode)
	}

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "var userEthInfos") {
			continue
		}
		return line, nil

	}
	return "", fmt.Errorf("invalid body response")
}
