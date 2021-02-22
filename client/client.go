package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"syscall"
	"time"
)

var NoMoreActiveLicensesAllowedErr = errors.New("no more active licenses allowed")
var TooManyRequestsErr = errors.New("too many requests")
var InternalServerErr = errors.New("internal server err")
var NoTreasureErr = errors.New("no treasure")

type BadRequestError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Client struct {
	httpClient *fasthttp.HostClient
}

func NewClient(address, port string) *Client {
	client := new(Client)
	client.httpClient = &fasthttp.HostClient{
		Addr: address + ":" + port,
	}

	return client
}

func (c *Client) HealthCheck() (bool, error) {
	request, response := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(request)
		fasthttp.ReleaseResponse(response)
	}()

	request.SetRequestURI("/health-check")
	request.Header.SetMethod("GET")
	request.Header.SetContentType("application/json")
	request.Header.SetHost("localhost")

	if err := c.httpClient.Do(request, response); err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			return false, nil
		}

		return false, errors.New("healthcheck error: " + err.Error())
	}

	if response.StatusCode() == 200 {
		return true, nil
	}

	return false, nil
}

func (c *Client) PostLicenses() (License, error) {
	request, response := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(request)
		fasthttp.ReleaseResponse(response)
	}()

	request.SetRequestURI("/licenses")
	request.Header.SetMethod("POST")
	request.Header.SetContentType("application/json")
	request.Header.SetHost("localhost")
	request.AppendBodyString("[]")

	license := License{}
	var licenseErr error

Loop:
	for {
		if err := c.httpClient.Do(request, response); err != nil {
			licenseErr = errors.New("post /licenses error: " + err.Error())
			break Loop
		}

		switch response.StatusCode() {
		case fasthttp.StatusOK:
			if err := json.Unmarshal(response.Body(), &license); err != nil {
				licenseErr = errors.New("unmarshal license error: " + err.Error())
			}

			break Loop
		case fasthttp.StatusServiceUnavailable, fasthttp.StatusBadGateway,
			fasthttp.StatusGatewayTimeout, fasthttp.StatusTooManyRequests, fasthttp.StatusConflict:
			log.Println("license sleep")
			time.Sleep(time.Microsecond * 200)
		default:
			errStr := fmt.Sprintf("post /licenses unexpected error: %d: %s", response.StatusCode(), string(response.Body()))
			licenseErr = errors.New(errStr)

			break Loop
		}
	}

	return license, licenseErr
}

func (c *Client) PostExplore(posX, posY, sizeX, sizeY int) (ExploreAreaOut, error) {
	exploreAreaIn := ExploreAreaIn{PosX: posX, PosY: posY, SizeX: sizeX, SizeY: sizeY}

	data, err := json.Marshal(exploreAreaIn)
	if err != nil {
		return ExploreAreaOut{}, errors.New("marshal exploreAreaIn error: " + err.Error())
	}

	request, response := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(request)
		fasthttp.ReleaseResponse(response)
	}()

	request.SetRequestURI("/explore")
	request.Header.SetMethod("POST")
	request.Header.SetContentType("application/json")
	request.Header.SetHost("localhost")
	request.AppendBody(data)

	if err := c.httpClient.Do(request, response); err != nil {
		return ExploreAreaOut{}, errors.New("post /explore error: " + err.Error())
	}

	switch response.StatusCode() {
	case fasthttp.StatusOK:
		exploreAreaOut := ExploreAreaOut{}
		if err := json.Unmarshal(response.Body(), &exploreAreaOut); err != nil {
			return ExploreAreaOut{}, errors.New("marshal exploreAreaOut error: " + err.Error())
		}

		return exploreAreaOut, nil
	case fasthttp.StatusTooManyRequests:
		return ExploreAreaOut{}, TooManyRequestsErr
	case fasthttp.StatusServiceUnavailable, fasthttp.StatusBadGateway, fasthttp.StatusGatewayTimeout:
		return ExploreAreaOut{}, InternalServerErr
	default:
		errStr := fmt.Sprintf("post /explore unexpected error: %d: %s", response.StatusCode(), string(response.Body()))
		return ExploreAreaOut{}, errors.New(errStr)
	}
}

func (c *Client) PostDig(licenseID, posX, posY, depth int) (Treasures, error) {
	digIn := DigIn{LicenseID: licenseID, PosX: posX, PosY: posY, Depth: depth}

	data, err := json.Marshal(digIn)
	if err != nil {
		return Treasures{}, errors.New("marshal DigIn error: " + err.Error())
	}

	request, response := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(request)
		fasthttp.ReleaseResponse(response)
	}()

	request.SetRequestURI("/dig")
	request.Header.SetMethod("POST")
	request.Header.SetContentType("application/json")
	request.Header.SetHost("localhost")
	request.AppendBody(data)

	if err := c.httpClient.Do(request, response); err != nil {
		return Treasures{}, errors.New("post /dig error: " + err.Error())
	}

	switch response.StatusCode() {
	case fasthttp.StatusOK:
		treasures := Treasures{}
		if err := json.Unmarshal(response.Body(), &treasures); err != nil {
			return Treasures{}, errors.New("marshal Treasures error: " + err.Error())
		}

		return treasures, nil
	case fasthttp.StatusNotFound:
		return Treasures{}, NoTreasureErr
	case fasthttp.StatusTooManyRequests:
		return Treasures{}, TooManyRequestsErr
	case fasthttp.StatusServiceUnavailable, fasthttp.StatusBadGateway, fasthttp.StatusGatewayTimeout:
		return Treasures{}, InternalServerErr
	default:
		errStr := fmt.Sprintf("post /dig unexpected error: %d: %s", response.StatusCode(), string(response.Body()))
		return Treasures{}, errors.New(errStr)
	}
}

func (c *Client) PostCash(treasure string) error {
	request, response := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(request)
		fasthttp.ReleaseResponse(response)
	}()

	request.SetRequestURI("/cash")
	request.Header.SetMethod("POST")
	request.Header.SetContentType("application/json")
	request.Header.SetHost("localhost")
	request.AppendBodyString("\"" + treasure + "\"")

	var cashErr error

Loop:
	for {
		if err := c.httpClient.Do(request, response); err != nil {
			cashErr = errors.New("post /cash error: " + err.Error())
			break Loop
		}

		switch response.StatusCode() {
		case fasthttp.StatusOK:
			break Loop
		case fasthttp.StatusServiceUnavailable, fasthttp.StatusBadGateway,
			fasthttp.StatusGatewayTimeout, fasthttp.StatusTooManyRequests, fasthttp.StatusConflict:
			log.Println("cash sleep")
			time.Sleep(time.Microsecond * 200)
		default:
			errStr := fmt.Sprintf("post /cash unexpected error: %d: %s", response.StatusCode(), string(response.Body()))
			cashErr = errors.New(errStr)

			break Loop
		}
	}

	return cashErr
}
