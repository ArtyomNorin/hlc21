package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"golang.org/x/time/rate"
	"log"
	"strconv"
	"sync/atomic"
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
	limiter    *rate.Limiter

	countReq       uint64
	cash500Cnt     uint64
	cash400Cnt     uint64
	licenses500Cnt uint64
	licenses400Cnt uint64
	dig500Cnt      uint64
	dig400Cnt      uint64
	explore500Cnt  uint64
	explore400Cnt  uint64
}

func NewClient(address, port string) *Client {
	client := new(Client)
	client.httpClient = &fasthttp.HostClient{
		Addr: address + ":" + port,
	}

	client.limiter = rate.NewLimiter(rate.Every(time.Millisecond), 1)

	return client
}

func (c *Client) doRequest(req *fasthttp.Request, res *fasthttp.Response) error {
	if err := c.limiter.Wait(context.Background()); err != nil {
		log.Fatalf("WAIT ERR: %s\n", err)
	}

	atomic.AddUint64(&c.countReq, 1)
	return c.httpClient.Do(req, res)
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

func (c *Client) PostLicenses(coin Coin) (License, error) {
	request, response := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(request)
		fasthttp.ReleaseResponse(response)
	}()

	request.SetRequestURI("/licenses")
	request.Header.SetMethod("POST")
	request.Header.SetContentType("application/json")
	request.Header.SetHost("localhost")
	request.AppendBodyString("[")
	if coin != 0 {
		request.AppendBodyString(strconv.Itoa(int(coin)))
	}
	request.AppendBodyString("]")

	license := License{}
	var licenseErr error

Loop:
	for {
		if err := c.doRequest(request, response); err != nil {
			licenseErr = errors.New("post /licenses error: " + err.Error())
			break Loop
		}

		switch response.StatusCode() {
		case fasthttp.StatusOK:
			if err := json.Unmarshal(response.Body(), &license); err != nil {
				licenseErr = errors.New("unmarshal license error: " + err.Error())
			}

			break Loop
		case fasthttp.StatusServiceUnavailable, fasthttp.StatusBadGateway, fasthttp.StatusGatewayTimeout:
			atomic.AddUint64(&c.licenses500Cnt, 1)
			log.Println(fmt.Sprintf("post /licenses 5** error: %d: %s", response.StatusCode(), string(response.Body())))
		case fasthttp.StatusTooManyRequests, fasthttp.StatusConflict:
			atomic.AddUint64(&c.licenses400Cnt, 1)
			log.Println(fmt.Sprintf("post /licenses 4** error: %d: %s", response.StatusCode(), string(response.Body())))
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

	exploreAreaOut := ExploreAreaOut{}
	var exploreErr error

Loop:
	for {
		if err := c.doRequest(request, response); err != nil {
			exploreErr = errors.New("post /explore error: " + err.Error())
			break Loop
		}

		switch response.StatusCode() {
		case fasthttp.StatusOK:
			if err := json.Unmarshal(response.Body(), &exploreAreaOut); err != nil {
				exploreErr = errors.New("marshal exploreAreaOut error: " + err.Error())
			}

			break Loop
		case fasthttp.StatusServiceUnavailable, fasthttp.StatusBadGateway, fasthttp.StatusGatewayTimeout:
			atomic.AddUint64(&c.explore500Cnt, 1)
			log.Println(fmt.Sprintf("post /explore 5** error: %d: %s", response.StatusCode(), string(response.Body())))
		case fasthttp.StatusTooManyRequests, fasthttp.StatusConflict:
			atomic.AddUint64(&c.explore400Cnt, 1)
			log.Println(fmt.Sprintf("post /explore 4** error: %d: %s", response.StatusCode(), string(response.Body())))
		default:
			errStr := fmt.Sprintf("post /explore unexpected error: %d: %s", response.StatusCode(), string(response.Body()))
			exploreErr = errors.New(errStr)

			break Loop
		}
	}

	return exploreAreaOut, exploreErr
}

func (c *Client) PostDig(licenseID, posX, posY, depth int) ([]Treasure, error) {
	digIn := DigIn{LicenseID: licenseID, PosX: posX, PosY: posY, Depth: depth}

	data, err := json.Marshal(digIn)
	if err != nil {
		return nil, errors.New("marshal DigIn error: " + err.Error())
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

	treasures := make([]Treasure, 0)
	var digErr error

Loop:
	for {
		if err := c.doRequest(request, response); err != nil {
			digErr = errors.New("post /dig error: " + err.Error())
			break Loop
		}

		switch response.StatusCode() {
		case fasthttp.StatusOK:
			if err := json.Unmarshal(response.Body(), &treasures); err != nil {
				digErr = errors.New("marshal Treasures error: " + err.Error())
			}

			break Loop
		case fasthttp.StatusNotFound:
			digErr = NoTreasureErr
			break Loop
		case fasthttp.StatusServiceUnavailable, fasthttp.StatusBadGateway, fasthttp.StatusGatewayTimeout:
			atomic.AddUint64(&c.dig500Cnt, 1)
			log.Println(fmt.Sprintf("post /dig 5** error: %d: %s", response.StatusCode(), string(response.Body())))
		case fasthttp.StatusTooManyRequests, fasthttp.StatusConflict:
			atomic.AddUint64(&c.dig400Cnt, 1)
			log.Println(fmt.Sprintf("post /dig 4** error: %d: %s", response.StatusCode(), string(response.Body())))
		default:
			errStr := fmt.Sprintf("post /dig unexpected error: %d: %s", response.StatusCode(), string(response.Body()))
			digErr = errors.New(errStr)

			break Loop
		}
	}

	return treasures, digErr
}

func (c *Client) PostCash(treasure Treasure) ([]Coin, error) {
	request, response := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(request)
		fasthttp.ReleaseResponse(response)
	}()

	request.SetRequestURI("/cash")
	request.Header.SetMethod("POST")
	request.Header.SetContentType("application/json")
	request.Header.SetHost("localhost")
	request.AppendBodyString("\"" + string(treasure) + "\"")

	coins := make([]Coin, 0)
	var cashErr error

Loop:
	for {
		if err := c.doRequest(request, response); err != nil {
			cashErr = errors.New("post /cash error: " + err.Error())
			break Loop
		}

		switch response.StatusCode() {
		case fasthttp.StatusOK:
			if err := json.Unmarshal(response.Body(), &coins); err != nil {
				cashErr = errors.New("marshal Coins error: " + err.Error())
			}

			break Loop
		case fasthttp.StatusServiceUnavailable, fasthttp.StatusBadGateway, fasthttp.StatusGatewayTimeout:
			atomic.AddUint64(&c.cash500Cnt, 1)
			return nil, nil
			//log.Println(fmt.Sprintf("post /cash 5** error: %d: %s", response.StatusCode(), string(response.Body())))
		case fasthttp.StatusTooManyRequests, fasthttp.StatusConflict:
			atomic.AddUint64(&c.cash400Cnt, 1)
			log.Println(fmt.Sprintf("post /cash 4** error: %d: %s", response.StatusCode(), string(response.Body())))
		default:
			errStr := fmt.Sprintf("post /cash unexpected error: %d: %s", response.StatusCode(), string(response.Body()))
			cashErr = errors.New(errStr)

			break Loop
		}
	}

	return coins, cashErr
}
