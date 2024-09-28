package myip

import (
	"io"
	"net/http"
)

const (
	ipv4ServerEndpoint = "https://api-ipv4.ip.sb/ip"
	ipv6ServerEndpoint = "https://api-ipv6.ip.sb/ip"
)

func GetIPv4Address() (string, error) {
	return getIPAddress(ipv4ServerEndpoint)
}

func GetIPv6Address() (string, error) {
	return getIPAddress(ipv6ServerEndpoint)
}

func getIPAddress(url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Vivaldi/6.9.3447.37")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
