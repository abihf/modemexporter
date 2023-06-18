package eg8141a5

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

	"github.com/abihf/modemexporter/modem"
	"github.com/abihf/modemexporter/utils"
)

type Modem struct {
	URL      string
	User     string
	Password string

	cookie string
	client http.Client
}

var _ modem.Modem = &Modem{}
var errNeedAuth = fmt.Errorf("need authentication")

var Info = &modem.Info{
	Vendor: "Huawei",
	Model:  "EG8141A5",
}

func init() {
	modem.Register(Info, register)
}

func register(url, user, pass string) modem.Modem {
	return &Modem{
		URL:      url,
		User:     user,
		Password: pass,
	}
}

// Info implements modem.Modem.
func (*Modem) Info(ctx context.Context) (*modem.Info, error) {
	return Info, nil
}

func (h *Modem) auth(ctx context.Context) error {
	token, err := h.getToken(ctx)
	if err != nil {
		return fmt.Errorf("can't get token: %w", err)
	}

	var param = url.Values{}
	param.Set("UserName", h.User)
	param.Set("PassWord", base64.URLEncoding.EncodeToString([]byte(h.Password)))
	param.Set("x.X_HW_Token", token)
	request, err := http.NewRequestWithContext(ctx, "POST", h.URL+"/login.cgi", bytes.NewBufferString(param.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("cookie", "Cookie=body:Language:english:id=-1")

	res, err := h.client.Do(request)
	if err != nil {
		return fmt.Errorf("can't access login page: %w", err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("invalid status code %d", res.StatusCode)
	}
	cookies := []string{}
	for _, cookie := range res.Cookies() {
		cookies = append(cookies, cookie.Name+"="+cookie.Value)
	}
	h.cookie = strings.Join(cookies, "; ")
	return nil
}

func (h *Modem) getToken(ctx context.Context) (string, error) {
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
func (h *Modem) Stat(ctx context.Context) (*modem.Stat, error) {
	line, err := h.readStat(ctx, false)
	if err == errNeedAuth {
		fmt.Println("not authentichated, try to reauth")
		line, err = h.readStat(ctx, true)
	}
	if err != nil {
		return nil, err
	}
	stat := modem.Stat{}
	re := regexp.MustCompile(`LANStats\("([^"]+)","([^"]+)","([^"]+)","([^"]+)","([^"]+)"`)
	matches := re.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		stat.TxPackets += utils.Must(strconv.ParseUint(match[2], 10, 64))
		stat.TxBytes += utils.Must(strconv.ParseUint(match[3], 10, 64))
		stat.RxPackets += utils.Must(strconv.ParseUint(match[4], 10, 64))
		stat.RxBytes += utils.Must(strconv.ParseUint(match[5], 10, 64))
	}
	return &stat, nil
}

func (h *Modem) readStat(ctx context.Context, reAuth bool) (string, error) {
	if reAuth || h.cookie == "" {
		err := h.auth(ctx)
		if err != nil {
			return "", fmt.Errorf("auth error: %w", err)
		}
	}

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
		if strings.Contains(line, "top.location.replace(pageName)") {
			return "", errNeedAuth
		}
		if strings.Contains(line, "var userEthInfos") {
			return line, nil
		}
	}
	return "", fmt.Errorf("invalid body response")
}
