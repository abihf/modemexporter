package eg8141a5

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
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
	Model:  "EG8141A5V5",
}

func init() {
	modem.Register(Info, factory)
}

func factory() modem.Modem {
	return &Modem{
		URL:      modem.EnvURL,
		User:     modem.EnvUser,
		Password: modem.EnvPassword,
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
	param.Set("Language", "english")
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
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("invalid status code when log in %d", res.StatusCode)
	}
	h.cookie = strings.Split(res.Header.Get("Set-Cookie"), ";")[0]
	return nil
}

func (h *Modem) getToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", h.URL+"/asp/GetRandCount.asp", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("cookie", "Cookie=body:Language:english:id=-1")
	res, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("can't fetch token: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("invalid status code when fetching token %d", res.StatusCode)
	}
	token, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("can't read token body: %w", err)
	}
	return string(token[3:]), nil
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
	re := regexp.MustCompile(`LANStats\(([^\)]+)\)`)
	matches := re.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		args := strings.Split(match[1], ",")
		stat.TxPackets += calculateArgs(args, 2, 1)
		stat.TxBytes += calculateArgs(args, 4, 3) + calculateArgs(args, 6, 5)
		stat.RxPackets += calculateArgs(args, 8, 7)
		stat.RxBytes += calculateArgs(args, 10, 9) + calculateArgs(args, 12, 11)
	}
	return &stat, nil
}

func calculateArgs(args []string, highIdx, lowIdx int) uint64 {
	high := parseIntArgs(args[highIdx])
	low := parseIntArgs(args[lowIdx])
	return high*4294967296 + low
}

func parseIntArgs(str string) uint64 {
	return utils.Must(strconv.ParseUint(strings.Trim(str, "\""), 10, 64))

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

	if !reAuth && res.StatusCode == 403 {
		return "", errNeedAuth
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("invalid status code when fetching ethinfo %d", res.StatusCode)
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
