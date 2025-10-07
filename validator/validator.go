package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"validator/config"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type Cfg struct {
	Path         string `json:"path"`
	Headless     bool   `json:"headless"`
	CookieName   string `json:"cookie_name"`
	CookieValue  string `json:"cookie_value"`
	CookieDomain string `json:"cookie_domain"`
}

var cfg *Cfg

func main() {

	if len(os.Args) < 2 {
		os.Exit(1)
	}
	link := os.Args[1]

	cfg = &Cfg{}

	if err := config.Load("validator.json", cfg); err != nil {
		cfg = &Cfg{
			Path:         "/usr/bin/chromium",
			Headless:     true,
			CookieName:   "CF_Authorization",
			CookieValue:  "VALID-CLOUDFLAREACCESS.COM-TOKEN",
			CookieDomain: "ORGANIZATION.cloudflareaccess.com",
		}
		if err := config.Save("validator.json", cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(cfg.Path), chromedp.Flag("headless", cfg.Headless))
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var html string

	cookieName := cfg.CookieName
	cookieValue := cfg.CookieValue
	cookieDomain := cfg.CookieDomain

	if err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			expires := cdp.TimeSinceEpoch(time.Now().Add(30 * time.Minute))
			return network.SetCookie(cookieName, cookieValue).
				WithDomain(cookieDomain).
				WithPath("/").
				WithExpires(&expires).
				Do(ctx)
		})); err != nil {
		fmt.Fprintln(os.Stderr, "Chromedp run cookie error:", err)
		os.Exit(1)
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate(link),
		chromedp.WaitVisible(`#code-form`, chromedp.ByID),
		chromedp.Click(`button[name="action"][value="approve"]`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
		chromedp.OuterHTML("html", &html),
	); err != nil {
		fmt.Fprintln(os.Stderr, "Chromedp run approve error:", err)
		os.Exit(1)
	}

	if strings.Contains(html, `<div class="Success">`) {
		fmt.Fprintln(os.Stdout, "Approve ok!")
		os.Exit(0)
	} else {
		fmt.Fprintln(os.Stdout, "Approve failed!")
		os.Exit(2)
	}

}
