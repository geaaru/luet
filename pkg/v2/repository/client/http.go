/*
Copyright © 2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package client

import (
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/geaaru/luet/pkg/config"
	fileHelper "github.com/geaaru/luet/pkg/helpers/file"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	"github.com/pkg/errors"

	"github.com/cavaliercoder/grab"
	"github.com/schollz/progressbar/v3"
)

type HttpClient struct {
	Repository *config.LuetRepository
}

func NewHttpClient(r *config.LuetRepository) *HttpClient {
	return &HttpClient{Repository: r}
}

func NewGrabClient() *grab.Client {
	httpTimeout := config.LuetCfg.GetGeneral().ClientTimeout
	timeout := os.Getenv("HTTP_TIMEOUT")
	if timeout != "" {
		timeoutI, err := strconv.Atoi(timeout)
		if err == nil {
			httpTimeout = timeoutI
		}
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	insecure := os.Getenv("INSECURE")
	if insecure == "1" {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	ans := &grab.Client{
		UserAgent: "grab",
		HTTPClient: &http.Client{
			Timeout:   time.Duration(httpTimeout) * time.Second,
			Transport: transport,
		},
	}

	return ans
}

func (c *HttpClient) PrepareReq(dst, url string) (*grab.Request, error) {

	req, err := grab.NewRequest(dst, url)
	if err != nil {
		return nil, err
	}

	if val, ok := c.Repository.Authentication["token"]; ok {
		req.HTTPRequest.Header.Set("Authorization", "token "+val)
	} else if val, ok := c.Repository.Authentication["basic"]; ok {
		req.HTTPRequest.Header.Set("Authorization", "Basic "+val)
	}

	return req, err
}

func Round(input float64) float64 {
	if input < 0 {
		return math.Ceil(input - 0.5)
	}
	return math.Floor(input + 0.5)
}

func (c *HttpClient) DownloadArtifact(a *artifact.PackageArtifact) error {
	var u *url.URL = nil
	var err error
	var req *grab.Request
	var temp string

	artifactName := path.Base(a.Path)
	cacheFile := filepath.Join(
		config.LuetCfg.GetSystem().GetSystemPkgsCacheDirPath(),
		artifactName,
	)
	ok := false

	// Check if file is already in cache
	if fileHelper.Exists(cacheFile) {
		Debug("Use artifact", artifactName, "from cache.")
	} else {

		temp, err = config.LuetCfg.GetSystem().TempDir("tree")
		if err != nil {
			return err
		}
		defer os.RemoveAll(temp)

		client := NewGrabClient()

		for _, uri := range c.Repository.Urls {
			Debug("Downloading artifact", artifactName, "from", uri)

			u, err = url.Parse(uri)
			if err != nil {
				continue
			}
			u.Path = path.Join(u.Path, artifactName)

			req, err = c.PrepareReq(temp, u.String())
			if err != nil {
				continue
			}

			resp := client.Do(req)

			bar := progressbar.NewOptions64(
				resp.Size(),
				progressbar.OptionSetDescription(
					fmt.Sprintf("[cyan][%40s] - [reset]",
						a.GetPackage().HumanReadableString())),
				//filepath.Base(resp.Request.HTTPRequest.URL.RequestURI()))),
				//progressbar.OptionSetRenderBlankState(true),
				progressbar.OptionEnableColorCodes(config.LuetCfg.GetLogging().Color),
				progressbar.OptionClearOnFinish(),
				progressbar.OptionShowBytes(true),
				progressbar.OptionShowCount(),
				progressbar.OptionSetPredictTime(true),
				progressbar.OptionFullWidth(),
				/*
					progressbar.OptionSetTheme(progressbar.Theme{
						Saucer:        "[white]=[reset]",
						SaucerHead:    "[white]>[reset]",
						SaucerPadding: " ",
						BarStart:      "[",
						BarEnd:        "]",
					})
				*/
			)

			// start download loop
			t := time.NewTicker(500 * time.Millisecond)
			defer t.Stop()

		download_loop:

			for {
				select {
				case <-t.C:
					bar.Set64(resp.BytesComplete())

				case <-resp.Done:

					//bar.Reset()
					bar.Finish()
					// download is complete
					break download_loop
				}
			}

			if err = resp.Err(); err != nil {
				continue
			}

			if err != nil {
				continue
			}

			//bar.Reset()
			//bar.Finish()

			Debug("\nDownloaded", artifactName, "of",
				fmt.Sprintf("%.2f", (float64(resp.BytesComplete())/1000)/1000), "MB (",
				fmt.Sprintf("%.2f", (float64(resp.BytesPerSecond())/1024)/1024), "MiB/s )")

			Debug("\nCopying file ", filepath.Join(temp, artifactName), "to", cacheFile)
			err = fileHelper.CopyFile(filepath.Join(temp, artifactName), cacheFile)
			if err != nil {
				return errors.Wrap(err,
					fmt.Sprintf("Copying file %s to %s", filepath.Join(temp, artifactName),
						cacheFile))
			}

			ok = true
			break
		}

		if !ok {
			return err
		}

	}

	a.CachePath = cacheFile
	return nil
}

func (c *HttpClient) DownloadFile(name string) (string, error) {
	var file *os.File = nil
	var u *url.URL = nil
	var err error
	var req *grab.Request
	var temp string

	ok := false

	temp, err = config.LuetCfg.GetSystem().TempDir("tree")
	if err != nil {
		return "", err
	}

	client := NewGrabClient()

	for _, uri := range c.Repository.Urls {

		file, err = config.LuetCfg.GetSystem().TempFile("HttpClient")
		if err != nil {
			continue
		}

		u, err = url.Parse(uri)
		if err != nil {
			continue
		}
		u.Path = path.Join(u.Path, name)

		Debug("Downloading", u.String())

		req, err = c.PrepareReq(temp, u.String())
		if err != nil {
			continue
		}

		resp := client.Do(req)
		if err = resp.Err(); err != nil {
			continue
		}

		Debug("Downloaded", filepath.Base(resp.Filename), "of",
			fmt.Sprintf("%.2f", (float64(resp.BytesComplete())/1000)/1000), "MB (",
			fmt.Sprintf("%.2f", (float64(resp.BytesPerSecond())/1024)/1024), "MiB/s )")

		err = fileHelper.CopyFile(filepath.Join(temp, name), file.Name())
		if err != nil {
			continue
		}
		ok = true
		break
	}

	if !ok {
		return "", err
	}

	return file.Name(), err
}
