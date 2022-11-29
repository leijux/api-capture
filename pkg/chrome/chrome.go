package chrome

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"changeme/pkg/config"

	"github.com/alphadose/haxmap"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func NewChromedpCtx(ctx context.Context, execPath string) (context.Context, context.CancelFunc) {
	if execPath != "" {
		ctx, _ = chromedp.NewExecAllocator(ctx,
			chromedp.NoSandbox,
			chromedp.NoDefaultBrowserCheck,
			chromedp.NoFirstRun,
			chromedp.ExecPath(execPath),
			chromedp.WindowSize(1920, 1080),
		)
	} else {
		ctx, _ = chromedp.NewExecAllocator(ctx,
			chromedp.NoSandbox,
			chromedp.NoDefaultBrowserCheck,
			chromedp.NoFirstRun,
			chromedp.WindowSize(1920, 1080),
		)
	}

	ctx, cancel := chromedp.NewContext(
		ctx,
		chromedp.WithLogf(log.Printf),
	)
	return ctx, cancel
}

func RunChromedp(ctx context.Context, cfg *config.Config, controlSignal chan struct{}, hm *haxmap.Map[string, *RequestInfo]) {
	ctx, cancel := NewChromedpCtx(ctx, cfg.BrowserPath)
	defer cancel()

	listenForNetworkEvent(ctx, hm)
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return nil
		}),
		network.Enable(),
		chromedp.Navigate(cfg.URL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			<-controlSignal
			cookies := getCookies(ctx)
			hm.ForEach(func(k string, v *RequestInfo) bool {
				if v.Validator.StatusCode == 0 {
					hm.Del(k)
				} else {
					v.Header.Cookies = cookies
					body, err := network.GetResponseBody(v.RequestID).Do(ctx)
					if err != nil {
						log.Ctx(ctx).Warn().Err(err).Send()
					}
					v.ResponseBody = body
				}
				return true
			})
			return nil
		}),
	)
	if err != nil {
		if len(controlSignal) == 1 {
			<-controlSignal
		}
		log.Ctx(ctx).Warn().Err(err).Send()
	}
}

func getCookies(ctx context.Context) string {
	cookies, err := network.GetCookies().Do(ctx)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Send()
	}
	s := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		s = append(s, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(s, "; ")
}

// 监听
func listenForNetworkEvent(ctx context.Context, hm *haxmap.Map[string, *RequestInfo]) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSent:
			req := ev.Request
			log.Ctx(ctx).Debug().
				Str("发现请求", req.URL).
				Str("请求方法", req.Method).
				Send()

			switch req.Method {
			case http.MethodGet:
				requestInfo := eventRequestGET(req)
				if requestInfo != nil {
					hm.Set(req.URL, requestInfo)
				}
			case http.MethodPost:
				if contentTypeIsJson(req.Headers) {
					requestInfo := eventRequestPOST(req)
					hm.Set(req.URL, requestInfo)
				}
			}

		case *network.EventResponseReceived:
			resp := ev.Response
			if contentTypeIsJson(resp.Headers) {
				if requestInfo, ok := hm.Get(resp.URL); ok {
					requestInfo.Validator.StatusCode = resp.Status
					requestInfo.RequestID = ev.RequestID

					runtime.EventsEmit(ctx, "capture requestInfo", requestInfo)

					log.Ctx(ctx).Info().
						Str("捕获到请求", requestInfo.Url).
						Str("请求方法", requestInfo.Method).
						Send()

				}
			}
		}
	})
}

func contentTypeIsJson(headers network.Headers) bool {
	//log.Debug().Interface("headers", headers).Send()
	if len(headers) == 0 {
		return false
	}
	if contentTyp, ok := headers["content-type"]; ok {
		if strings.Contains(contentTyp.(string), "application/json") {
			return true
		}
	} else if contentTyp, ok := headers["Content-Type"]; ok {
		if strings.Contains(contentTyp.(string), "application/json") {
			return true
		}
	}
	return false
}

func toHeader(headers network.Headers) Header {
	var h Header

	if authorization, ok := headers["Authorization"]; ok {
		h.Authorization = cast.ToString(authorization)
	}

	if contentType, ok := headers["content-type"]; ok {
		h.ContentType = cast.ToString(contentType)
	} else if contentType, ok := headers["Content-Type"]; ok {
		h.ContentType = cast.ToString(contentType)
	}

	if userAgent, ok := headers["User-Agent"]; ok {
		h.UserAgent = cast.ToString(userAgent)
	}
	return h
}

func eventRequestGET(req *network.Request) *RequestInfo {
	var parseUrl, _ = url.Parse(req.URL)

	if parseUrl.Scheme == "https" || parseUrl.Scheme == "http" {
		return &RequestInfo{
			Url:    fmt.Sprintf("%s://%s%s", parseUrl.Scheme, parseUrl.Host, parseUrl.Path),
			Method: http.MethodGet,
			Header: toHeader(req.Headers),
			Data: Data{
				Params: parseUrl.RawQuery,
			},
		}
	}
	return nil
}

func eventRequestPOST(req *network.Request) *RequestInfo {
	r := &RequestInfo{
		Url:    req.URL,
		Method: http.MethodPost,
		Header: toHeader(req.Headers),
	}
	if gjson.Valid(req.PostData) {
		r.Data.Payload = req.PostData
	}
	return r
}

type YamlData struct {
	RequestInfo RequestInfo `yaml:"api"`
}

// RequestInfo 请求数据
type RequestInfo struct {
	Url       string    `yaml:"url"`
	Method    string    `yaml:"method"`
	Header    Header    `yaml:"header,omitempty"`
	Data      Data      `yaml:"data,omitempty"`
	Validator Validator `yaml:"validator"`

	RequestID    network.RequestID `yaml:"-"`
	ResponseBody []byte            `yaml:"-"`
}

// Header 请求头数据
type Header struct {
	ContentType   string `yaml:"Content-Type,omitempty"`
	Authorization string `yaml:"Authorization,omitempty"`
	Cookies       string `yaml:"Cookie,omitempty"`
	UserAgent     string `yaml:"User-Agent,omitempty"`
}

// Data 请求参数/请求体
type Data struct {
	Params  string `yaml:"params,omitempty"`
	Payload string `yaml:"payload,omitempty"`
}

// Validator 验证
type Validator struct {
	StatusCode int64 `yaml:"status_code"`
}
